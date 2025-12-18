package server

import (
	"io"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/omnihance/omnihance-a3-agent/internal/constants"
	"github.com/omnihance/omnihance-a3-agent/internal/db"
	"github.com/omnihance/omnihance-a3-agent/internal/mw"
	"github.com/omnihance/omnihance-a3-agent/internal/utils"
)

func (s *Server) InitializeGameClientDataRoutes(r *chi.Mux) {
	r.Route("/api/game-client-data", func(r chi.Router) {
		r.Use(mw.CheckCookie(s.internalDB, s.cfg.CookieSecret))
		r.Get("/monsters", s.handleMonsters)
		r.Post("/upload-mon-file", s.handleUploadMONFile)
		r.Get("/maps", s.handleMaps)
		r.Post("/upload-mc-file", s.handleUploadMCFile)
		r.Get("/items", s.handleItems)
	})
}

func (s *Server) handleMonsters(w http.ResponseWriter, r *http.Request) {
	search := r.URL.Query().Get("s")

	data, err := s.internalDB.GetAllMonsterClientData(search)
	if err != nil {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusInternalServerError, map[string]interface{}{
			"errorCode": constants.ErrorCodeInternalServerError,
			"context":   "game-data",
			"errors":    []string{err.Error()},
		})
		return
	}

	response := make([]GameClientDataResponse, 0, len(data))
	for _, item := range data {
		response = append(response, GameClientDataResponse{
			ID:   item.ID,
			Name: item.Name,
		})
	}

	_ = utils.WriteJSONResponse(w, response)
}

func (s *Server) handleMaps(w http.ResponseWriter, r *http.Request) {
	search := r.URL.Query().Get("s")

	data, err := s.internalDB.GetAllMapClientData(search)
	if err != nil {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusInternalServerError, map[string]interface{}{
			"errorCode": constants.ErrorCodeInternalServerError,
			"context":   "game-data",
			"errors":    []string{err.Error()},
		})
		return
	}

	response := make([]GameClientDataResponse, 0, len(data))
	for _, item := range data {
		response = append(response, GameClientDataResponse{
			ID:   item.ID,
			Name: item.Name,
		})
	}

	_ = utils.WriteJSONResponse(w, response)
}

func (s *Server) handleItems(w http.ResponseWriter, r *http.Request) {
	search := r.URL.Query().Get("s")

	data, err := s.internalDB.GetAllItemClientData(search)
	if err != nil {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusInternalServerError, map[string]interface{}{
			"errorCode": constants.ErrorCodeInternalServerError,
			"context":   "game-data",
			"errors":    []string{err.Error()},
		})
		return
	}

	response := make([]GameClientDataResponse, 0, len(data))
	for _, item := range data {
		response = append(response, GameClientDataResponse{
			ID:   item.ID,
			Name: item.Name,
		})
	}

	_ = utils.WriteJSONResponse(w, response)
}

func (s *Server) handleUploadMONFile(w http.ResponseWriter, r *http.Request) {
	userID, ok := utils.GetUserIdFromContext(r.Context())
	if !ok {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusUnauthorized, map[string]interface{}{
			"errorCode": constants.ErrorCodeUnauthorized,
			"context":   "game-data",
			"errors":    []string{"User ID not found in context"},
		})
		return
	}

	maxUploadSize := int64(s.cfg.MaxFileUploadSizeMb) * 1024 * 1024

	if err := r.ParseMultipartForm(maxUploadSize); err != nil {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusBadRequest, map[string]interface{}{
			"errorCode": constants.ErrorCodeBadRequest,
			"context":   "game-data",
			"errors":    []string{"Failed to parse multipart form: " + err.Error()},
		})
		return
	}

	file, fileHeader, err := r.FormFile("file")
	if err != nil {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusBadRequest, map[string]interface{}{
			"errorCode": constants.ErrorCodeBadRequest,
			"context":   "game-data",
			"errors":    []string{"Failed to get file from form: " + err.Error()},
		})
		return
	}
	defer func() {
		_ = file.Close()
	}()

	if fileHeader.Size > maxUploadSize {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusBadRequest, map[string]interface{}{
			"errorCode": constants.ErrorCodeBadRequest,
			"context":   "game-data",
			"errors":    []string{"File size exceeds maximum allowed size"},
		})
		return
	}

	fileData, err := io.ReadAll(file)
	if err != nil {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusInternalServerError, map[string]interface{}{
			"errorCode": constants.ErrorCodeInternalServerError,
			"context":   "game-data",
			"errors":    []string{"Failed to read file: " + err.Error()},
		})
		return
	}

	if len(fileData) > int(maxUploadSize) {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusBadRequest, map[string]interface{}{
			"errorCode": constants.ErrorCodeBadRequest,
			"context":   "game-data",
			"errors":    []string{"File size exceeds maximum allowed size"},
		})
		return
	}

	utils.DecodeULL(&fileData, len(fileData))
	monsterData, err := s.fileEditor.ReadClientMonsterFileBytes(fileData)
	if err != nil {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusBadRequest, map[string]interface{}{
			"errorCode": constants.ErrorCodeBadRequest,
			"context":   "game-data",
			"errors":    []string{"Failed to parse monster file: " + err.Error()},
		})
		return
	}

	now := time.Now()
	dbMonsterData := make([]db.MonsterClientData, 0, len(monsterData))
	uniqueMonsterMap := make(map[uint32]bool)
	for _, monster := range monsterData {
		name := utils.ReadStringFromBytes(monster.Name[:])
		if _, ok := uniqueMonsterMap[monster.ID]; ok {
			continue
		}

		uniqueMonsterMap[monster.ID] = true
		dbMonsterData = append(dbMonsterData, db.MonsterClientData{
			ID:        int64(monster.ID),
			Name:      name,
			CreatedBy: &userID,
			UpdatedBy: &userID,
			UpdatedAt: &now,
		})
	}

	if err := s.internalDB.BulkReplaceMonsterClientData(dbMonsterData); err != nil {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusInternalServerError, map[string]interface{}{
			"errorCode": constants.ErrorCodeInternalServerError,
			"context":   "game-data",
			"errors":    []string{"Failed to save monster data: " + err.Error()},
		})
		return
	}

	_ = utils.WriteJSONResponse(w, map[string]interface{}{
		"message": "Monster list file uploaded successfully",
		"count":   len(dbMonsterData),
	})
}

