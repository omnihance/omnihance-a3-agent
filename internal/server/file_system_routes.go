package server

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io/fs"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"time"

	externalUtils "github.com/cyberinferno/go-utils/utils"
	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/omnihance/omnihance-a3-agent/internal/constants"
	"github.com/omnihance/omnihance-a3-agent/internal/logger"
	"github.com/omnihance/omnihance-a3-agent/internal/mw"
	"github.com/omnihance/omnihance-a3-agent/internal/permissions"
	"github.com/omnihance/omnihance-a3-agent/internal/services"
	"github.com/omnihance/omnihance-a3-agent/internal/utils"
	"github.com/project-agonyl/agonyl-utils-go/questfile"
)

func (s *Server) InitializeFileSystemRoutes(r *chi.Mux) {
	r.Route("/api/file-tree", func(r chi.Router) {
		r.Use(mw.CheckCookie(s.internalDB, s.cfg.CookieSecret))
		r.Get("/", s.handleFileTree)
		r.Get("/npc-file", s.handleNPCFileData)
		r.Put("/npc-file", s.handleUpdateNPCFile)
		r.Get("/text-file", s.handleTextFileData)
		r.Put("/text-file", s.handleUpdateTextFile)
		r.Get("/spawn-file", s.handleSpawnFileData)
		r.Put("/spawn-file", s.handleUpdateSpawnFile)
		r.Get("/quest-file", s.handleQuestFileData)
		r.Put("/quest-file", s.handleUpdateQuestFile)
		r.Post("/revert-file", s.handleRevertFile)
		r.Get("/revision-summary", s.handleRevisionSummary)
	})
}

func (s *Server) handleFileTree(w http.ResponseWriter, r *http.Request) {
	pathParam := r.URL.Query().Get("path")
	showDotfiles, _ := strconv.ParseBool(r.URL.Query().Get("show_dotfiles"))

	var rootNode *FileNode
	var err error

	if pathParam == "" {
		rootNode, err = s.getSystemRoots(showDotfiles)
	} else {
		cleanPath := filepath.Clean(pathParam)
		rootNode, err = s.getDirectoryNode(cleanPath, showDotfiles)
	}

	if err != nil {
		if s.fileEditor.IsNotExist(err) {
			_ = utils.WriteJSONResponseWithStatus(w, http.StatusNotFound, map[string]interface{}{
				"errorCode": constants.ErrorCodeNotFound,
				"context":   "file-system",
				"errors":    []string{"Path not found"},
			})
			return
		}

		_ = utils.WriteJSONResponseWithStatus(w, http.StatusInternalServerError, map[string]interface{}{
			"errorCode": constants.ErrorCodeInternalServerError,
			"context":   "file-system",
			"errors":    []string{err.Error()},
		})
		return
	}

	response := FileTreeResponse{
		OS:       runtime.GOOS,
		FileTree: rootNode,
	}

	_ = utils.WriteJSONResponse(w, response)
}

func (s *Server) getSystemRoots(showDotfiles bool) (*FileNode, error) {
	hostname, _ := s.fileEditor.Hostname()
	if hostname == "" {
		hostname = "A3 Online Server"
	}

	root := &FileNode{
		ID:       "root",
		Name:     hostname,
		Kind:     "directory",
		Depth:    0,
		Children: []*FileNode{},
	}

	if runtime.GOOS == "windows" {
		for _, drive := range "ABCDEFGHIJKLMNOPQRSTUVWXYZ" {
			drivePath := string(drive) + ":\\"
			if info, err := s.fileEditor.Stat(drivePath); err == nil {
				modTime := info.ModTime()
				node := &FileNode{
					ID:           utils.GenerateMD5Hash(drivePath),
					Name:         string(drive) + ":",
					Kind:         "directory",
					Depth:        1,
					LastModified: &modTime,
					Permissions:  info.Mode().String(),
					Children:     []*FileNode{},
				}
				root.Children = append(root.Children, node)
			}
		}
	} else {
		rootPath := "/"
		entries, err := s.fileEditor.ReadDir(rootPath)
		if err != nil {
			return nil, err
		}

		for _, entry := range entries {
			if !showDotfiles && len(entry.Name()) > 0 && entry.Name()[0] == '.' {
				continue
			}

			node := s.createNodeFromEntry(rootPath, entry, 1)
			root.Children = append(root.Children, node)
		}
	}

	return root, nil
}

func (s *Server) getDirectoryNode(path string, showDotfiles bool) (*FileNode, error) {
	info, err := s.fileEditor.Stat(path)
	if err != nil {
		return nil, err
	}

	name := info.Name()
	if runtime.GOOS == "windows" && len(path) == 3 && path[1] == ':' && (path[2] == '\\' || path[2] == '/') {
		name = path[:2]
	}

	modTime := info.ModTime()
	node := &FileNode{
		ID:           utils.GenerateMD5Hash(path),
		Name:         name,
		Kind:         "file",
		Depth:        0,
		LastModified: &modTime,
		Permissions:  info.Mode().String(),
		Children:     []*FileNode{},
	}

	if info.IsDir() {
		node.Kind = "directory"
		entries, err := s.fileEditor.ReadDir(path)
		if err == nil {
			for _, entry := range entries {
				if !showDotfiles && len(entry.Name()) > 0 && entry.Name()[0] == '.' {
					continue
				}

				child := s.createNodeFromEntry(path, entry, 1)
				node.Children = append(node.Children, child)
			}
		}
	} else {
		node.FileSize = info.Size()
		node.FileExtension = filepath.Ext(name)
		node.MimeType = mime.TypeByExtension(node.FileExtension)
		node.FileType = s.fileEditor.GetFileType(path, info)
		node.IsEditable = s.fileEditor.IsFileEditable(path, info)
		node.IsViewable = s.fileEditor.IsFileViewable(path, info)
		node.APIEndpoint = s.fileEditor.GetFileAPIEndpoint(path, info)
	}

	return node, nil
}

func (s *Server) createNodeFromEntry(parentPath string, entry fs.DirEntry, depth int) *FileNode {
	kind := "file"
	if entry.IsDir() {
		kind = "directory"
	}

	fullPath := filepath.Join(parentPath, entry.Name())

	node := &FileNode{
		ID:       utils.GenerateMD5Hash(fullPath),
		Name:     entry.Name(),
		Kind:     kind,
		Depth:    depth,
		Children: []*FileNode{},
	}

	info, err := entry.Info()
	if err == nil {
		modTime := info.ModTime()
		node.LastModified = &modTime
		node.Permissions = info.Mode().String()
		if !entry.IsDir() {
			node.FileSize = info.Size()
			node.FileExtension = filepath.Ext(entry.Name())
			node.MimeType = mime.TypeByExtension(node.FileExtension)
			node.FileType = s.fileEditor.GetFileType(fullPath, info)
			node.IsEditable = s.fileEditor.IsFileEditable(fullPath, info)
			node.IsViewable = s.fileEditor.IsFileViewable(fullPath, info)
			node.APIEndpoint = s.fileEditor.GetFileAPIEndpoint(fullPath, info)
		}
	}

	return node
}

