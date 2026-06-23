package server

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/omnihance/omnihance-a3-agent/internal/config"
	"github.com/omnihance/omnihance-a3-agent/internal/constants"
	"github.com/omnihance/omnihance-a3-agent/internal/services"
	"github.com/omnihance/omnihance-a3-agent/internal/utils"
)

func TestCreateFileUploadRejectsViewerAndRootDestination(t *testing.T) {
	server := newFileUploadTestServer(t)

	req := fileUploadRequest(t, http.MethodPost, "/api/file-tree/uploads", CreateFileUploadRequest{
		DestinationPath: t.TempDir(),
		ChunkSize:       4,
		Files: []CreateFileUploadRequestFile{
			{ClientFileID: "file-1", RelativePath: "server.txt", Size: 4},
		},
	}, constants.RoleUser)
	rr := httptest.NewRecorder()
	server.handleCreateFileUpload(rr, req)
	require.Equal(t, http.StatusForbidden, rr.Code)

	req = fileUploadRequest(t, http.MethodPost, "/api/file-tree/uploads", CreateFileUploadRequest{
		DestinationPath: "",
		ChunkSize:       4,
		Files: []CreateFileUploadRequestFile{
			{ClientFileID: "file-1", RelativePath: "server.txt", Size: 4},
		},
	}, constants.RoleAdmin)
	rr = httptest.NewRecorder()
	server.handleCreateFileUpload(rr, req)
	require.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestCreateFileUploadRejectsOversizedChunkSize(t *testing.T) {
	manager := newFileUploadTestManager(t)
	destination := t.TempDir()

	_, err := manager.CreateSession(CreateFileUploadRequest{
		DestinationPath: destination,
		ChunkSize:       fileUploadMaxChunkSize + 1,
		Files: []CreateFileUploadRequestFile{
			{ClientFileID: "file-1", RelativePath: "server.txt", Size: 1},
		},
	})

	require.Error(t, err)
	var uploadErr *fileUploadHTTPError
	require.ErrorAs(t, err, &uploadErr)
	require.Equal(t, http.StatusBadRequest, uploadErr.status)
}

func TestFileUploadManagerStartStopCanRestart(t *testing.T) {
	manager := newFileUploadTestManager(t)

	require.NoError(t, manager.Start())
	manager.Stop()
	require.NoError(t, manager.Start())
	manager.Stop()
}

func TestFileUploadSessionReservesDuplicateNamesAcrossTabs(t *testing.T) {
	manager := newFileUploadTestManager(t)
	destination := t.TempDir()
	content := []byte("large upload body")

	first := createUploadSessionForTest(t, manager, destination, "patch.bin", int64(len(content)), 5)
	second := createUploadSessionForTest(t, manager, destination, "patch.bin", int64(len(content)), 5)

	require.Equal(t, "patch.bin", first.Files[0].ResolvedRelativePath)
	require.Equal(t, "patch (copy).bin", second.Files[0].ResolvedRelativePath)

	completeUploadForTest(t, manager, first, content)
	completeUploadForTest(t, manager, second, content)

	require.FileExists(t, filepath.Join(destination, "patch.bin"))
	require.FileExists(t, filepath.Join(destination, "patch (copy).bin"))
}

func TestFileUploadDirectoryTopLevelConflictUsesCopyName(t *testing.T) {
	manager := newFileUploadTestManager(t)
	destination := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(destination, "Data"), 0755))

	response := createUploadSessionForTest(t, manager, destination, filepath.Join("Data", "zone.txt"), 4, 4)

	require.Equal(t, filepath.Join("Data (copy)", "zone.txt"), response.Files[0].ResolvedRelativePath)
}

