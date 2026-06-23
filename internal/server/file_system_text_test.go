package server

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"

	"github.com/omnihance/omnihance-a3-agent/internal/config"
	"github.com/omnihance/omnihance-a3-agent/internal/constants"
	"github.com/omnihance/omnihance-a3-agent/internal/logger"
	"github.com/omnihance/omnihance-a3-agent/internal/services"
	"github.com/omnihance/omnihance-a3-agent/internal/utils"
)

func TestUpdateTextFileAllowsEmptyContent(t *testing.T) {
	server, userID := newTextFileUpdateTestServer(t)
	filePath := writeTextUpdateTestFile(t, t.TempDir(), ".env", []byte("PORT=8080\n"))

	req := textFileUpdateRequest(http.MethodPut, "/api/file-tree/text-file?path="+url.QueryEscape(filePath), bytes.NewBufferString(`{"content":""}`), userID)
	rr := httptest.NewRecorder()
	server.handleUpdateTextFile(rr, req)

	require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())
	content, err := os.ReadFile(filePath)
	require.NoError(t, err)
	require.Empty(t, content)
}

func TestUpdateTextFilePreservesPermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows does not expose POSIX file mode bits consistently")
	}

	server, userID := newTextFileUpdateTestServer(t)
	filePath := writeTextUpdateTestFile(t, t.TempDir(), ".env", []byte("PORT=8080\n"))
	require.NoError(t, os.Chmod(filePath, 0600))

	beforeInfo, err := os.Stat(filePath)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0600), beforeInfo.Mode().Perm())

	req := textFileUpdateRequest(http.MethodPut, "/api/file-tree/text-file?path="+url.QueryEscape(filePath), bytes.NewBufferString(`{"content":"PORT=9090\n"}`), userID)
	rr := httptest.NewRecorder()
	server.handleUpdateTextFile(rr, req)

	require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())
	afterInfo, err := os.Stat(filePath)
	require.NoError(t, err)
	require.Equal(t, beforeInfo.Mode().Perm(), afterInfo.Mode().Perm())
}

func TestUpdateTextFileRejectsMissingContent(t *testing.T) {
	server, userID := newTextFileUpdateTestServer(t)
	originalContent := []byte("PORT=8080\n")
	filePath := writeTextUpdateTestFile(t, t.TempDir(), ".env.local", originalContent)

	req := textFileUpdateRequest(http.MethodPut, "/api/file-tree/text-file?path="+url.QueryEscape(filePath), bytes.NewBufferString(`{}`), userID)
	rr := httptest.NewRecorder()
	server.handleUpdateTextFile(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code, rr.Body.String())
	content, err := os.ReadFile(filePath)
	require.NoError(t, err)
	require.Equal(t, originalContent, content)
}

func newTextFileUpdateTestServer(t *testing.T) (*Server, int64) {
	t.Helper()

	internalDB := newTestInternalDB(t)
	user, err := internalDB.CreateUser("text-file-admin@example.com", "password", constants.RoleAdmin, nil)
	require.NoError(t, err)

	log := logger.NewZerologLogger(zerolog.Nop(), "test", zerolog.Disabled)
	return &Server{
		cfg: &config.EnvVars{
			CookieSecret:       "test-secret",
			RevisionsDirectory: t.TempDir(),
		},
		log:        log,
		internalDB: internalDB,
		fileEditor: services.NewFileEditorService(log),
	}, user.ID
}

func textFileUpdateRequest(method string, target string, body *bytes.Buffer, userID int64) *http.Request {
	req := httptest.NewRequest(method, target, strings.NewReader(body.String()))
	ctx := req.Context()
	ctx = utils.SetUserIdInContext(ctx, userID)
	ctx = utils.SetUserRolesInContext(ctx, []string{constants.RoleAdmin})
	return req.WithContext(ctx)
}

func writeTextUpdateTestFile(t *testing.T, dir string, name string, content []byte) string {
	t.Helper()

	path := filepath.Join(dir, name)
	require.NoError(t, os.WriteFile(path, content, 0600))
	return path
}