func (s *Server) handleNPCFileData(w http.ResponseWriter, r *http.Request) {
	pathParam := r.URL.Query().Get("path")

	if pathParam == "" {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusBadRequest, map[string]interface{}{
			"errorCode": constants.ErrorCodeBadRequest,
			"context":   "file-system",
			"errors":    []string{"Path parameter is required"},
		})
		return
	}

	cleanPath := filepath.Clean(pathParam)
	info, err := s.fileEditor.Stat(cleanPath)

	if err != nil {
		if s.fileEditor.IsNotExist(err) {
			_ = utils.WriteJSONResponseWithStatus(w, http.StatusNotFound, map[string]interface{}{
				"errorCode": constants.ErrorCodeNotFound,
				"context":   "file-system",
				"errors":    []string{"Path not found"},
			})
			return
		}

		_ = utils.WriteJSONResponseWithStatus(w, http.StatusInternalServerError, map[string]interface{}{
			"errorCode": constants.ErrorCodeFileReadError,
			"context":   "file-system",
			"errors":    []string{"Cannot read file: " + err.Error()},
		})
		return
	}

	if info.IsDir() {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusBadRequest, map[string]interface{}{
			"errorCode": constants.ErrorCodePathIsDirectory,
			"context":   "file-system",
			"errors":    []string{"Path is a directory, not a file"},
		})
		return
	}

	fileType := s.fileEditor.GetFileType(cleanPath, info)
	if fileType != services.FileTypeNPC {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusBadRequest, map[string]interface{}{
			"errorCode": constants.ErrorCodeFileNotViewable,
			"context":   "file-system",
			"errors":    []string{"File is not an NPC file"},
		})
		return
	}

	npcData, err := s.fileEditor.ReadNPCFileData(cleanPath)
	if err != nil {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusInternalServerError, map[string]interface{}{
			"errorCode": constants.ErrorCodeFileReadError,
			"context":   "file-system",
			"errors":    []string{"Failed to read NPC file data: " + err.Error()},
		})
		return
	}

	id := npcData.Id
	respawnRate := npcData.RespawnRate
	attackTypeInfo := npcData.AttackTypeInfo
	targetSelectionInfo := npcData.TargetSelectionInfo
	defense := npcData.Defense
	additionalDefense := npcData.AdditionalDefense
	attackSpeedLow := npcData.AttackSpeedLow
	attackSpeedHigh := npcData.AttackSpeedHigh
	movementSpeed := npcData.MovementSpeed
	level := npcData.Level
	playerExp := npcData.PlayerExp
	appearance := npcData.Appearance
	hp := npcData.HP
	blueAttackDefense := npcData.BlueAttackDefense
	redAttackDefense := npcData.RedAttackDefense
	greyAttackDefense := npcData.GreyAttackDefense
	mercenaryExp := npcData.MercenaryExp
	apiData := NPCFileAPIData{
		Name:                externalUtils.ReadStringFromBytes(npcData.Name[:]),
		Id:                  &id,
		RespawnRate:         &respawnRate,
		AttackTypeInfo:      &attackTypeInfo,
		TargetSelectionInfo: &targetSelectionInfo,
		Defense:             &defense,
		AdditionalDefense:   &additionalDefense,
		Attacks:             npcData.Attacks[:],
		AttackSpeedLow:      &attackSpeedLow,
		AttackSpeedHigh:     &attackSpeedHigh,
		MovementSpeed:       &movementSpeed,
		Level:               &level,
		PlayerExp:           &playerExp,
		Appearance:          &appearance,
		HP:                  &hp,
		BlueAttackDefense:   &blueAttackDefense,
		RedAttackDefense:    &redAttackDefense,
		GreyAttackDefense:   &greyAttackDefense,
		MercenaryExp:        &mercenaryExp,
	}

	_ = utils.WriteJSONResponse(w, apiData)
}

func (s *Server) handleTextFileData(w http.ResponseWriter, r *http.Request) {
	pathParam := r.URL.Query().Get("path")

	if pathParam == "" {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusBadRequest, map[string]interface{}{
			"errorCode": constants.ErrorCodeBadRequest,
			"context":   "file-system",
			"errors":    []string{"Path parameter is required"},
		})
		return
	}

	cleanPath := filepath.Clean(pathParam)
	info, err := s.fileEditor.Stat(cleanPath)

	if err != nil {
		if s.fileEditor.IsNotExist(err) {
			_ = utils.WriteJSONResponseWithStatus(w, http.StatusNotFound, map[string]interface{}{
				"errorCode": constants.ErrorCodeNotFound,
				"context":   "file-system",
				"errors":    []string{"Path not found"},
			})
			return
		}

		_ = utils.WriteJSONResponseWithStatus(w, http.StatusInternalServerError, map[string]interface{}{
			"errorCode": constants.ErrorCodeFileReadError,
			"context":   "file-system",
			"errors":    []string{"Cannot read file: " + err.Error()},
		})
		return
	}

	if info.IsDir() {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusBadRequest, map[string]interface{}{
			"errorCode": constants.ErrorCodePathIsDirectory,
			"context":   "file-system",
			"errors":    []string{"Path is a directory, not a file"},
		})
		return
	}

	fileType := s.fileEditor.GetFileType(cleanPath, info)
	if fileType != services.FileTypeText {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusBadRequest, map[string]interface{}{
			"errorCode": constants.ErrorCodeFileNotViewable,
			"context":   "file-system",
			"errors":    []string{"File is not a text file"},
		})
		return
	}

	content, err := s.fileEditor.ReadFile(cleanPath)
	if err != nil {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusInternalServerError, map[string]interface{}{
			"errorCode": constants.ErrorCodeFileReadError,
			"context":   "file-system",
			"errors":    []string{"Failed to read text file: " + err.Error()},
		})
		return
	}

	apiData := TextFileAPIData{
		Content: string(content),
	}

	_ = utils.WriteJSONResponse(w, apiData)
}

func (s *Server) handleSpawnFileData(w http.ResponseWriter, r *http.Request) {
	pathParam := r.URL.Query().Get("path")
	if pathParam == "" {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusBadRequest, map[string]interface{}{
			"errorCode": constants.ErrorCodeBadRequest,
			"context":   "file-system",
			"errors":    []string{"Path parameter is required"},
		})
		return
	}

	cleanPath := filepath.Clean(pathParam)
	info, err := s.fileEditor.Stat(cleanPath)
	if err != nil {
		if s.fileEditor.IsNotExist(err) {
			_ = utils.WriteJSONResponseWithStatus(w, http.StatusNotFound, map[string]interface{}{
				"errorCode": constants.ErrorCodeNotFound,
				"context":   "file-system",
				"errors":    []string{"Path not found"},
			})
			return
		}

		_ = utils.WriteJSONResponseWithStatus(w, http.StatusInternalServerError, map[string]interface{}{
			"errorCode": constants.ErrorCodeFileReadError,
			"context":   "file-system",
			"errors":    []string{"Cannot read file: " + err.Error()},
		})
		return
	}

	if info.IsDir() {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusBadRequest, map[string]interface{}{
			"errorCode": constants.ErrorCodePathIsDirectory,
			"context":   "file-system",
			"errors":    []string{"Path is a directory, not a file"},
		})
		return
	}

	fileType := s.fileEditor.GetFileType(cleanPath, info)
	if fileType != services.FileTypeSpawn {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusBadRequest, map[string]interface{}{
			"errorCode": constants.ErrorCodeFileNotViewable,
			"context":   "file-system",
			"errors":    []string{"File is not a spawn file"},
		})
		return
	}

	spawnData, err := s.fileEditor.ReadSpawnFileData(cleanPath)
	if err != nil {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusInternalServerError, map[string]interface{}{
			"errorCode": constants.ErrorCodeFileReadError,
			"context":   "file-system",
			"errors":    []string{"Failed to read spawn file data: " + err.Error()},
		})
		return
	}

	apiSpawns := make([]NPCSpawnAPIData, len(spawnData))
	for i, spawn := range spawnData {
		id := spawn.Id
		x := spawn.X
		y := spawn.Y
		unknown1 := spawn.Unknown1
		orientation := spawn.Orientation
		spwanStep := spawn.SpwanStep
		apiSpawns[i] = NPCSpawnAPIData{
			Id:          &id,
			X:           &x,
			Y:           &y,
			Unknown1:    &unknown1,
			Orientation: &orientation,
			SpwanStep:   &spwanStep,
		}
	}

	apiData := SpawnFileAPIData{
		Spawns: apiSpawns,
	}

	_ = utils.WriteJSONResponse(w, apiData)
}

