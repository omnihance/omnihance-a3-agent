package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"unicode"

	"github.com/go-chi/chi/v5"
	"github.com/omnihance/omnihance-a3-agent/internal/constants"
	"github.com/omnihance/omnihance-a3-agent/internal/db"
	"github.com/omnihance/omnihance-a3-agent/internal/mw"
	"github.com/omnihance/omnihance-a3-agent/internal/permissions"
	"github.com/omnihance/omnihance-a3-agent/internal/utils"
)

const settingsErrorContext = "settings"

const requiredServerInfoFileName = "SvrInfo.ini"

type SettingDefinition struct {
	Key         string `json:"key"`
	Label       string `json:"label"`
	Description string `json:"description"`
	ValueType   string `json:"value_type"`
	InputType   string `json:"input_type"`
}

var supportedSettingDefinitions = []SettingDefinition{
	{
		Key:         constants.SettingKeyDBHost,
		Label:       "Game server DB host",
		Description: "SQL Server host name or IP address used by the game server database.",
		ValueType:   "string",
		InputType:   "text",
	},
	{
		Key:         constants.SettingKeyDBPort,
		Label:       "Game server DB port",
		Description: "SQL Server TCP port used by the game server database.",
		ValueType:   "uint16",
		InputType:   "number",
	},
	{
		Key:         constants.SettingKeyDBUser,
		Label:       "Game server DB username",
		Description: "SQL Server login username used by the game server database.",
		ValueType:   "string",
		InputType:   "text",
	},
	{
		Key:         constants.SettingKeyDBPass,
		Label:       "Game server DB password",
		Description: "SQL Server login password used by the game server database.",
		ValueType:   "string",
		InputType:   "password",
	},
	{
		Key:         constants.SettingKeyZoneServerPath,
		Label:       "Zone Server Path",
		Description: "Directory path used for the Zone Server. Must contain SvrInfo.ini.",
		ValueType:   "string",
		InputType:   "directory",
	},
	{
		Key:         constants.SettingKeyAccountServerPath,
		Label:       "Account Server Path",
		Description: "Directory path used for the Account Server. Must contain SvrInfo.ini.",
		ValueType:   "string",
		InputType:   "directory",
	},
	{
		Key:         constants.SettingKeyMainServerPath,
		Label:       "Main Server Path",
		Description: "Directory path used for the Main Server. Must contain SvrInfo.ini.",
		ValueType:   "string",
		InputType:   "directory",
	},
	{
		Key:         constants.SettingKeyBattleServerPath,
		Label:       "Battle Server Path",
		Description: "Directory path used for the Battle Server. Must contain SvrInfo.ini.",
		ValueType:   "string",
		InputType:   "directory",
	},
}

func (s *Server) InitializeSettingsRoutes(r *chi.Mux) {
	r.Route("/api/settings", func(r chi.Router) {
		r.Use(mw.CheckCookie(s.internalDB, s.cfg.CookieSecret))
		r.Get("/", s.handleListSettings)
		r.Post("/", s.handleCreateSetting)
		r.Put("/{key}", s.handleUpdateSetting)
		r.Delete("/{key}", s.handleDeleteSetting)
	})
}

func (s *Server) handleListSettings(w http.ResponseWriter, r *http.Request) {
	if !s.requireUserPermission(w, r, permissions.ActionManageServer) {
		return
	}

	settings, err := s.internalDB.GetSettings()
	if err != nil {
		writeSettingsError(w, http.StatusInternalServerError, constants.ErrorCodeInternalServerError, "Failed to retrieve settings")
		return
	}

	settingsByKey := make(map[string]db.Settings, len(settings))
	for _, setting := range settings {
		if isSupportedSettingKey(setting.Key) {
			settingsByKey[setting.Key] = setting
		}
	}

	filteredSettings := make([]db.Settings, 0, len(settingsByKey))
	for _, definition := range supportedSettingDefinitions {
		if setting, ok := settingsByKey[definition.Key]; ok {
			filteredSettings = append(filteredSettings, setting)
		}
	}

	_ = utils.WriteJSONResponse(w, SettingsResponse{
		Settings:    filteredSettings,
		Definitions: supportedSettingDefinitions,
	})
}