func (s *Server) handleUploadMCFile(w http.ResponseWriter, r *http.Request) {
	userID, ok := utils.GetUserIdFromContext(r.Context())
	if !ok {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusUnauthorized, map[string]interface{}{
			"errorCode": constants.ErrorCodeUnauthorized,
			"context":   "game-data",
			"errors":    []string{"User ID not found in context"},
		})
	}

	maxUploadSize := int64(s.cfg.MaxFileUploadSizeMb) * 1024 * 1024

	if err := r.ParseMultipartForm(maxUploadSize); err != nil {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusBadRequest, map[string]interface{}{
			"errorCode": constants.ErrorCodeBadRequest,
			"context":   "game-data",
			"errors":    []string{"Failed to parse multipart form: " + err.Error()},
		})
	}

	file, fileHeader, err := r.FormFile("file")
	if err != nil {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusBadRequest, map[string]interface{}{
			"errorCode": constants.ErrorCodeBadRequest,
			"context":   "game-data",
			"errors":    []string{"Failed to get file from form: " + err.Error()},
		})
	}

	defer func() {
		_ = file.Close()
	}()

	if fileHeader.Size > maxUploadSize {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusBadRequest, map[string]interface{}{
			"errorCode": constants.ErrorCodeBadRequest,
			"context":   "game-data",
			"errors":    []string{"File size exceeds maximum allowed size"},
		})
	}

	fileData, err := io.ReadAll(file)
	if err != nil {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusInternalServerError, map[string]interface{}{
			"errorCode": constants.ErrorCodeInternalServerError,
			"context":   "game-data",
			"errors":    []string{"Failed to read file: " + err.Error()},
		})
	}

	if len(fileData) > int(maxUploadSize) {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusBadRequest, map[string]interface{}{
			"errorCode": constants.ErrorCodeBadRequest,
			"context":   "game-data",
			"errors":    []string{"File size exceeds maximum allowed size"},
		})
	}

	utils.DecodeULL(&fileData, len(fileData))
	mapData, err := s.fileEditor.ReadClientMapFileBytes(fileData)
	if err != nil {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusBadRequest, map[string]interface{}{
			"errorCode": constants.ErrorCodeBadRequest,
			"context":   "game-data",
			"errors":    []string{"Failed to parse map file: " + err.Error()},
		})
	}

	now := time.Now()
	dbMapData := make([]db.MapClientData, 0, len(mapData))
	uniqueMapMap := make(map[uint32]bool)
	for _, mapItem := range mapData {
		name := utils.ReadStringFromBytes(mapItem.Name[:])
		if _, ok := uniqueMapMap[mapItem.ID]; ok {
			continue
		}

		uniqueMapMap[mapItem.ID] = true
		dbMapData = append(dbMapData, db.MapClientData{
			ID:        int64(mapItem.ID),
			Name:      name,
			CreatedBy: &userID,
			UpdatedBy: &userID,
			UpdatedAt: &now,
		})
	}

	if err := s.internalDB.BulkReplaceMapClientData(dbMapData); err != nil {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusInternalServerError, map[string]interface{}{
			"errorCode": constants.ErrorCodeInternalServerError,
			"context":   "game-data",
			"errors":    []string{"Failed to save map data: " + err.Error()},
		})
		return
	}

	_ = utils.WriteJSONResponse(w, map[string]interface{}{
		"message": "Map list file uploaded successfully",
		"count":   len(dbMapData),
	})
}

type GameClientDataResponse struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}