func (s *Server) validateFileUpdateRequest(w http.ResponseWriter, r *http.Request, expectedFileType services.FileType, fileTypeName string) (*fileUpdateContext, bool) {
	userID, ok := utils.GetUserIdFromContext(r.Context())
	if !ok {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusUnauthorized, map[string]interface{}{
			"errorCode": constants.ErrorCodeUnauthorized,
			"context":   "file-system",
			"errors":    []string{"User ID not found in context"},
		})
		return nil, false
	}

	pathParam := r.URL.Query().Get("path")
	if pathParam == "" {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusBadRequest, map[string]interface{}{
			"errorCode": constants.ErrorCodeBadRequest,
			"context":   "file-system",
			"errors":    []string{"Path parameter is required"},
		})
		return nil, false
	}

	cleanPath := filepath.Clean(pathParam)
	info, err := s.fileEditor.Stat(cleanPath)
	if err != nil {
		if s.fileEditor.IsNotExist(err) {
			_ = utils.WriteJSONResponseWithStatus(w, http.StatusNotFound, map[string]interface{}{
				"errorCode": constants.ErrorCodeNotFound,
				"context":   "file-system",
				"errors":    []string{"Path not found"},
			})
			return nil, false
		}

		_ = utils.WriteJSONResponseWithStatus(w, http.StatusInternalServerError, map[string]interface{}{
			"errorCode": constants.ErrorCodeFileReadError,
			"context":   "file-system",
			"errors":    []string{"Cannot read file: " + err.Error()},
		})
		return nil, false
	}

	if info.IsDir() {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusBadRequest, map[string]interface{}{
			"errorCode": constants.ErrorCodePathIsDirectory,
			"context":   "file-system",
			"errors":    []string{"Path is a directory, not a file"},
		})
		return nil, false
	}

	fileType := s.fileEditor.GetFileType(cleanPath, info)
	if fileType != expectedFileType {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusBadRequest, map[string]interface{}{
			"errorCode": constants.ErrorCodeFileNotViewable,
			"context":   "file-system",
			"errors":    []string{"File is not a " + fileTypeName + " file"},
		})
		return nil, false
	}

	if !s.fileEditor.IsFileEditable(cleanPath, info) {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusBadRequest, map[string]interface{}{
			"errorCode": constants.ErrorCodeBadRequest,
			"context":   "file-system",
			"errors":    []string{"File is not editable"},
		})
		return nil, false
	}

	return &fileUpdateContext{
		userID:    userID,
		cleanPath: cleanPath,
		info:      info,
		fileID:    utils.GenerateMD5Hash(cleanPath),
	}, true
}

func (s *Server) createFileRevision(w http.ResponseWriter, ctx *fileUpdateContext, previousData []byte, currentData []byte) (int64, bool) {
	previousHash := utils.CalculateFileHash(previousData)
	currentHash := utils.CalculateFileHash(currentData)

	if previousHash == currentHash {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusBadRequest, map[string]interface{}{
			"errorCode": constants.ErrorCodeBadRequest,
			"context":   "file-system",
			"errors":    []string{"No changes detected. The file content is identical to the existing content."},
		})
		return 0, false
	}

	lockPath, err := s.acquireFileLock(ctx.fileID)
	if err != nil {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusConflict, map[string]interface{}{
			"errorCode": constants.ErrorCodeBadRequest,
			"context":   "file-system",
			"errors":    []string{err.Error()},
		})
		return 0, false
	}

	defer s.releaseFileLock(lockPath)

	tx, err := s.internalDB.BeginTx()
	if err != nil {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusInternalServerError, map[string]interface{}{
			"errorCode": constants.ErrorCodeInternalServerError,
			"context":   "file-system",
			"errors":    []string{"Failed to begin transaction: " + err.Error()},
		})
		return 0, false
	}

	defer func() {
		if err != nil {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				s.log.Error("Failed to rollback transaction", logger.Field{Key: "error", Value: rollbackErr})
			}
		}
	}()

	revisionID, err := s.internalDB.CreateFileRevision(tx, ctx.fileID, ctx.cleanPath, "", previousHash, currentHash, ctx.userID)
	if err != nil {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusInternalServerError, map[string]interface{}{
			"errorCode": constants.ErrorCodeInternalServerError,
			"context":   "file-system",
			"errors":    []string{"Failed to create revision: " + err.Error()},
		})
		return 0, false
	}

	revisionDir := filepath.Join(s.cfg.RevisionsDirectory, ctx.fileID, strconv.FormatInt(revisionID, 10))
	if err = s.fileEditor.MkdirAll(revisionDir, 0755); err != nil {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusInternalServerError, map[string]interface{}{
			"errorCode": constants.ErrorCodeInternalServerError,
			"context":   "file-system",
			"errors":    []string{"Failed to create revision directory: " + err.Error()},
		})
		return 0, false
	}

	epochTime := time.Now().Unix()
	fileName := filepath.Base(ctx.cleanPath)
	revisionFileName := strconv.FormatInt(epochTime, 10) + "_" + fileName
	revisionPath := filepath.Join(revisionDir, revisionFileName)
	if err = s.fileEditor.WriteFile(revisionPath, previousData, 0644); err != nil {
		if removeErr := s.fileEditor.RemoveAll(revisionDir); removeErr != nil {
			s.log.Error("Failed to remove revision directory during cleanup", logger.Field{Key: "error", Value: removeErr})
		}

		_ = utils.WriteJSONResponseWithStatus(w, http.StatusInternalServerError, map[string]interface{}{
			"errorCode": constants.ErrorCodeInternalServerError,
			"context":   "file-system",
			"errors":    []string{"Failed to save revision copy: " + err.Error()},
		})
		return 0, false
	}

	if err = s.internalDB.UpdateFileRevisionPath(tx, revisionID, revisionPath, ctx.userID); err != nil {
		if removeErr := s.fileEditor.Remove(revisionPath); removeErr != nil {
			s.log.Error("Failed to remove revision file during cleanup", logger.Field{Key: "error", Value: removeErr})
		}

		if removeErr := s.fileEditor.RemoveAll(revisionDir); removeErr != nil {
			s.log.Error("Failed to remove revision directory during cleanup", logger.Field{Key: "error", Value: removeErr})
		}

		_ = utils.WriteJSONResponseWithStatus(w, http.StatusInternalServerError, map[string]interface{}{
			"errorCode": constants.ErrorCodeInternalServerError,
			"context":   "file-system",
			"errors":    []string{"Failed to update revision path: " + err.Error()},
		})
		return 0, false
	}

	if err = s.internalDB.UpdateFileRevisionStatus(tx, revisionID, "completed", ctx.userID); err != nil {
		if removeErr := s.fileEditor.Remove(revisionPath); removeErr != nil {
			s.log.Error("Failed to remove revision file during cleanup", logger.Field{Key: "error", Value: removeErr})
		}

		if removeErr := s.fileEditor.RemoveAll(revisionDir); removeErr != nil {
			s.log.Error("Failed to remove revision directory during cleanup", logger.Field{Key: "error", Value: removeErr})
		}

		_ = utils.WriteJSONResponseWithStatus(w, http.StatusInternalServerError, map[string]interface{}{
			"errorCode": constants.ErrorCodeInternalServerError,
			"context":   "file-system",
			"errors":    []string{"Failed to update revision status: " + err.Error()},
		})
		return 0, false
	}

	if err = tx.Commit(); err != nil {
		if removeErr := s.fileEditor.Remove(revisionPath); removeErr != nil {
			s.log.Error("Failed to remove revision file during cleanup", logger.Field{Key: "error", Value: removeErr})
		}

		if removeErr := s.fileEditor.RemoveAll(revisionDir); removeErr != nil {
			s.log.Error("Failed to remove revision directory during cleanup", logger.Field{Key: "error", Value: removeErr})
		}

		_ = utils.WriteJSONResponseWithStatus(w, http.StatusInternalServerError, map[string]interface{}{
			"errorCode": constants.ErrorCodeInternalServerError,
			"context":   "file-system",
			"errors":    []string{"Failed to commit transaction: " + err.Error()},
		})
		return 0, false
	}

	return revisionID, true
}

