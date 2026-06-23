package server

import (
	"errors"
	"io"
	"net/http"
	"time"

	externalUtils "github.com/cyberinferno/go-utils/utils"
	"github.com/go-chi/chi/v5"
	"github.com/omnihance/omnihance-a3-agent/internal/constants"
	"github.com/omnihance/omnihance-a3-agent/internal/db"
	"github.com/omnihance/omnihance-a3-agent/internal/mw"
	"github.com/omnihance/omnihance-a3-agent/internal/permissions"
	"github.com/omnihance/omnihance-a3-agent/internal/utils"
	"github.com/project-agonyl/agonyl-utils-go/itemfile"
	agonylUtils "github.com/project-agonyl/agonyl-utils-go/utils"
)

func (s *Server) InitializeGameClientDataRoutes(r *chi.Mux) {
	r.Route("/api/game-client-data", func(r chi.Router) {
		r.Use(mw.CheckCookie(s.internalDB, s.cfg.CookieSecret))
		r.Get("/counts", s.handleGameClientDataCounts)
		r.Get("/monsters", s.handleMonsters)
		r.Post("/upload-mon-file", s.handleUploadMONFile)
		r.Get("/maps", s.handleMaps)
		r.Post("/upload-mc-file", s.handleUploadMCFile)
		r.Get("/items", s.handleItems)
		r.Post("/upload-it0-file", s.handleUploadIT0File)
		r.Post("/upload-it1-file", s.handleUploadIT1File)
		r.Post("/upload-it2-file", s.handleUploadIT2File)
		r.Post("/upload-it3-file", s.handleUploadIT3File)
	})
}

func (s *Server) handleGameClientDataCounts(w http.ResponseWriter, r *http.Request) {
	monsterCount, err := s.internalDB.GetMonsterClientDataCount()
	if err != nil {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusInternalServerError, map[string]interface{}{
			"errorCode": constants.ErrorCodeInternalServerError,
			"context":   "game-data",
			"errors":    []string{err.Error()},
		})
		return
	}

	mapCount, err := s.internalDB.GetMapClientDataCount()
	if err != nil {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusInternalServerError, map[string]interface{}{
			"errorCode": constants.ErrorCodeInternalServerError,
			"context":   "game-data",
			"errors":    []string{err.Error()},
		})
		return
	}

	itemCounts, err := s.internalDB.GetItemClientDataCounts()
	if err != nil {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusInternalServerError, map[string]interface{}{
			"errorCode": constants.ErrorCodeInternalServerError,
			"context":   "game-data",
			"errors":    []string{err.Error()},
		})
		return
	}

	_ = utils.WriteJSONResponse(w, GameClientDataCountsResponse{
		Monsters: monsterCount,
		Maps:     mapCount,
		Items:    itemCounts,
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
			ID:       item.ID,
			Name:     item.Name,
			ItemType: item.ItemType,
		})
	}

	_ = utils.WriteJSONResponse(w, response)
}