func (s *Server) handleCreateSetting(w http.ResponseWriter, r *http.Request) {
	if !s.requireUserPermission(w, r, permissions.ActionManageServer) {
		return
	}

	var req SettingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeSettingsError(w, http.StatusBadRequest, constants.ErrorCodeBadRequest, "Invalid request body")
		return
	}

	key := strings.TrimSpace(req.Key)
	value, err := s.validateSettingValue(key, req.Value)
	if err != nil {
		writeSettingsError(w, http.StatusBadRequest, constants.ErrorCodeBadRequest, err.Error())
		return
	}

	setting, err := s.internalDB.CreateSetting(key, value, settingUserID(r))
	if err != nil {
		if errors.Is(err, db.ErrSettingAlreadyExists) {
			writeSettingsError(w, http.StatusBadRequest, constants.ErrorCodeBadRequest, "Setting already exists")
			return
		}

		writeSettingsError(w, http.StatusInternalServerError, constants.ErrorCodeInternalServerError, "Failed to create setting")
		return
	}

	_ = utils.WriteJSONResponse(w, setting)
}

func (s *Server) handleUpdateSetting(w http.ResponseWriter, r *http.Request) {
	if !s.requireUserPermission(w, r, permissions.ActionManageServer) {
		return
	}

	key := strings.TrimSpace(chi.URLParam(r, "key"))
	var req UpdateSettingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeSettingsError(w, http.StatusBadRequest, constants.ErrorCodeBadRequest, "Invalid request body")
		return
	}

	value, err := s.validateSettingValue(key, req.Value)
	if err != nil {
		writeSettingsError(w, http.StatusBadRequest, constants.ErrorCodeBadRequest, err.Error())
		return
	}

	setting, err := s.internalDB.UpdateSetting(key, value, settingUserID(r))
	if err != nil {
		if errors.Is(err, db.ErrSettingNotFound) {
			writeSettingsError(w, http.StatusNotFound, constants.ErrorCodeNotFound, "Setting not found")
			return
		}

		writeSettingsError(w, http.StatusInternalServerError, constants.ErrorCodeInternalServerError, "Failed to update setting")
		return
	}

	_ = utils.WriteJSONResponse(w, setting)
}

func (s *Server) handleDeleteSetting(w http.ResponseWriter, r *http.Request) {
	if !s.requireUserPermission(w, r, permissions.ActionManageServer) {
		return
	}

	key := strings.TrimSpace(chi.URLParam(r, "key"))
	if !isSupportedSettingKey(key) {
		writeSettingsError(w, http.StatusBadRequest, constants.ErrorCodeBadRequest, "Unsupported setting key")
		return
	}

	if err := s.internalDB.DeleteSetting(key); err != nil {
		if errors.Is(err, db.ErrSettingNotFound) {
			writeSettingsError(w, http.StatusNotFound, constants.ErrorCodeNotFound, "Setting not found")
			return
		}

		writeSettingsError(w, http.StatusInternalServerError, constants.ErrorCodeInternalServerError, "Failed to delete setting")
		return
	}

	_ = utils.WriteJSONResponse(w, map[string]interface{}{
		"message": "Setting deleted successfully",
	})
}