func (s *Server) handleUpdateNPCFile(w http.ResponseWriter, r *http.Request) {
	if !s.requireUserPermission(w, r, permissions.ActionEditFiles) {
		return
	}

	ctx, ok := s.validateFileUpdateRequest(w, r, services.FileTypeNPC, "NPC")
	if !ok {
		return
	}

	var req NPCFileAPIData
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusBadRequest, map[string]interface{}{
			"errorCode": constants.ErrorCodeBadRequest,
			"context":   "file-system",
			"errors":    []string{"Invalid request body: " + err.Error()},
		})
		return
	}

	validate := validator.New()
	if err := validate.Struct(req); err != nil {
		var errors []string
		for _, err := range err.(validator.ValidationErrors) {
			errors = append(errors, err.Field()+" is required")
		}

		_ = utils.WriteJSONResponseWithStatus(w, http.StatusBadRequest, map[string]interface{}{
			"errorCode": constants.ErrorCodeBadRequest,
			"context":   "file-system",
			"errors":    errors,
		})
		return
	}

	previousData, err := s.fileEditor.ReadFile(ctx.cleanPath)
	if err != nil {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusInternalServerError, map[string]interface{}{
			"errorCode": constants.ErrorCodeFileReadError,
			"context":   "file-system",
			"errors":    []string{"Failed to read file: " + err.Error()},
		})
		return
	}

	var nameBytes [0x14]byte
	copy(nameBytes[:], []byte(req.Name))

	var attacks [0x3]services.NPCAttack
	copy(attacks[:], req.Attacks)

	npcData := &services.NPCFileData{
		Name:                nameBytes,
		Id:                  *req.Id,
		RespawnRate:         *req.RespawnRate,
		AttackTypeInfo:      *req.AttackTypeInfo,
		TargetSelectionInfo: *req.TargetSelectionInfo,
		Defense:             *req.Defense,
		AdditionalDefense:   *req.AdditionalDefense,
		Attacks:             attacks,
		AttackSpeedLow:      *req.AttackSpeedLow,
		AttackSpeedHigh:     *req.AttackSpeedHigh,
		MovementSpeed:       *req.MovementSpeed,
		Level:               *req.Level,
		PlayerExp:           *req.PlayerExp,
		Appearance:          *req.Appearance,
		HP:                  *req.HP,
		BlueAttackDefense:   *req.BlueAttackDefense,
		RedAttackDefense:    *req.RedAttackDefense,
		GreyAttackDefense:   *req.GreyAttackDefense,
		MercenaryExp:        *req.MercenaryExp,
		Unknown:             0,
	}

	var currentDataBuffer bytes.Buffer
	if err := binary.Write(&currentDataBuffer, binary.LittleEndian, npcData); err != nil {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusInternalServerError, map[string]interface{}{
			"errorCode": constants.ErrorCodeInternalServerError,
			"context":   "file-system",
			"errors":    []string{"Failed to serialize NPC data: " + err.Error()},
		})
		return
	}

	currentData := currentDataBuffer.Bytes()

	revisionID, ok := s.createFileRevision(w, ctx, previousData, currentData)
	if !ok {
		return
	}

	if err = s.fileEditor.WriteNPCFileData(ctx.cleanPath, npcData); err != nil {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusInternalServerError, map[string]interface{}{
			"errorCode": constants.ErrorCodeInternalServerError,
			"context":   "file-system",
			"errors":    []string{"Failed to write file: " + err.Error()},
		})
		return
	}

	_ = utils.WriteJSONResponse(w, map[string]interface{}{
		"message":     "File updated successfully",
		"revision_id": revisionID,
	})
}

func (s *Server) handleUpdateTextFile(w http.ResponseWriter, r *http.Request) {
	if !s.requireUserPermission(w, r, permissions.ActionEditFiles) {
		return
	}

	ctx, ok := s.validateFileUpdateRequest(w, r, services.FileTypeText, "text")
	if !ok {
		return
	}

	var req TextFileAPIData
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusBadRequest, map[string]interface{}{
			"errorCode": constants.ErrorCodeBadRequest,
			"context":   "file-system",
			"errors":    []string{"Invalid request body: " + err.Error()},
		})
		return
	}

	validate := validator.New()
	if err := validate.Struct(req); err != nil {
		var errors []string
		for _, err := range err.(validator.ValidationErrors) {
			errors = append(errors, err.Field()+" is required")
		}
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusBadRequest, map[string]interface{}{
			"errorCode": constants.ErrorCodeBadRequest,
			"context":   "file-system",
			"errors":    errors,
		})
		return
	}

	previousData, err := s.fileEditor.ReadFile(ctx.cleanPath)
	if err != nil {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusInternalServerError, map[string]interface{}{
			"errorCode": constants.ErrorCodeFileReadError,
			"context":   "file-system",
			"errors":    []string{"Failed to read file: " + err.Error()},
		})
		return
	}

	currentData := []byte(req.Content)

	revisionID, ok := s.createFileRevision(w, ctx, previousData, currentData)
	if !ok {
		return
	}

	if err = s.fileEditor.WriteTextFileData(ctx.cleanPath, req.Content); err != nil {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusInternalServerError, map[string]interface{}{
			"errorCode": constants.ErrorCodeInternalServerError,
			"context":   "file-system",
			"errors":    []string{"Failed to write file: " + err.Error()},
		})
		return
	}

	_ = utils.WriteJSONResponse(w, map[string]interface{}{
		"message":     "File updated successfully",
		"revision_id": revisionID,
	})
}

func (s *Server) handleUpdateSpawnFile(w http.ResponseWriter, r *http.Request) {
	if !s.requireUserPermission(w, r, permissions.ActionEditFiles) {
		return
	}

	ctx, ok := s.validateFileUpdateRequest(w, r, services.FileTypeSpawn, "spawn")
	if !ok {
		return
	}

	var req SpawnFileAPIData
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusBadRequest, map[string]interface{}{
			"errorCode": constants.ErrorCodeBadRequest,
			"context":   "file-system",
			"errors":    []string{"Invalid request body: " + err.Error()},
		})
		return
	}

	validate := validator.New()
	if err := validate.Struct(req); err != nil {
		var errors []string
		for _, err := range err.(validator.ValidationErrors) {
			errors = append(errors, err.Field()+" is required")
		}

		_ = utils.WriteJSONResponseWithStatus(w, http.StatusBadRequest, map[string]interface{}{
			"errorCode": constants.ErrorCodeBadRequest,
			"context":   "file-system",
			"errors":    errors,
		})
		return
	}

	previousData, err := s.fileEditor.ReadFile(ctx.cleanPath)
	if err != nil {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusInternalServerError, map[string]interface{}{
			"errorCode": constants.ErrorCodeFileReadError,
			"context":   "file-system",
			"errors":    []string{"Failed to read file: " + err.Error()},
		})
		return
	}

	spawnData := make([]services.NPCSpawnData, len(req.Spawns))
	for i, spawn := range req.Spawns {
		spawnData[i] = services.NPCSpawnData{
			Id:          *spawn.Id,
			X:           *spawn.X,
			Y:           *spawn.Y,
			Unknown1:    *spawn.Unknown1,
			Orientation: *spawn.Orientation,
			SpwanStep:   *spawn.SpwanStep,
		}
	}

	var currentDataBuffer bytes.Buffer
	if err := binary.Write(&currentDataBuffer, binary.LittleEndian, spawnData); err != nil {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusInternalServerError, map[string]interface{}{
			"errorCode": constants.ErrorCodeInternalServerError,
			"context":   "file-system",
			"errors":    []string{"Failed to serialize spawn data: " + err.Error()},
		})
		return
	}

	currentData := currentDataBuffer.Bytes()

	revisionID, ok := s.createFileRevision(w, ctx, previousData, currentData)
	if !ok {
		return
	}

	if err = s.fileEditor.WriteSpawnFileData(ctx.cleanPath, spawnData); err != nil {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusInternalServerError, map[string]interface{}{
			"errorCode": constants.ErrorCodeInternalServerError,
			"context":   "file-system",
			"errors":    []string{"Failed to write file: " + err.Error()},
		})
		return
	}

	_ = utils.WriteJSONResponse(w, map[string]interface{}{
		"message":     "File updated successfully",
		"revision_id": revisionID,
	})
}