func TestFileUploadDirectoryTopLevelReservationAcrossSessions(t *testing.T) {
	manager := newFileUploadTestManager(t)
	destination := t.TempDir()

	first := createUploadSessionForTest(t, manager, destination, filepath.Join("Data", "a.txt"), 1, 1)
	second := createUploadSessionForTest(t, manager, destination, filepath.Join("Data", "b.txt"), 1, 1)

	require.Equal(t, filepath.Join("Data", "a.txt"), first.Files[0].ResolvedRelativePath)
	require.Equal(t, filepath.Join("Data (copy)", "b.txt"), second.Files[0].ResolvedRelativePath)
}

func TestFileUploadCopyNameUsesCopyTwoAfterCopyExists(t *testing.T) {
	manager := newFileUploadTestManager(t)
	destination := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(destination, "server.txt"), []byte("original"), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(destination, "server (copy).txt"), []byte("copy"), 0600))

	response := createUploadSessionForTest(t, manager, destination, "server.txt", 1, 1)

	require.Equal(t, "server (copy 2).txt", response.Files[0].ResolvedRelativePath)
}

func TestFileUploadConcurrentSessionsRegisterAllTempRoots(t *testing.T) {
	manager := newFileUploadTestManager(t)
	destination := t.TempDir()
	sessionCount := 12
	errors := make(chan error, sessionCount)
	var wg sync.WaitGroup

	for index := 0; index < sessionCount; index++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()

			_, err := manager.CreateSession(CreateFileUploadRequest{
				DestinationPath: destination,
				ChunkSize:       1,
				Files: []CreateFileUploadRequestFile{
					{
						ClientFileID: "file-1",
						RelativePath: "server-" + strconv.Itoa(index) + ".txt",
						Size:         1,
					},
				},
			})
			errors <- err
		}(index)
	}

	wg.Wait()
	close(errors)
	for err := range errors {
		require.NoError(t, err)
	}

	tempRoots, err := manager.readRegisteredTempRoots()
	require.NoError(t, err)
	require.Len(t, tempRoots, sessionCount)
}

func TestFileUploadCancelRemovesTempAndReleasesReservation(t *testing.T) {
	manager := newFileUploadTestManager(t)
	destination := t.TempDir()
	content := []byte("cancel me")
	response := createUploadSessionForTest(t, manager, destination, "server.txt", int64(len(content)), 4)

	uploadChunksForTest(t, manager, response, content)
	tempRoot := manager.sessions[response.UploadID].TempRoot
	require.DirExists(t, tempRoot)

	require.NoError(t, manager.CancelSession(response.UploadID))
	require.NoDirExists(t, tempRoot)

	next := createUploadSessionForTest(t, manager, destination, "server.txt", int64(len(content)), 4)
	require.Equal(t, "server.txt", next.Files[0].ResolvedRelativePath)
}

func TestFileUploadExpiredCleanupRemovesTempAndReleasesReservation(t *testing.T) {
	manager := newFileUploadTestManager(t)
	destination := t.TempDir()
	response := createUploadSessionForTest(t, manager, destination, "server.txt", 4, 4)
	tempRoot := manager.sessions[response.UploadID].TempRoot
	require.DirExists(t, tempRoot)

	manager.sessions[response.UploadID].LastSeenAt = time.Now().Add(-fileUploadSessionTTL - time.Second)
	manager.cleanupExpiredSessions(time.Now())

	require.NoDirExists(t, tempRoot)
	_, ok := manager.sessions[response.UploadID]
	require.False(t, ok)

	next := createUploadSessionForTest(t, manager, destination, "server.txt", 4, 4)
	require.Equal(t, "server.txt", next.Files[0].ResolvedRelativePath)
}

