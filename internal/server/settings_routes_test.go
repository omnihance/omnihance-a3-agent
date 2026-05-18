package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/omnihance/omnihance-a3-agent/internal/constants"
	"github.com/omnihance/omnihance-a3-agent/internal/db"
	"github.com/omnihance/omnihance-a3-agent/internal/utils"
	"github.com/stretchr/testify/require"
)

func TestHandleListSettingsReturnsSupportedSettingsAndDefinitions(t *testing.T) {
	internalDB := newTestInternalDB(t)
	_, err := internalDB.CreateSetting(constants.SettingKeyDBHost, "127.0.0.1", nil)
	require.NoError(t, err)
	_, err = internalDB.CreateSetting("DB_NAME", "ASD", nil)
	require.NoError(t, err)

	server := &Server{internalDB: internalDB}
	req := settingsRequest(http.MethodGet, "/api/settings", nil, constants.RoleAdmin)
	rr := httptest.NewRecorder()

	server.handleListSettings(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)

	var response SettingsResponse
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &response))
	require.Len(t, response.Settings, 1)
	require.Equal(t, constants.SettingKeyDBHost, response.Settings[0].Key)
	require.Len(t, response.Definitions, 4)
}

func TestHandleCreateSettingRequiresManageServer(t *testing.T) {
	internalDB := newTestInternalDB(t)
	server := &Server{internalDB: internalDB}
	body := bytes.NewBufferString(`{"key":"DB_HOST","value":"127.0.0.1"}`)
	req := settingsRequest(http.MethodPost, "/api/settings", body, constants.RoleUser)
	rr := httptest.NewRecorder()

	server.handleCreateSetting(rr, req)

	require.Equal(t, http.StatusForbidden, rr.Code)
}

func TestHandleCreateSettingRejectsUnsupportedKey(t *testing.T) {
	internalDB := newTestInternalDB(t)
	server := &Server{internalDB: internalDB}
	body := bytes.NewBufferString(`{"key":"DB_NAME","value":"ASD"}`)
	req := settingsRequest(http.MethodPost, "/api/settings", body, constants.RoleAdmin)
	rr := httptest.NewRecorder()

	server.handleCreateSetting(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestHandleCreateSettingRejectsDuplicate(t *testing.T) {
	internalDB := newTestInternalDB(t)
	createSettingsTestUser(t, internalDB)
	_, err := internalDB.CreateSetting(constants.SettingKeyDBHost, "127.0.0.1", nil)
	require.NoError(t, err)
	server := &Server{internalDB: internalDB}
	body := bytes.NewBufferString(`{"key":"DB_HOST","value":"localhost"}`)
	req := settingsRequest(http.MethodPost, "/api/settings", body, constants.RoleAdmin)
	rr := httptest.NewRecorder()

	server.handleCreateSetting(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestHandleCreateSettingValidatesValues(t *testing.T) {
	tests := []struct {
		name           string
		key            string
		value          string
		expectedStatus int
		expectedValue  string
	}{
		{name: "valid host", key: constants.SettingKeyDBHost, value: " localhost ", expectedStatus: http.StatusOK, expectedValue: "localhost"},
		{name: "invalid host", key: constants.SettingKeyDBHost, value: "bad\nhost", expectedStatus: http.StatusBadRequest},
		{name: "valid port", key: constants.SettingKeyDBPort, value: "01433", expectedStatus: http.StatusOK, expectedValue: "1433"},
		{name: "invalid port", key: constants.SettingKeyDBPort, value: "70000", expectedStatus: http.StatusBadRequest},
		{name: "valid user", key: constants.SettingKeyDBUser, value: " sa ", expectedStatus: http.StatusOK, expectedValue: "sa"},
		{name: "invalid user", key: constants.SettingKeyDBUser, value: "", expectedStatus: http.StatusBadRequest},
		{name: "valid password", key: constants.SettingKeyDBPass, value: " secret ", expectedStatus: http.StatusOK, expectedValue: " secret "},
		{name: "invalid password", key: constants.SettingKeyDBPass, value: "", expectedStatus: http.StatusBadRequest},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			internalDB := newTestInternalDB(t)
			createSettingsTestUser(t, internalDB)
			server := &Server{internalDB: internalDB}
			body := bytes.NewBufferString(`{"key":"` + test.key + `","value":` + jsonString(test.value) + `}`)
			req := settingsRequest(http.MethodPost, "/api/settings", body, constants.RoleAdmin)
			rr := httptest.NewRecorder()

			server.handleCreateSetting(rr, req)

			require.Equal(t, test.expectedStatus, rr.Code, rr.Body.String())
			if test.expectedStatus != http.StatusOK {
				return
			}

			var setting struct {
				Key   string `json:"key"`
				Value string `json:"value"`
			}
			require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &setting))
			require.Equal(t, test.expectedValue, setting.Value)
		})
	}
}

func TestHandleUpdateSettingUpdatesValueOnly(t *testing.T) {
	internalDB := newTestInternalDB(t)
	createSettingsTestUser(t, internalDB)
	_, err := internalDB.CreateSetting(constants.SettingKeyDBPort, "1433", nil)
	require.NoError(t, err)
	server := &Server{internalDB: internalDB}
	body := bytes.NewBufferString(`{"key":"DB_HOST","value":"1444"}`)
	req := settingsRequest(http.MethodPut, "/api/settings/DB_PORT", body, constants.RoleAdmin)
	req = withURLParam(req, "key", constants.SettingKeyDBPort)
	rr := httptest.NewRecorder()

	server.handleUpdateSetting(rr, req)

	require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())

	var setting struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	}
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &setting))
	require.Equal(t, constants.SettingKeyDBPort, setting.Key)
	require.Equal(t, "1444", setting.Value)
}

func TestHandleDeleteSettingDeletesExistingSetting(t *testing.T) {
	internalDB := newTestInternalDB(t)
	_, err := internalDB.CreateSetting(constants.SettingKeyDBPass, "secret", nil)
	require.NoError(t, err)
	server := &Server{internalDB: internalDB}
	req := settingsRequest(http.MethodDelete, "/api/settings/DB_PASS", nil, constants.RoleAdmin)
	req = withURLParam(req, "key", constants.SettingKeyDBPass)
	rr := httptest.NewRecorder()

	server.handleDeleteSetting(rr, req)

	require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())
	_, err = internalDB.GetSetting(constants.SettingKeyDBPass)
	require.Error(t, err)
}

func settingsRequest(method string, target string, body *bytes.Buffer, role string) *http.Request {
	var reader *strings.Reader
	if body != nil {
		reader = strings.NewReader(body.String())
	} else {
		reader = strings.NewReader("")
	}

	req := httptest.NewRequest(method, target, reader)
	ctx := req.Context()
	ctx = utils.SetUserIdInContext(ctx, 1)
	ctx = utils.SetUserRolesInContext(ctx, []string{role})
	return req.WithContext(ctx)
}

func withURLParam(req *http.Request, key string, value string) *http.Request {
	routeContext := chi.NewRouteContext()
	routeContext.URLParams.Add(key, value)
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, routeContext)
	return req.WithContext(ctx)
}

func jsonString(value string) string {
	data, _ := json.Marshal(value)
	return string(data)
}

func createSettingsTestUser(t *testing.T, internalDB db.InternalDB) {
	t.Helper()
	_, err := internalDB.CreateUser("settings-admin@example.com", "password", constants.RoleAdmin, nil)
	require.NoError(t, err)
}