func (s *Server) handleQuestFileData(w http.ResponseWriter, r *http.Request) {
	pathParam := r.URL.Query().Get("path")
	if pathParam == "" {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusBadRequest, map[string]interface{}{
			"errorCode": constants.ErrorCodeBadRequest,
			"context":   "file-system",
			"errors":    []string{"Path parameter is required"},
		})
		return
	}

	cleanPath := filepath.Clean(pathParam)
	info, err := s.fileEditor.Stat(cleanPath)
	if err != nil {
		if s.fileEditor.IsNotExist(err) {
			_ = utils.WriteJSONResponseWithStatus(w, http.StatusNotFound, map[string]interface{}{
				"errorCode": constants.ErrorCodeNotFound,
				"context":   "file-system",
				"errors":    []string{"Path not found"},
			})
			return
		}

		_ = utils.WriteJSONResponseWithStatus(w, http.StatusInternalServerError, map[string]interface{}{
			"errorCode": constants.ErrorCodeFileReadError,
			"context":   "file-system",
			"errors":    []string{"Cannot read file: " + err.Error()},
		})
		return
	}

	if info.IsDir() {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusBadRequest, map[string]interface{}{
			"errorCode": constants.ErrorCodePathIsDirectory,
			"context":   "file-system",
			"errors":    []string{"Path is a directory, not a file"},
		})
		return
	}

	fileType := s.fileEditor.GetFileType(cleanPath, info)
	if fileType != services.FileTypeQuest {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusBadRequest, map[string]interface{}{
			"errorCode": constants.ErrorCodeFileNotViewable,
			"context":   "file-system",
			"errors":    []string{"File is not a quest file"},
		})
		return
	}

	qf, err := s.fileEditor.ReadQuestFileData(cleanPath)
	if err != nil {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusInternalServerError, map[string]interface{}{
			"errorCode": constants.ErrorCodeFileReadError,
			"context":   "file-system",
			"errors":    []string{"Failed to read quest file data: " + err.Error()},
		})
		return
	}

	_ = utils.WriteJSONResponse(w, questFileToAPIData(qf))
}

const maxQuestObjectiveNameLen = 255

func getQuestObjectiveTypeName(objectiveType uint8) string {
	switch objectiveType {
	case questfile.TypeKILL:
		return "KILL"
	case questfile.TypeQUESTITEM:
		return "QUESTITEM"
	case questfile.TypeBRINGNPC:
		return "BRINGNPC"
	case questfile.TypeDROP:
		return "DROP"
	case questfile.TypeFIND:
		return "FIND"
	case questfile.TypeUnused:
		return "UNUSED"
	default:
		return "UNKNOWN"
	}
}

func questFileToAPIData(qf questfile.QuestFile) QuestFileAPIData {
	h := &qf.Header
	questID := h.QuestID()
	giverNPC := h.GivenNPCID()
	targetNPC := binary.LittleEndian.Uint16(h.TargetNPCBlock[:2])
	minLevel := h.MinLevel
	maxLevel := h.MaxLevel
	flags := h.QuestFlags
	expReward := h.EXP
	woonzReward := h.Woonz
	loreReward := h.Lore

	rewardItems := make([]*uint16, 4)
	rewardCounts := make([]*uint8, 4)
	setQuestRewardAPIData(rewardItems, rewardCounts, 0, binary.LittleEndian.Uint16(h.RewardSlot1[:2]), h.Count1)
	setQuestRewardAPIData(rewardItems, rewardCounts, 1, binary.LittleEndian.Uint16(h.RewardSlot2[:2]), h.Count2)
	setQuestRewardAPIData(rewardItems, rewardCounts, 2, binary.LittleEndian.Uint16(h.RewardSlot3[:2]), h.Count3)

	objectives := make([]ObjectiveAPIData, questfile.NumObjectives)
	for i := range qf.Objectives {
		objectives[i] = questObjectiveToAPIData(qf.Objectives[i])
	}

	continuations := make([]*uint32, 3)
	for i := range qf.Continuation {
		continuation := qf.Continuation[i]
		if continuation != questfile.UnusedContinuation {
			continuations[i] = &continuation
		}
	}

	return QuestFileAPIData{
		QuestID:       &questID,
		GiverNPC:      &giverNPC,
		TargetNPC:     &targetNPC,
		MinLevel:      &minLevel,
		MaxLevel:      &maxLevel,
		Flags:         &flags,
		RewardItems:   rewardItems,
		RewardCounts:  rewardCounts,
		ExpReward:     &expReward,
		WoonzReward:   &woonzReward,
		LoreReward:    &loreReward,
		Objectives:    objectives,
		Continuations: continuations,
	}
}

func setQuestRewardAPIData(items []*uint16, counts []*uint8, index int, item uint16, count uint8) {
	if item == questfile.UnusedRewardItemCode {
		return
	}

	items[index] = &item
	counts[index] = &count
}

func questObjectiveToAPIData(objective questfile.Objective) ObjectiveAPIData {
	blk := &objective.Block
	objType := blk[0]
	mapID := binary.LittleEndian.Uint16(blk[4:6])
	locationID := binary.LittleEndian.Uint16(blk[8:10])
	locationX := uint8(locationID)
	locationY := uint8(locationID >> 8)
	radius := blk[12]
	targetID := binary.LittleEndian.Uint16(blk[16:18])
	killCount := binary.LittleEndian.Uint16(blk[20:22])
	questItemID := binary.LittleEndian.Uint16(blk[24:26])
	requiredItemCount := binary.LittleEndian.Uint16(blk[56:58])
	dropItems := make([]*uint16, 3)
	dropProbs := make([]*uint8, 3)

	for j := range 3 {
		item := binary.LittleEndian.Uint16(blk[28+j*2:][:2])
		probability := blk[76+j*4]
		dropItems[j] = &item
		dropProbs[j] = &probability
	}

	return ObjectiveAPIData{
		Type:              &objType,
		TypeName:          getQuestObjectiveTypeName(objType),
		MapID:             &mapID,
		Location:          &QuestLocationAPIData{X: &locationX, Y: &locationY},
		Radius:            &radius,
		TargetID:          &targetID,
		KillCount:         &killCount,
		QuestItemID:       &questItemID,
		DropItems:         dropItems,
		RequiredItemCount: &requiredItemCount,
		DropProbs:         dropProbs,
		Name:              string(objective.Name),
		IsUnused:          objType == questfile.TypeUnused,
	}
}