func TestFileUploadHashMismatchRemovesTempAndReleasesReservation(t *testing.T) {
	manager := newFileUploadTestManager(t)
	destination := t.TempDir()
	content := []byte("corrupt check")
	response := createUploadSessionForTest(t, manager, destination, "server.txt", int64(len(content)), 4)

	uploadChunksForTest(t, manager, response, content)
	tempPath := manager.sessions[response.UploadID].Files[response.Files[0].FileID].TempPath
	require.FileExists(t, tempPath)

	_, err := manager.CompleteFile(response.UploadID, response.Files[0].FileID, strings.Repeat("0", 64))
	require.Error(t, err)
	var uploadErr *fileUploadHTTPError
	require.ErrorAs(t, err, &uploadErr)
	require.Equal(t, http.StatusConflict, uploadErr.status)
	require.NoFileExists(t, tempPath)

	next := createUploadSessionForTest(t, manager, destination, "server.txt", int64(len(content)), 4)
	require.Equal(t, "server.txt", next.Files[0].ResolvedRelativePath)
}

func TestFileUploadCompleteRenamesWhenFinalPathAppearsExternally(t *testing.T) {
	manager := newFileUploadTestManager(t)
	destination := t.TempDir()
	content := []byte("server data")
	response := createUploadSessionForTest(t, manager, destination, "server.txt", int64(len(content)), 4)

	uploadChunksForTest(t, manager, response, content)
	require.NoError(t, os.WriteFile(filepath.Join(destination, "server.txt"), []byte("external"), 0600))

	complete := completeUploadForTest(t, manager, response, content)

	require.Equal(t, filepath.Join(destination, "server (copy).txt"), complete.FinalPath)
	require.FileExists(t, filepath.Join(destination, "server.txt"))
	require.FileExists(t, filepath.Join(destination, "server (copy).txt"))
}

func newFileUploadTestServer(t *testing.T) *Server {
	t.Helper()

	return &Server{
		cfg:        &config.EnvVars{CookieSecret: "test-secret", RevisionsDirectory: t.TempDir()},
		fileEditor: services.NewFileEditorService(nil),
	}
}

func newFileUploadTestManager(t *testing.T) *fileUploadManager {
	t.Helper()

	return newFileUploadManager(t.TempDir(), services.NewFileEditorService(nil), nil)
}

func fileUploadRequest(t *testing.T, method string, target string, body any, role string) *http.Request {
	t.Helper()

	var payload bytes.Buffer
	require.NoError(t, json.NewEncoder(&payload).Encode(body))
	req := httptest.NewRequest(method, target, &payload)
	ctx := utils.SetUserRolesInContext(req.Context(), []string{role})
	return req.WithContext(ctx)
}

func createUploadSessionForTest(t *testing.T, manager *fileUploadManager, destination string, relativePath string, size int64, chunkSize int64) *CreateFileUploadResponse {
	t.Helper()

	response, err := manager.CreateSession(CreateFileUploadRequest{
		DestinationPath: destination,
		ChunkSize:       chunkSize,
		Files: []CreateFileUploadRequestFile{
			{ClientFileID: "file-1", RelativePath: relativePath, Size: size},
		},
	})
	require.NoError(t, err)
	require.Len(t, response.Files, 1)
	return response
}

func uploadChunksForTest(t *testing.T, manager *fileUploadManager, response *CreateFileUploadResponse, content []byte) {
	t.Helper()

	file := response.Files[0]
	for chunkIndex := 0; chunkIndex < file.TotalChunks; chunkIndex++ {
		start := int64(chunkIndex) * file.ChunkSize
		end := start + file.ChunkSize
		if end > int64(len(content)) {
			end = int64(len(content))
		}

		_, err := manager.UploadChunk(response.UploadID, file.FileID, chunkIndex, bytes.NewReader(content[start:end]))
		require.NoError(t, err)
	}
}

func completeUploadForTest(t *testing.T, manager *fileUploadManager, response *CreateFileUploadResponse, content []byte) *CompleteFileUploadResponse {
	t.Helper()

	uploadChunksForTest(t, manager, response, content)
	hash := sha256.Sum256(content)
	complete, err := manager.CompleteFile(response.UploadID, response.Files[0].FileID, hex.EncodeToString(hash[:]))
	require.NoError(t, err)
	return complete
}