func (s *Server) handleUploadMONFile(w http.ResponseWriter, r *http.Request) {
	if !s.requireUserPermission(w, r, permissions.ActionUploadGameData) {
		return
	}

	userID, ok := utils.GetUserIdFromContext(r.Context())
	if !ok {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusUnauthorized, map[string]interface{}{
			"errorCode": constants.ErrorCodeUnauthorized,
			"context":   "game-data",
			"errors":    []string{"User ID not found in context"},
		})
		return
	}

	fileData, ok := s.readGameClientUploadFile(w, r)
	if !ok {
		return
	}

	agonylUtils.DecodeULL(fileData, len(fileData))
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
		name := externalUtils.ReadStringFromBytes(monster.Name[:])
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
	if !s.requireUserPermission(w, r, permissions.ActionUploadGameData) {
		return
	}

	userID, ok := utils.GetUserIdFromContext(r.Context())
	if !ok {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusUnauthorized, map[string]interface{}{
			"errorCode": constants.ErrorCodeUnauthorized,
			"context":   "game-data",
			"errors":    []string{"User ID not found in context"},
		})
		return
	}

	fileData, ok := s.readGameClientUploadFile(w, r)
	if !ok {
		return
	}

	agonylUtils.DecodeULL(fileData, len(fileData))
	mapData, err := s.fileEditor.ReadClientMapFileBytes(fileData)
	if err != nil {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusBadRequest, map[string]interface{}{
			"errorCode": constants.ErrorCodeBadRequest,
			"context":   "game-data",
			"errors":    []string{"Failed to parse map file: " + err.Error()},
		})
		return
	}

	now := time.Now()
	dbMapData := make([]db.MapClientData, 0, len(mapData))
	uniqueMapMap := make(map[uint32]bool)
	for _, mapItem := range mapData {
		name := externalUtils.ReadStringFromBytes(mapItem.Name[:])
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

func (s *Server) handleUploadIT0File(w http.ResponseWriter, r *http.Request) {
	s.handleUploadITFile(w, r, db.ItemClientDataTypeIT0, "IT0", s.fileEditor.ReadClientIT0FileBytes)
}

func (s *Server) handleUploadIT1File(w http.ResponseWriter, r *http.Request) {
	s.handleUploadITFile(w, r, db.ItemClientDataTypeIT1, "IT1", s.fileEditor.ReadClientIT1FileBytes)
}

func (s *Server) handleUploadIT2File(w http.ResponseWriter, r *http.Request) {
	s.handleUploadITFile(w, r, db.ItemClientDataTypeIT2, "IT2", s.fileEditor.ReadClientIT2FileBytes)
}

func (s *Server) handleUploadIT3File(w http.ResponseWriter, r *http.Request) {
	s.handleUploadITFile(w, r, db.ItemClientDataTypeIT3, "IT3", s.fileEditor.ReadClientIT3FileBytes)
}

func (s *Server) handleUploadITFile(
	w http.ResponseWriter,
	r *http.Request,
	itemType db.ItemClientDataType,
	label string,
	readItems func([]byte) ([]itemfile.Item, error),
) {
	if !s.requireUserPermission(w, r, permissions.ActionUploadGameData) {
		return
	}

	userID, ok := utils.GetUserIdFromContext(r.Context())
	if !ok {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusUnauthorized, map[string]interface{}{
			"errorCode": constants.ErrorCodeUnauthorized,
			"context":   "game-data",
			"errors":    []string{"User ID not found in context"},
		})
		return
	}

	fileData, ok := s.readGameClientUploadFile(w, r)
	if !ok {
		return
	}

	agonylUtils.DecodeULL(fileData, len(fileData))
	items, err := readItems(fileData)
	if err != nil {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusBadRequest, map[string]interface{}{
			"errorCode": constants.ErrorCodeBadRequest,
			"context":   "game-data",
			"errors":    []string{"Failed to parse " + label + " file: " + err.Error()},
		})
		return
	}

	now := time.Now()
	dbItemData := make([]db.ItemClientData, 0, len(items))
	uniqueItemMap := make(map[uint32]bool)
	for _, item := range items {
		if _, ok := uniqueItemMap[item.ItemCode]; ok {
			continue
		}

		uniqueItemMap[item.ItemCode] = true
		dbItemData = append(dbItemData, db.ItemClientData{
			ID:        int64(item.ItemCode),
			Name:      item.ItemName,
			ItemType:  string(itemType),
			CreatedBy: &userID,
			UpdatedBy: &userID,
			UpdatedAt: &now,
		})
	}

	if err := s.internalDB.BulkReplaceItemClientData(itemType, dbItemData); err != nil {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusInternalServerError, map[string]interface{}{
			"errorCode": constants.ErrorCodeInternalServerError,
			"context":   "game-data",
			"errors":    []string{"Failed to save " + label + " item data: " + err.Error()},
		})
		return
	}

	_ = utils.WriteJSONResponse(w, map[string]interface{}{
		"message": label + " item file uploaded successfully",
		"count":   len(dbItemData),
	})
}

func (s *Server) readGameClientUploadFile(w http.ResponseWriter, r *http.Request) ([]byte, bool) {
	maxUploadSize := s.cfg.MaxFileUploadSizeBytes()
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize+gameClientMultipartOverhead)

	if err := r.ParseMultipartForm(gameClientMultipartMemoryBytes(maxUploadSize)); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			writeGameDataUploadTooLargeError(w, "", maxUploadSize)
			return nil, false
		}

		_ = utils.WriteJSONResponseWithStatus(w, http.StatusBadRequest, map[string]interface{}{
			"errorCode": constants.ErrorCodeBadRequest,
			"context":   "game-data",
			"errors":    []string{"Failed to parse multipart form: " + err.Error()},
		})
		return nil, false
	}

	file, fileHeader, err := r.FormFile("file")
	if err != nil {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusBadRequest, map[string]interface{}{
			"errorCode": constants.ErrorCodeBadRequest,
			"context":   "game-data",
			"errors":    []string{"Failed to get file from form: " + err.Error()},
		})
		return nil, false
	}
	defer func() {
		_ = file.Close()
	}()

	if fileHeader.Size > maxUploadSize {
		writeGameDataUploadTooLargeError(w, fileHeader.Filename, maxUploadSize)
		return nil, false
	}

	fileData, err := io.ReadAll(io.LimitReader(file, maxUploadSize+1))
	if err != nil {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusInternalServerError, map[string]interface{}{
			"errorCode": constants.ErrorCodeInternalServerError,
			"context":   "game-data",
			"errors":    []string{"Failed to read file: " + err.Error()},
		})
		return nil, false
	}

	if int64(len(fileData)) > maxUploadSize {
		writeGameDataUploadTooLargeError(w, fileHeader.Filename, maxUploadSize)
		return nil, false
	}

	return fileData, true
}

type GameClientDataResponse struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	ItemType string `json:"item_type,omitempty"`
}

type GameClientDataCountsResponse struct {
	Monsters int64                   `json:"monsters"`
	Maps     int64                   `json:"maps"`
	Items    db.ItemClientDataCounts `json:"items"`
}