func applyQuestFileAPIData(qf *questfile.QuestFile, req QuestFileAPIData) {
	qf.Header.SetQuestID(*req.QuestID)
	qf.Header.SetGivenNPCID(*req.GiverNPC)
	binary.LittleEndian.PutUint16(qf.Header.TargetNPCBlock[:2], *req.TargetNPC)
	qf.Header.MinLevel = *req.MinLevel
	qf.Header.MaxLevel = *req.MaxLevel
	qf.Header.QuestFlags = *req.Flags
	qf.Header.EXP = *req.ExpReward
	qf.Header.Woonz = *req.WoonzReward
	qf.Header.Lore = *req.LoreReward

	setQuestRewardItem(qf.Header.RewardSlot1[:], req.RewardItems[0])
	setQuestRewardItem(qf.Header.RewardSlot2[:], req.RewardItems[1])
	setQuestRewardItem(qf.Header.RewardSlot3[:], req.RewardItems[2])
	setQuestRewardCount(&qf.Header.Count1, req.RewardCounts[0])
	setQuestRewardCount(&qf.Header.Count2, req.RewardCounts[1])
	setQuestRewardCount(&qf.Header.Count3, req.RewardCounts[2])

	for i := range questfile.NumObjectives {
		applyQuestObjectiveAPIData(&qf.Objectives[i], req.Objectives[i])
	}

	for i := range qf.Continuation {
		if req.Continuations[i] != nil {
			qf.Continuation[i] = *req.Continuations[i]
		} else {
			qf.Continuation[i] = questfile.UnusedContinuation
		}
	}
}

func setQuestRewardItem(slot []byte, item *uint16) {
	if item != nil {
		binary.LittleEndian.PutUint16(slot[:2], *item)
		return
	}

	binary.LittleEndian.PutUint16(slot[:2], questfile.UnusedRewardItemCode)
}

func setQuestRewardCount(count *uint8, value *uint8) {
	if value != nil {
		*count = *value
	}
}

func applyQuestObjectiveAPIData(objective *questfile.Objective, objAPI ObjectiveAPIData) {
	objType := *objAPI.Type
	if objType == questfile.TypeUnused || objAPI.IsUnused {
		objective.Block = unusedQuestObjectiveBlock()
		objective.Name = nil
		return
	}

	blk := &objective.Block
	blk[0] = objType
	binary.LittleEndian.PutUint16(blk[4:6], *objAPI.MapID)
	locationID := uint16(*objAPI.Location.X) | (uint16(*objAPI.Location.Y) << 8)
	binary.LittleEndian.PutUint16(blk[8:10], locationID)
	blk[12] = *objAPI.Radius
	binary.LittleEndian.PutUint16(blk[16:18], *objAPI.TargetID)
	binary.LittleEndian.PutUint16(blk[20:22], *objAPI.KillCount)
	binary.LittleEndian.PutUint16(blk[24:26], *objAPI.QuestItemID)
	binary.LittleEndian.PutUint16(blk[56:58], *objAPI.RequiredItemCount)

	for j := range 3 {
		if objAPI.DropItems[j] != nil {
			binary.LittleEndian.PutUint16(blk[28+j*2:][:2], *objAPI.DropItems[j])
		} else {
			binary.LittleEndian.PutUint16(blk[28+j*2:][:2], questfile.UnusedRewardItemCode)
		}

		if objAPI.DropProbs[j] != nil {
			blk[76+j*4] = *objAPI.DropProbs[j]
		} else {
			blk[76+j*4] = 0xff
		}
	}

	if objType != questfile.TypeDROP && objType != questfile.TypeFIND {
		blk[92] = 0
		objective.Name = nil
		return
	}

	nameLen := min(len(objAPI.Name), maxQuestObjectiveNameLen)
	blk[92] = byte(nameLen)
	if nameLen == 0 {
		objective.Name = nil
		return
	}

	objective.Name = []byte(objAPI.Name[:nameLen])
}

func unusedQuestObjectiveBlock() [questfile.ObjectiveBlockSize]byte {
	var block [questfile.ObjectiveBlockSize]byte
	for i := range block {
		block[i] = 0xff
	}

	block[9] = 0xfe
	block[92] = 0
	block[93] = 0
	block[94] = 0
	block[95] = 0
	return block
}

func (s *Server) handleUpdateQuestFile(w http.ResponseWriter, r *http.Request) {
	if !s.requireUserPermission(w, r, permissions.ActionEditFiles) {
		return
	}

	ctx, ok := s.validateFileUpdateRequest(w, r, services.FileTypeQuest, "quest")
	if !ok {
		return
	}

	var req QuestFileAPIData
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusBadRequest, map[string]interface{}{
			"errorCode": constants.ErrorCodeBadRequest,
			"context":   "file-system",
			"errors":    []string{"Invalid request body: " + err.Error()},
		})
		return
	}

	validate := validator.New()
	if err := validate.Struct(req); err != nil {
		var errors []string
		for _, err := range err.(validator.ValidationErrors) {
			errors = append(errors, err.Field()+" is required")
		}

		_ = utils.WriteJSONResponseWithStatus(w, http.StatusBadRequest, map[string]interface{}{
			"errorCode": constants.ErrorCodeBadRequest,
			"context":   "file-system",
			"errors":    errors,
		})
		return
	}

	if validationErrs := validateQuestFileRequest(req); len(validationErrs) > 0 {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusBadRequest, map[string]interface{}{
			"errorCode": constants.ErrorCodeBadRequest,
			"context":   "file-system",
			"errors":    validationErrs,
		})
		return
	}

	previousData, err := s.fileEditor.ReadFile(ctx.cleanPath)
	if err != nil {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusInternalServerError, map[string]interface{}{
			"errorCode": constants.ErrorCodeFileReadError,
			"context":   "file-system",
			"errors":    []string{"Failed to read file: " + err.Error()},
		})
		return
	}

	qf, err := s.fileEditor.ReadQuestFileData(ctx.cleanPath)
	if err != nil {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusInternalServerError, map[string]interface{}{
			"errorCode": constants.ErrorCodeFileReadError,
			"context":   "file-system",
			"errors":    []string{"Failed to read quest file data: " + err.Error()},
		})
		return
	}

	applyQuestFileAPIData(&qf, req)

	var currentDataBuffer bytes.Buffer
	if err := questfile.Write(&currentDataBuffer, qf); err != nil {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusInternalServerError, map[string]interface{}{
			"errorCode": constants.ErrorCodeInternalServerError,
			"context":   "file-system",
			"errors":    []string{"Failed to serialize quest data: " + err.Error()},
		})
		return
	}

	currentData := currentDataBuffer.Bytes()
	if _, err := questfile.Read(bytes.NewReader(currentData)); err != nil {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusInternalServerError, map[string]interface{}{
			"errorCode": constants.ErrorCodeInternalServerError,
			"context":   "file-system",
			"errors":    []string{"Failed to validate quest data: " + err.Error()},
		})
		return
	}

	revisionID, ok := s.createFileRevision(w, ctx, previousData, currentData)
	if !ok {
		return
	}

	if err = s.fileEditor.WriteQuestFileData(ctx.cleanPath, qf); err != nil {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusInternalServerError, map[string]interface{}{
			"errorCode": constants.ErrorCodeInternalServerError,
			"context":   "file-system",
			"errors":    []string{"Failed to write file: " + err.Error()},
		})
		return
	}

	_ = utils.WriteJSONResponse(w, map[string]interface{}{
		"message":     "File updated successfully",
		"revision_id": revisionID,
	})
}

func (s *Server) acquireFileLock(fileID string) (string, error) {
	locksDir := filepath.Join(s.cfg.RevisionsDirectory, "locks")
	if err := s.fileEditor.MkdirAll(locksDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create locks directory: %w", err)
	}

	lockPath := filepath.Join(locksDir, fileID+".lock")
	lockFile, err := s.fileEditor.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if err != nil {
		if s.fileEditor.IsExist(err) {
			return "", fmt.Errorf("file is currently being edited by another process")
		}
		return "", fmt.Errorf("failed to create lock file: %w", err)
	}

	if err := lockFile.Close(); err != nil {
		s.log.Error("Failed to close lock file", logger.Field{Key: "error", Value: err})
	}

	return lockPath, nil
}

