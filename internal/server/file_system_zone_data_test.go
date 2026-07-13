package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"github.com/omnihance/omnihance-a3-agent/internal/config"
	"github.com/omnihance/omnihance-a3-agent/internal/constants"
	"github.com/omnihance/omnihance-a3-agent/internal/logger"
	"github.com/omnihance/omnihance-a3-agent/internal/services"
	"github.com/omnihance/omnihance-a3-agent/internal/utils"
	"github.com/project-agonyl/agonyl-utils-go/npcskillfile"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
)

func TestZoneDataFileReadAndUpdateCreatesRevision(t *testing.T) {
	server, userID, filePath, original := newZoneDataTestServer(t)

	getRequest := zoneDataRequest(http.MethodGet, filePath, nil, userID, constants.RoleAdmin)
	getResponse := httptest.NewRecorder()
	server.handleZoneDataFile(getResponse, getRequest)
	require.Equal(t, http.StatusOK, getResponse.Code, getResponse.Body.String())

	var decoded services.ZoneDataFile
	require.NoError(t, json.NewDecoder(getResponse.Body).Decode(&decoded))
	require.Equal(t, services.ZoneDataFormatNPCSkill, decoded.Format)
	require.Equal(t, utils.CalculateFileHash(original), decoded.SourceHash)

	body := bytes.NewBufferString(`{"source_hash":"` + decoded.SourceHash + `","operations":[{"scope":"row","row":0,"field":"effect_code","value":48879}]}`)
	putRequest := zoneDataRequest(http.MethodPut, filePath, body, userID, constants.RoleAdmin)
	putResponse := httptest.NewRecorder()
	server.handleUpdateZoneDataFile(putResponse, putRequest)
	require.Equal(t, http.StatusOK, putResponse.Code, putResponse.Body.String())

	updated, err := os.ReadFile(filePath)
	require.NoError(t, err)
	require.Equal(t, original[:14], updated[:14])
	require.Equal(t, []byte{0xef, 0xbe}, updated[14:16])
	require.Equal(t, original[16:], updated[16:])

	revision, err := server.internalDB.GetLastCompletedFileRevision(utils.GenerateMD5Hash(filePath))
	require.NoError(t, err)
	require.NotNil(t, revision)
	revisionData, err := os.ReadFile(revision.RevisionPath)
	require.NoError(t, err)
	require.Equal(t, original, revisionData)
}

func TestZoneDataFileUpdateRejectsStaleHash(t *testing.T) {
	server, userID, filePath, original := newZoneDataTestServer(t)
	body := bytes.NewBufferString(`{"source_hash":"stale","operations":[{"scope":"row","row":0,"field":"effect_code","value":1}]}`)
	request := zoneDataRequest(http.MethodPut, filePath, body, userID, constants.RoleAdmin)
	response := httptest.NewRecorder()

	server.handleUpdateZoneDataFile(response, request)

	require.Equal(t, http.StatusConflict, response.Code, response.Body.String())
	current, err := os.ReadFile(filePath)
	require.NoError(t, err)
	require.Equal(t, original, current)
	revision, err := server.internalDB.GetLastCompletedFileRevision(utils.GenerateMD5Hash(filePath))
	require.NoError(t, err)
	require.Nil(t, revision)
}

func TestZoneDataFileUpdateRequiresEditPermission(t *testing.T) {
	server, userID, filePath, _ := newZoneDataTestServer(t)
	body := bytes.NewBufferString(`{"source_hash":"hash","operations":[{"scope":"row","row":0,"field":"effect_code","value":1}]}`)
	request := zoneDataRequest(http.MethodPut, filePath, body, userID, constants.RoleUser)
	response := httptest.NewRecorder()

	server.handleUpdateZoneDataFile(response, request)

	require.Equal(t, http.StatusForbidden, response.Code, response.Body.String())
}

func newZoneDataTestServer(t *testing.T) (*Server, int64, string, []byte) {
	t.Helper()

	internalDB := newTestInternalDB(t)
	user, err := internalDB.CreateUser("zone-data-admin@example.com", "password", constants.RoleAdmin, nil)
	require.NoError(t, err)
	root := t.TempDir()
	_, err = internalDB.CreateSetting(constants.SettingKeyZoneServerPath, root, &user.ID)
	require.NoError(t, err)

	filePath := filepath.Join(root, "ZoneData", "npc", "NPCSkill")
	require.NoError(t, os.MkdirAll(filepath.Dir(filePath), 0755))
	original := make([]byte, npcskillfile.RecordSize)
	for index := range original {
		original[index] = byte(index + 1)
	}
	require.NoError(t, os.WriteFile(filePath, original, 0600))

	log := logger.NewZerologLogger(zerolog.Nop(), "test", zerolog.Disabled)
	fileEditor := services.NewFileEditorService(log)
	server := &Server{
		cfg: &config.EnvVars{CookieSecret: "test-secret", RevisionsDirectory: t.TempDir()},
		log: log, internalDB: internalDB, fileEditor: fileEditor,
	}
	server.zoneDataService = services.NewZoneDataService(internalDB, fileEditor)
	return server, user.ID, filePath, original
}

func zoneDataRequest(method string, path string, body *bytes.Buffer, userID int64, role string) *http.Request {
	var request *http.Request
	if body == nil {
		request = httptest.NewRequest(method, "/api/file-tree/zone-data-file?path="+url.QueryEscape(path), nil)
	} else {
		request = httptest.NewRequest(method, "/api/file-tree/zone-data-file?path="+url.QueryEscape(path), body)
	}
	ctx := utils.SetUserIdInContext(request.Context(), userID)
	ctx = utils.SetUserRolesInContext(ctx, []string{role})
	return request.WithContext(ctx)
}