func (s *Server) validateSettingValue(key string, value string) (string, error) {
	if !isSupportedSettingKey(key) {
		return "", fmt.Errorf("unsupported setting key")
	}

	switch key {
	case constants.SettingKeyDBHost:
		return validateStringSetting(value, "Game server DB host", 255, true, true)
	case constants.SettingKeyDBPort:
		return validatePortSetting(value)
	case constants.SettingKeyDBUser:
		return validateStringSetting(value, "Game server DB username", 128, true, false)
	case constants.SettingKeyDBPass:
		return validateStringSetting(value, "Game server DB password", 512, false, false)
	case constants.SettingKeyZoneServerPath:
		return s.validateDirectorySetting(value, "Zone Server Path")
	case constants.SettingKeyAccountServerPath:
		return s.validateDirectorySetting(value, "Account Server Path")
	case constants.SettingKeyMainServerPath:
		return s.validateDirectorySetting(value, "Main Server Path")
	case constants.SettingKeyBattleServerPath:
		return s.validateDirectorySetting(value, "Battle Server Path")
	default:
		return "", fmt.Errorf("unsupported setting key")
	}
}

func (s *Server) validateDirectorySetting(value string, label string) (string, error) {
	normalizedValue := filepath.Clean(strings.TrimSpace(value))
	if normalizedValue == "" || normalizedValue == "." {
		return "", fmt.Errorf("%s is required", label)
	}

	info, err := s.fileEditor.Stat(normalizedValue)
	if err != nil {
		if s.fileEditor.IsNotExist(err) {
			return "", fmt.Errorf("%s must be an existing directory", label)
		}

		return "", fmt.Errorf("cannot access %s: %v", label, err)
	}

	if !info.IsDir() {
		return "", fmt.Errorf("%s must be a directory", label)
	}

	serverInfoPath := filepath.Join(normalizedValue, requiredServerInfoFileName)
	serverInfo, err := s.fileEditor.Stat(serverInfoPath)
	if err != nil {
		if s.fileEditor.IsNotExist(err) {
			return "", fmt.Errorf("%s must contain %s", label, requiredServerInfoFileName)
		}

		return "", fmt.Errorf("cannot access %s in %s: %v", requiredServerInfoFileName, label, err)
	}

	if serverInfo.IsDir() {
		return "", fmt.Errorf("%s must contain %s as a file", label, requiredServerInfoFileName)
	}

	return normalizedValue, nil
}

func validateStringSetting(value string, label string, maxLength int, trim bool, rejectControlCharacters bool) (string, error) {
	normalizedValue := value
	if trim {
		normalizedValue = strings.TrimSpace(value)
	}

	if normalizedValue == "" {
		return "", fmt.Errorf("%s is required", label)
	}

	if len(normalizedValue) > maxLength {
		return "", fmt.Errorf("%s must be at most %d characters", label, maxLength)
	}

	if rejectControlCharacters {
		for _, char := range normalizedValue {
			if unicode.IsControl(char) {
				return "", fmt.Errorf("%s cannot contain control characters", label)
			}
		}
	}

	return normalizedValue, nil
}

func validatePortSetting(value string) (string, error) {
	normalizedValue := strings.TrimSpace(value)
	port, err := strconv.ParseUint(normalizedValue, 10, 16)
	if err != nil || port == 0 {
		return "", fmt.Errorf("game server DB port must be between 1 and 65535")
	}

	return strconv.FormatUint(port, 10), nil
}

func isSupportedSettingKey(key string) bool {
	for _, definition := range supportedSettingDefinitions {
		if definition.Key == key {
			return true
		}
	}

	return false
}

func settingUserID(r *http.Request) *int64 {
	userID, ok := utils.GetUserIdFromContext(r.Context())
	if !ok {
		return nil
	}

	return &userID
}

func writeSettingsError(w http.ResponseWriter, status int, errorCode string, message string) {
	_ = utils.WriteJSONResponseWithStatus(w, status, map[string]interface{}{
		"errorCode": errorCode,
		"context":   settingsErrorContext,
		"errors":    []string{message},
	})
}

type SettingsResponse struct {
	Settings    []db.Settings       `json:"settings"`
	Definitions []SettingDefinition `json:"definitions"`
}

type SettingRequest struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type UpdateSettingRequest struct {
	Value string `json:"value"`
}