func (s *Server) handleRevertFile(w http.ResponseWriter, r *http.Request) {
	if !s.requireUserPermission(w, r, permissions.ActionRevertFiles) {
		return
	}

	userID, ok := utils.GetUserIdFromContext(r.Context())
	if !ok {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusUnauthorized, map[string]interface{}{
			"errorCode": constants.ErrorCodeUnauthorized,
			"context":   "file-system",
			"errors":    []string{"User ID not found in context"},
		})
		return
	}

	pathParam := r.URL.Query().Get("path")
	if pathParam == "" {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusBadRequest, map[string]interface{}{
			"errorCode": constants.ErrorCodeBadRequest,
			"context":   "file-system",
			"errors":    []string{"Path parameter is required"},
		})
		return
	}

	cleanPath := filepath.Clean(pathParam)
	info, err := s.fileEditor.Stat(cleanPath)
	if err != nil {
		if s.fileEditor.IsNotExist(err) {
			_ = utils.WriteJSONResponseWithStatus(w, http.StatusNotFound, map[string]interface{}{
				"errorCode": constants.ErrorCodeNotFound,
				"context":   "file-system",
				"errors":    []string{"Path not found"},
			})
			return
		}

		_ = utils.WriteJSONResponseWithStatus(w, http.StatusInternalServerError, map[string]interface{}{
			"errorCode": constants.ErrorCodeFileReadError,
			"context":   "file-system",
			"errors":    []string{"Cannot read file: " + err.Error()},
		})
		return
	}

	if info.IsDir() {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusBadRequest, map[string]interface{}{
			"errorCode": constants.ErrorCodePathIsDirectory,
			"context":   "file-system",
			"errors":    []string{"Path is a directory, not a file"},
		})
		return
	}

	fileID := utils.GenerateMD5Hash(cleanPath)

	lockPath, err := s.acquireFileLock(fileID)
	if err != nil {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusConflict, map[string]interface{}{
			"errorCode": constants.ErrorCodeBadRequest,
			"context":   "file-system",
			"errors":    []string{err.Error()},
		})
		return
	}

	defer s.releaseFileLock(lockPath)

	revision, err := s.internalDB.GetLastCompletedFileRevision(fileID)
	if err != nil {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusInternalServerError, map[string]interface{}{
			"errorCode": constants.ErrorCodeInternalServerError,
			"context":   "file-system",
			"errors":    []string{"Failed to query file revisions: " + err.Error()},
		})
		return
	}

	if revision == nil {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusNotFound, map[string]interface{}{
			"errorCode": constants.ErrorCodeNotFound,
			"context":   "file-system",
			"errors":    []string{"No completed revisions found for this file"},
		})
		return
	}

	if _, err := s.fileEditor.Stat(revision.RevisionPath); s.fileEditor.IsNotExist(err) {
		tx, err := s.internalDB.BeginTx()
		if err != nil {
			_ = utils.WriteJSONResponseWithStatus(w, http.StatusInternalServerError, map[string]interface{}{
				"errorCode": constants.ErrorCodeInternalServerError,
				"context":   "file-system",
				"errors":    []string{"Failed to begin transaction: " + err.Error()},
			})
			return
		}

		defer func() {
			if err != nil {
				if rollbackErr := tx.Rollback(); rollbackErr != nil {
					s.log.Error("Failed to rollback transaction", logger.Field{Key: "error", Value: rollbackErr})
				}
			}
		}()

		if err = s.internalDB.UpdateFileRevisionStatus(tx, revision.ID, "corrupted", userID); err != nil {
			_ = utils.WriteJSONResponseWithStatus(w, http.StatusInternalServerError, map[string]interface{}{
				"errorCode": constants.ErrorCodeInternalServerError,
				"context":   "file-system",
				"errors":    []string{"Failed to update revision status: " + err.Error()},
			})
			return
		}

		if err = tx.Commit(); err != nil {
			_ = utils.WriteJSONResponseWithStatus(w, http.StatusInternalServerError, map[string]interface{}{
				"errorCode": constants.ErrorCodeInternalServerError,
				"context":   "file-system",
				"errors":    []string{"Failed to commit transaction: " + err.Error()},
			})
			return
		}

		_ = utils.WriteJSONResponseWithStatus(w, http.StatusInternalServerError, map[string]interface{}{
			"errorCode": constants.ErrorCodeInternalServerError,
			"context":   "file-system",
			"errors":    []string{"Revision file is missing or corrupted"},
		})
		return
	}

	revisionData, err := s.fileEditor.ReadFile(revision.RevisionPath)
	if err != nil {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusInternalServerError, map[string]interface{}{
			"errorCode": constants.ErrorCodeFileReadError,
			"context":   "file-system",
			"errors":    []string{"Failed to read revision file: " + err.Error()},
		})
		return
	}

	tx, err := s.internalDB.BeginTx()
	if err != nil {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusInternalServerError, map[string]interface{}{
			"errorCode": constants.ErrorCodeInternalServerError,
			"context":   "file-system",
			"errors":    []string{"Failed to begin transaction: " + err.Error()},
		})
		return
	}

	defer func() {
		if err != nil {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				s.log.Error("Failed to rollback transaction", logger.Field{Key: "error", Value: rollbackErr})
			}
		}
	}()

	if err = s.internalDB.UpdateFileRevisionStatus(tx, revision.ID, "reverted", userID); err != nil {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusInternalServerError, map[string]interface{}{
			"errorCode": constants.ErrorCodeInternalServerError,
			"context":   "file-system",
			"errors":    []string{"Failed to update revision status: " + err.Error()},
		})
		return
	}

	if err = tx.Commit(); err != nil {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusInternalServerError, map[string]interface{}{
			"errorCode": constants.ErrorCodeInternalServerError,
			"context":   "file-system",
			"errors":    []string{"Failed to commit transaction: " + err.Error()},
		})
		return
	}

	err = nil

	if err = s.fileEditor.WriteFile(cleanPath, revisionData, 0644); err != nil {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusInternalServerError, map[string]interface{}{
			"errorCode": constants.ErrorCodeInternalServerError,
			"context":   "file-system",
			"errors":    []string{"Failed to write file: " + err.Error()},
		})
		return
	}

	_ = utils.WriteJSONResponse(w, map[string]interface{}{
		"message":     "File reverted successfully",
		"revision_id": revision.ID,
	})
}

func (s *Server) handleRevisionSummary(w http.ResponseWriter, r *http.Request) {
	pathParam := r.URL.Query().Get("path")
	if pathParam == "" {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusBadRequest, map[string]interface{}{
			"errorCode": constants.ErrorCodeBadRequest,
			"context":   "file-system",
			"errors":    []string{"Path parameter is required"},
		})
		return
	}

	cleanPath := filepath.Clean(pathParam)
	info, err := s.fileEditor.Stat(cleanPath)
	if err != nil {
		if s.fileEditor.IsNotExist(err) {
			_ = utils.WriteJSONResponseWithStatus(w, http.StatusNotFound, map[string]interface{}{
				"errorCode": constants.ErrorCodeNotFound,
				"context":   "file-system",
				"errors":    []string{"Path not found"},
			})
			return
		}

		_ = utils.WriteJSONResponseWithStatus(w, http.StatusInternalServerError, map[string]interface{}{
			"errorCode": constants.ErrorCodeFileReadError,
			"context":   "file-system",
			"errors":    []string{"Cannot read file: " + err.Error()},
		})
		return
	}

	if info.IsDir() {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusBadRequest, map[string]interface{}{
			"errorCode": constants.ErrorCodePathIsDirectory,
			"context":   "file-system",
			"errors":    []string{"Path is a directory, not a file"},
		})
		return
	}

	fileID := utils.GenerateMD5Hash(cleanPath)
	summary, err := s.internalDB.GetRevisionSummary(fileID)
	if err != nil {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusInternalServerError, map[string]interface{}{
			"errorCode": constants.ErrorCodeInternalServerError,
			"context":   "file-system",
			"errors":    []string{"Failed to get revision summary: " + err.Error()},
		})
		return
	}

	_ = utils.WriteJSONResponse(w, summary)
}

func (s *Server) releaseFileLock(lockPath string) {
	if lockPath != "" {
		if err := s.fileEditor.Remove(lockPath); err != nil {
			s.log.Error("Failed to remove lock file", logger.Field{Key: "error", Value: err})
		}
	}
}

func validateQuestFileRequest(req QuestFileAPIData) []string {
	var errs []string
	if *req.MinLevel != 0xff && *req.MaxLevel != 0xff && *req.MinLevel > *req.MaxLevel {
		errs = append(errs, "min_level must be less than or equal to max_level")
	}
	for i := 0; i < 7 && i < len(req.Objectives); i++ {
		obj := req.Objectives[i]
		t := *obj.Type
		isUnused := t == questfile.TypeUnused || obj.IsUnused
		if t > questfile.TypeFIND && t != questfile.TypeUnused {
			errs = append(errs, fmt.Sprintf("objectives[%d]: objective type must be 0-4 (KILL, QUESTITEM, BRINGNPC, DROP, FIND) or 255 (UNUSED)", i))
		}
		if obj.IsUnused && t != questfile.TypeUnused {
			errs = append(errs, fmt.Sprintf("objectives[%d]: unused objectives must use type 255", i))
		}
		if !isUnused && t <= questfile.TypeBRINGNPC && len(obj.Name) > 0 {
			errs = append(errs, fmt.Sprintf("objectives[%d]: name must be empty for objective type %d (KILL/QUESTITEM/BRINGNPC)", i, t))
		}
		if (t == questfile.TypeDROP || t == questfile.TypeFIND) && len(obj.Name) > maxQuestObjectiveNameLen {
			errs = append(errs, fmt.Sprintf("objectives[%d]: name length must not exceed %d bytes", i, maxQuestObjectiveNameLen))
		}
	}
	return errs
}

type FileNode struct {
	ID            string            `json:"id"`
	Name          string            `json:"name"`
	Kind          string            `json:"kind"`
	Depth         int               `json:"depth"`
	LastModified  *time.Time        `json:"last_modified,omitempty"`
	Permissions   string            `json:"permissions,omitempty"`
	FileSize      int64             `json:"file_size,omitempty"`
	FileExtension string            `json:"file_extension,omitempty"`
	FileType      services.FileType `json:"file_type,omitempty"`
	MimeType      string            `json:"mime_type,omitempty"`
	IsEditable    bool              `json:"is_editable"`
	IsViewable    bool              `json:"is_viewable"`
	APIEndpoint   string            `json:"api_endpoint,omitempty"`
	Children      []*FileNode       `json:"children"`
}

type FileTreeResponse struct {
	OS       string    `json:"os"`
	FileTree *FileNode `json:"file_tree"`
}

type NPCFileAPIData struct {
	Name                string               `json:"name" validate:"required"`
	Id                  *uint16              `json:"id" validate:"required"`
	RespawnRate         *uint16              `json:"respawn_rate" validate:"required"`
	AttackTypeInfo      *byte                `json:"attack_type_info" validate:"required"`
	TargetSelectionInfo *byte                `json:"target_selection_info" validate:"required"`
	Defense             *byte                `json:"defense" validate:"required"`
	AdditionalDefense   *byte                `json:"additional_defense" validate:"required"`
	Attacks             []services.NPCAttack `json:"attacks" validate:"required,len=3"`
	AttackSpeedLow      *uint16              `json:"attack_speed_low" validate:"required"`
	AttackSpeedHigh     *uint16              `json:"attack_speed_high" validate:"required"`
	MovementSpeed       *uint32              `json:"movement_speed" validate:"required"`
	Level               *byte                `json:"level" validate:"required"`
	PlayerExp           *uint16              `json:"player_exp" validate:"required"`
	Appearance          *byte                `json:"appearance" validate:"required"`
	HP                  *uint32              `json:"hp" validate:"required"`
	BlueAttackDefense   *uint16              `json:"blue_attack_defense" validate:"required"`
	RedAttackDefense    *uint16              `json:"red_attack_defense" validate:"required"`
	GreyAttackDefense   *uint16              `json:"grey_attack_defense" validate:"required"`
	MercenaryExp        *uint16              `json:"mercenary_exp" validate:"required"`
}

type TextFileAPIData struct {
	Content string `json:"content" validate:"required"`
}

type SpawnFileAPIData struct {
	Spawns []NPCSpawnAPIData `json:"spawns" validate:"required"`
}

type NPCSpawnAPIData struct {
	Id          *uint16 `json:"id" validate:"required"`
	X           *byte   `json:"x" validate:"required"`
	Y           *byte   `json:"y" validate:"required"`
	Unknown1    *uint16 `json:"unknown1" validate:"required"`
	Orientation *byte   `json:"orientation" validate:"required"`
	SpwanStep   *byte   `json:"spwan_step" validate:"required"`
}

type QuestFileAPIData struct {
	QuestID       *uint16            `json:"quest_id" validate:"required"`
	GiverNPC      *uint16            `json:"giver_npc" validate:"required"`
	TargetNPC     *uint16            `json:"target_npc" validate:"required"`
	MinLevel      *uint8             `json:"min_level" validate:"required"`
	MaxLevel      *uint8             `json:"max_level" validate:"required"`
	Flags         *uint32            `json:"flags" validate:"required"`
	RewardItems   []*uint16          `json:"reward_items" validate:"required,len=4"`
	RewardCounts  []*uint8           `json:"reward_counts" validate:"required,len=4"`
	ExpReward     *uint32            `json:"exp_reward" validate:"required"`
	WoonzReward   *uint32            `json:"woonz_reward" validate:"required"`
	LoreReward    *uint32            `json:"lore_reward" validate:"required"`
	Objectives    []ObjectiveAPIData `json:"objectives" validate:"required,len=7"`
	Continuations []*uint32          `json:"continuations" validate:"required,len=3"`
}

type ObjectiveAPIData struct {
	Type              *uint8                `json:"type" validate:"required"`
	TypeName          string                `json:"type_name"`
	MapID             *uint16               `json:"map_id" validate:"required"`
	Location          *QuestLocationAPIData `json:"location" validate:"required"`
	Radius            *uint8                `json:"radius" validate:"required"`
	TargetID          *uint16               `json:"target_id" validate:"required"`
	KillCount         *uint16               `json:"kill_count" validate:"required"`
	QuestItemID       *uint16               `json:"quest_item_id" validate:"required"`
	DropItems         []*uint16             `json:"drop_items" validate:"required,len=3"`
	RequiredItemCount *uint16               `json:"required_item_count" validate:"required"`
	DropProbs         []*uint8              `json:"drop_probs" validate:"required,len=3"`
	Name              string                `json:"name"`
	IsUnused          bool                  `json:"is_unused"`
}

type QuestLocationAPIData struct {
	X *uint8 `json:"x" validate:"required"`
	Y *uint8 `json:"y" validate:"required"`
}

type fileUpdateContext struct {
	userID    int64
	cleanPath string
	info      fs.FileInfo
	fileID    string
}
