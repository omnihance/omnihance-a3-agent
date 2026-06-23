package server

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	externalUtils "github.com/cyberinferno/go-utils/utils"
	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/project-agonyl/agonyl-utils-go/dropfile"
	"github.com/project-agonyl/agonyl-utils-go/itemcombinationdata"
	"github.com/project-agonyl/agonyl-utils-go/itemfile"
	"github.com/project-agonyl/agonyl-utils-go/questfile"
	textencoding "golang.org/x/text/encoding"
	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/encoding/korean"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/encoding/traditionalchinese"
	"golang.org/x/text/transform"

	"github.com/omnihance/omnihance-a3-agent/internal/constants"
	"github.com/omnihance/omnihance-a3-agent/internal/logger"
	"github.com/omnihance/omnihance-a3-agent/internal/mw"
	"github.com/omnihance/omnihance-a3-agent/internal/permissions"
	"github.com/omnihance/omnihance-a3-agent/internal/services"
	"github.com/omnihance/omnihance-a3-agent/internal/utils"
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
		r.Get("/drop-file", s.handleDropFileData)
		r.Put("/drop-file", s.handleUpdateDropFile)
		r.Get("/item-file", s.handleItemFileData)
		r.Put("/item-file", s.handleUpdateItemFile)
		r.Get("/item-combination-data", s.handleItemCombinationDataFileData)
		r.Put("/item-combination-data", s.handleUpdateItemCombinationDataFile)
		r.Get("/quest-file", s.handleQuestFileData)
		r.Put("/quest-file", s.handleUpdateQuestFile)
		r.Post("/revert-file", s.handleRevertFile)
		r.Post("/duplicate-file", s.handleDuplicateFile)
		r.Get("/revision-summary", s.handleRevisionSummary)
		r.Post("/download-link", s.handleCreateFileDownloadLink)
		r.Post("/directory-download-link", s.handleCreateDirectoryDownloadLink)
		r.Get("/directory-downloads/{run_id}", s.handleGetDirectoryDownloadStatus)
		r.Get("/download/{token}", s.handleDownloadLinkedFile)
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

func (s *Server) handleDropFileData(w http.ResponseWriter, r *http.Request) {
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
	if fileType != services.FileTypeDrop {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusBadRequest, map[string]interface{}{
			"errorCode": constants.ErrorCodeFileNotViewable,
			"context":   "file-system",
			"errors":    []string{"File is not a drop file"},
		})
		return
	}

	dropData, err := s.fileEditor.ReadDropFileData(cleanPath)
	if err != nil {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusInternalServerError, map[string]interface{}{
			"errorCode": constants.ErrorCodeFileReadError,
			"context":   "file-system",
			"errors":    []string{"Failed to read drop file data: " + err.Error()},
		})
		return
	}

	_ = utils.WriteJSONResponse(w, dropFileToAPIData(dropData))
}

func (s *Server) handleItemCombinationDataFileData(w http.ResponseWriter, r *http.Request) {
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
	if fileType != services.FileTypeItemCombinationData {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusBadRequest, map[string]interface{}{
			"errorCode": constants.ErrorCodeFileNotViewable,
			"context":   "file-system",
			"errors":    []string{"File is not an item combination data file"},
		})
		return
	}

	itemCombinationData, err := s.fileEditor.ReadItemCombinationData(cleanPath)
	if err != nil {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusInternalServerError, map[string]interface{}{
			"errorCode": constants.ErrorCodeFileReadError,
			"context":   "file-system",
			"errors":    []string{"Failed to read item combination data: " + err.Error()},
		})
		return
	}

	_ = utils.WriteJSONResponse(w, itemCombinationDataToAPIData(itemCombinationData))
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

func (s *Server) updateFileWithRevision(w http.ResponseWriter, ctx *fileUpdateContext, buildUpdate func() ([]byte, func() error, bool)) (int64, bool) {
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

	previousData, err := s.fileEditor.ReadFile(ctx.cleanPath)
	if err != nil {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusInternalServerError, map[string]interface{}{
			"errorCode": constants.ErrorCodeFileReadError,
			"context":   "file-system",
			"errors":    []string{"Failed to read file: " + err.Error()},
		})
		return 0, false
	}

	currentData, writeFile, ok := buildUpdate()
	if !ok {
		return 0, false
	}

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

	tx, err := s.internalDB.BeginTx()
	if err != nil {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusInternalServerError, map[string]interface{}{
			"errorCode": constants.ErrorCodeInternalServerError,
			"context":   "file-system",
			"errors":    []string{"Failed to begin transaction: " + err.Error()},
		})
		return 0, false
	}

	fileWriteAttempted := false
	defer func() {
		if err != nil {
			if fileWriteAttempted {
				s.restoreFileAfterFailedWrite(ctx.cleanPath, previousData, ctx.info.Mode())
			}

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

	fileWriteAttempted = true
	if err = writeFile(); err != nil {
		if removeErr := s.fileEditor.Remove(revisionPath); removeErr != nil {
			s.log.Error("Failed to remove revision file during cleanup", logger.Field{Key: "error", Value: removeErr})
		}

		if removeErr := s.fileEditor.RemoveAll(revisionDir); removeErr != nil {
			s.log.Error("Failed to remove revision directory during cleanup", logger.Field{Key: "error", Value: removeErr})
		}

		_ = utils.WriteJSONResponseWithStatus(w, http.StatusInternalServerError, map[string]interface{}{
			"errorCode": constants.ErrorCodeInternalServerError,
			"context":   "file-system",
			"errors":    []string{"Failed to write file: " + err.Error()},
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

func (s *Server) restoreFileAfterFailedWrite(path string, previousData []byte, previousMode os.FileMode) {
	if restoreErr := s.fileEditor.WriteFile(path, previousData, previousMode); restoreErr != nil {
		s.log.Error("Failed to restore file after failed revision update", logger.Field{Key: "error", Value: restoreErr})
	}
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

	revisionID, ok := s.updateFileWithRevision(w, ctx, func() ([]byte, func() error, bool) {
		return currentData, func() error {
			return s.fileEditor.WriteNPCFileData(ctx.cleanPath, npcData)
		}, true
	})
	if !ok {
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

	var req TextFileUpdateAPIData
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

	currentData := []byte(*req.Content)

	revisionID, ok := s.updateFileWithRevision(w, ctx, func() ([]byte, func() error, bool) {
		return currentData, func() error {
			return s.fileEditor.WriteTextFileData(ctx.cleanPath, *req.Content, ctx.info.Mode())
		}, true
	})
	if !ok {
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

	revisionID, ok := s.updateFileWithRevision(w, ctx, func() ([]byte, func() error, bool) {
		return currentData, func() error {
			return s.fileEditor.WriteSpawnFileData(ctx.cleanPath, spawnData)
		}, true
	})
	if !ok {
		return
	}

	_ = utils.WriteJSONResponse(w, map[string]interface{}{
		"message":     "File updated successfully",
		"revision_id": revisionID,
	})
}

func dropFileToAPIData(dropData dropfile.DropFile) DropFileAPIData {
	apiDrops := make([]DropAPIData, len(dropData))
	for i, drop := range dropData {
		itemCode := drop.ItemID
		dropRate := drop.DropRate
		dropGroup := drop.DropGroup
		apiDrops[i] = DropAPIData{
			ItemCode:  &itemCode,
			DropRate:  &dropRate,
			DropGroup: &dropGroup,
		}
	}

	return DropFileAPIData{Drops: apiDrops}
}

func dropFileFromAPIData(req DropFileAPIData) dropfile.DropFile {
	dropData := make(dropfile.DropFile, len(req.Drops))
	for i, drop := range req.Drops {
		dropData[i] = dropfile.Drop{
			ItemID:    *drop.ItemCode,
			DropRate:  *drop.DropRate,
			DropGroup: *drop.DropGroup,
		}
	}

	return dropData
}

func (s *Server) handleUpdateDropFile(w http.ResponseWriter, r *http.Request) {
	if !s.requireUserPermission(w, r, permissions.ActionEditFiles) {
		return
	}

	ctx, ok := s.validateFileUpdateRequest(w, r, services.FileTypeDrop, "drop")
	if !ok {
		return
	}

	var req DropFileAPIData
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

	dropData := dropFileFromAPIData(req)

	var currentDataBuffer bytes.Buffer
	if err := dropfile.Write(&currentDataBuffer, dropData); err != nil {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusInternalServerError, map[string]interface{}{
			"errorCode": constants.ErrorCodeInternalServerError,
			"context":   "file-system",
			"errors":    []string{"Failed to serialize drop data: " + err.Error()},
		})
		return
	}

	currentData := currentDataBuffer.Bytes()
	if _, err := dropfile.Read(bytes.NewReader(currentData)); err != nil {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusInternalServerError, map[string]interface{}{
			"errorCode": constants.ErrorCodeInternalServerError,
			"context":   "file-system",
			"errors":    []string{"Failed to validate drop data: " + err.Error()},
		})
		return
	}

	revisionID, ok := s.updateFileWithRevision(w, ctx, func() ([]byte, func() error, bool) {
		return currentData, func() error {
			return s.fileEditor.WriteDropFileData(ctx.cleanPath, dropData)
		}, true
	})
	if !ok {
		return
	}

	_ = utils.WriteJSONResponse(w, map[string]interface{}{
		"message":     "File updated successfully",
		"revision_id": revisionID,
	})
}

const (
	itemFileTypeIT0   = "it0"
	itemFileTypeIT0Ex = "it0ex"
	itemFileTypeIT1   = "it1"
	itemFileTypeIT2   = "it2"
	itemFileTypeIT3   = "it3"
	itemFileNameSize  = 32

	itemFileNameEncodingUTF8     = "utf-8"
	itemFileNameEncodingEUCKR    = "euc-kr"
	itemFileNameEncodingGBK      = "gbk"
	itemFileNameEncodingBig5     = "big5"
	itemFileNameEncodingShiftJIS = "shift-jis"
)

var itemFileNameLegacyEncodings = []*itemFileNameEncoding{
	{label: itemFileNameEncodingEUCKR, encoding: korean.EUCKR},
	{label: itemFileNameEncodingGBK, encoding: simplifiedchinese.GBK},
	{label: itemFileNameEncodingBig5, encoding: traditionalchinese.Big5},
	{label: itemFileNameEncodingShiftJIS, encoding: japanese.ShiftJIS},
}

func (s *Server) handleItemFileData(w http.ResponseWriter, r *http.Request) {
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
	if !isItemFileType(fileType) {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusBadRequest, map[string]interface{}{
			"errorCode": constants.ErrorCodeFileNotViewable,
			"context":   "file-system",
			"errors":    []string{"File is not an item file"},
		})
		return
	}

	if !s.fileEditor.IsFileViewable(cleanPath, info) {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusBadRequest, map[string]interface{}{
			"errorCode": constants.ErrorCodeFileNotViewable,
			"context":   "file-system",
			"errors":    []string{"IT0Ex item file requires sibling 0 file"},
		})
		return
	}

	nameEncoding := r.URL.Query().Get("name_encoding")
	if _, _, err := requestedItemFileNameEncoding(nameEncoding); err != nil {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusBadRequest, map[string]interface{}{
			"errorCode": constants.ErrorCodeBadRequest,
			"context":   "file-system",
			"errors":    []string{err.Error()},
		})
		return
	}

	apiData, err := s.readItemFileAPIData(cleanPath, fileType, nameEncoding)
	if err != nil {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusInternalServerError, map[string]interface{}{
			"errorCode": constants.ErrorCodeFileReadError,
			"context":   "file-system",
			"errors":    []string{"Failed to read item file data: " + err.Error()},
		})
		return
	}

	_ = utils.WriteJSONResponse(w, apiData)
}

func (s *Server) handleUpdateItemFile(w http.ResponseWriter, r *http.Request) {
	if !s.requireUserPermission(w, r, permissions.ActionEditFiles) {
		return
	}

	ctx, fileType, ok := s.validateItemFileUpdateRequest(w, r)
	if !ok {
		return
	}

	var req ItemFileAPIData
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusBadRequest, map[string]interface{}{
			"errorCode": constants.ErrorCodeBadRequest,
			"context":   "file-system",
			"errors":    []string{"Invalid request body: " + err.Error()},
		})
		return
	}

	queryNameEncoding := r.URL.Query().Get("name_encoding")
	if _, _, err := requestedItemFileNameEncoding(queryNameEncoding); err != nil {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusBadRequest, map[string]interface{}{
			"errorCode": constants.ErrorCodeBadRequest,
			"context":   "file-system",
			"errors":    []string{err.Error()},
		})
		return
	}

	if req.NameEncoding == "" {
		req.NameEncoding = queryNameEncoding
	}

	if expectedType := itemFileAPIType(fileType); req.ItemFileType != expectedType {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusBadRequest, map[string]interface{}{
			"errorCode": constants.ErrorCodeBadRequest,
			"context":   "file-system",
			"errors":    []string{"item_file_type must be " + expectedType},
		})
		return
	}

	if _, _, err := requestedItemFileNameEncoding(req.NameEncoding); err != nil {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusBadRequest, map[string]interface{}{
			"errorCode": constants.ErrorCodeBadRequest,
			"context":   "file-system",
			"errors":    []string{err.Error()},
		})
		return
	}

	revisionID, ok := s.updateFileWithRevision(w, ctx, func() ([]byte, func() error, bool) {
		currentData, writeFile, validationErrs, err := s.buildItemFileUpdate(ctx.cleanPath, fileType, req)
		if err != nil {
			_ = utils.WriteJSONResponseWithStatus(w, http.StatusInternalServerError, map[string]interface{}{
				"errorCode": constants.ErrorCodeFileReadError,
				"context":   "file-system",
				"errors":    []string{"Failed to read item file data: " + err.Error()},
			})
			return nil, nil, false
		}

		if len(validationErrs) > 0 {
			_ = utils.WriteJSONResponseWithStatus(w, http.StatusBadRequest, map[string]interface{}{
				"errorCode": constants.ErrorCodeBadRequest,
				"context":   "file-system",
				"errors":    validationErrs,
			})
			return nil, nil, false
		}

		return currentData, writeFile, true
	})
	if !ok {
		return
	}

	_ = utils.WriteJSONResponse(w, map[string]interface{}{
		"message":     "File updated successfully",
		"revision_id": revisionID,
	})
}

func (s *Server) validateItemFileUpdateRequest(w http.ResponseWriter, r *http.Request) (*fileUpdateContext, services.FileType, bool) {
	userID, ok := utils.GetUserIdFromContext(r.Context())
	if !ok {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusUnauthorized, map[string]interface{}{
			"errorCode": constants.ErrorCodeUnauthorized,
			"context":   "file-system",
			"errors":    []string{"User ID not found in context"},
		})
		return nil, "", false
	}

	pathParam := r.URL.Query().Get("path")
	if pathParam == "" {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusBadRequest, map[string]interface{}{
			"errorCode": constants.ErrorCodeBadRequest,
			"context":   "file-system",
			"errors":    []string{"Path parameter is required"},
		})
		return nil, "", false
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
			return nil, "", false
		}

		_ = utils.WriteJSONResponseWithStatus(w, http.StatusInternalServerError, map[string]interface{}{
			"errorCode": constants.ErrorCodeFileReadError,
			"context":   "file-system",
			"errors":    []string{"Cannot read file: " + err.Error()},
		})
		return nil, "", false
	}

	if info.IsDir() {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusBadRequest, map[string]interface{}{
			"errorCode": constants.ErrorCodePathIsDirectory,
			"context":   "file-system",
			"errors":    []string{"Path is a directory, not a file"},
		})
		return nil, "", false
	}

	fileType := s.fileEditor.GetFileType(cleanPath, info)
	if !isItemFileType(fileType) {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusBadRequest, map[string]interface{}{
			"errorCode": constants.ErrorCodeFileNotViewable,
			"context":   "file-system",
			"errors":    []string{"File is not an item file"},
		})
		return nil, "", false
	}

	if !s.fileEditor.IsFileEditable(cleanPath, info) {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusBadRequest, map[string]interface{}{
			"errorCode": constants.ErrorCodeBadRequest,
			"context":   "file-system",
			"errors":    []string{"File is not editable"},
		})
		return nil, "", false
	}

	return &fileUpdateContext{
		userID:    userID,
		cleanPath: cleanPath,
		info:      info,
		fileID:    utils.GenerateMD5Hash(cleanPath),
	}, fileType, true
}

func (s *Server) readItemFileAPIData(path string, fileType services.FileType, requestedNameEncoding string) (ItemFileAPIData, error) {
	switch fileType {
	case services.FileTypeIT0Item:
		data, err := s.fileEditor.ReadIT0ItemFileData(path)
		if err != nil {
			return ItemFileAPIData{}, err
		}

		return it0ItemFileToAPIData(data, requestedNameEncoding), nil
	case services.FileTypeIT0ExItem:
		data, err := s.fileEditor.ReadIT0ExItemFileData(path)
		if err != nil {
			return ItemFileAPIData{}, err
		}

		baseData, err := s.fileEditor.ReadIT0ItemFileData(siblingIT0ItemFilePath(path))
		if err != nil {
			return ItemFileAPIData{}, err
		}

		return it0ExItemFileToAPIData(data, baseData, requestedNameEncoding)
	case services.FileTypeIT1Item:
		data, err := s.fileEditor.ReadIT1ItemFileData(path)
		if err != nil {
			return ItemFileAPIData{}, err
		}

		return it1ItemFileToAPIData(data, requestedNameEncoding), nil
	case services.FileTypeIT2Item:
		data, err := s.fileEditor.ReadIT2ItemFileData(path)
		if err != nil {
			return ItemFileAPIData{}, err
		}

		return it2ItemFileToAPIData(data, requestedNameEncoding), nil
	case services.FileTypeIT3Item:
		data, err := s.fileEditor.ReadIT3ItemFileData(path)
		if err != nil {
			return ItemFileAPIData{}, err
		}

		return it3ItemFileToAPIData(data, requestedNameEncoding), nil
	default:
		return ItemFileAPIData{}, fmt.Errorf("unsupported item file type %s", fileType)
	}
}

func (s *Server) buildItemFileUpdate(path string, fileType services.FileType, req ItemFileAPIData) ([]byte, func() error, []string, error) {
	switch fileType {
	case services.FileTypeIT0Item:
		existing, err := s.fileEditor.ReadIT0ItemFileData(path)
		if err != nil {
			return nil, nil, nil, err
		}

		next, validationErrs := it0ItemFileFromAPIData(req, existing)
		if len(validationErrs) > 0 {
			return nil, nil, validationErrs, nil
		}

		currentData, err := serializeIT0ItemFile(next)
		if err != nil {
			return nil, nil, nil, err
		}

		return currentData, func() error {
			return s.fileEditor.WriteIT0ItemFileData(path, next)
		}, nil, nil
	case services.FileTypeIT0ExItem:
		existingBase, err := s.fileEditor.ReadIT0ItemFileData(siblingIT0ItemFilePath(path))
		if err != nil {
			return nil, nil, nil, err
		}

		next, validationErrs := it0ExItemFileFromAPIData(req, existingBase)
		if len(validationErrs) > 0 {
			return nil, nil, validationErrs, nil
		}

		currentData, err := serializeIT0ExItemFile(next)
		if err != nil {
			return nil, nil, nil, err
		}

		return currentData, func() error {
			return s.fileEditor.WriteIT0ExItemFileData(path, next)
		}, nil, nil
	case services.FileTypeIT1Item:
		existing, err := s.fileEditor.ReadIT1ItemFileData(path)
		if err != nil {
			return nil, nil, nil, err
		}

		next, validationErrs := it1ItemFileFromAPIData(req, existing)
		if len(validationErrs) > 0 {
			return nil, nil, validationErrs, nil
		}

		currentData, err := serializeIT1ItemFile(next)
		if err != nil {
			return nil, nil, nil, err
		}

		return currentData, func() error {
			return s.fileEditor.WriteIT1ItemFileData(path, next)
		}, nil, nil
	case services.FileTypeIT2Item:
		existing, err := s.fileEditor.ReadIT2ItemFileData(path)
		if err != nil {
			return nil, nil, nil, err
		}

		next, validationErrs := it2ItemFileFromAPIData(req, existing)
		if len(validationErrs) > 0 {
			return nil, nil, validationErrs, nil
		}

		currentData, err := serializeIT2ItemFile(next)
		if err != nil {
			return nil, nil, nil, err
		}

		return currentData, func() error {
			return s.fileEditor.WriteIT2ItemFileData(path, next)
		}, nil, nil
	case services.FileTypeIT3Item:
		existing, err := s.fileEditor.ReadIT3ItemFileData(path)
		if err != nil {
			return nil, nil, nil, err
		}

		next, validationErrs := it3ItemFileFromAPIData(req, existing)
		if len(validationErrs) > 0 {
			return nil, nil, validationErrs, nil
		}

		currentData, err := serializeIT3ItemFile(next)
		if err != nil {
			return nil, nil, nil, err
		}

		return currentData, func() error {
			return s.fileEditor.WriteIT3ItemFileData(path, next)
		}, nil, nil
	default:
		return nil, nil, nil, fmt.Errorf("unsupported item file type %s", fileType)
	}
}

func it0ItemFileToAPIData(data itemfile.IT0File, requestedNameEncoding string) ItemFileAPIData {
	nameEncoding, forceNameEncoding := itemFileNameEncodingSelection(requestedNameEncoding, it0ItemFileNameEncoding(data))
	items := make([]ItemFileItemAPIData, len(data))
	for i, raw := range data {
		items[i] = ItemFileItemAPIData{
			ItemCodeBase: uint16Ptr(raw.ItemCodeBase),
			Row:          uint16Ptr(raw.Row),
			Slot:         uint16Ptr(raw.Slot),
			Type:         uint16Ptr(raw.Type),
			ItemCode:     uint32Ptr((uint32(raw.ItemCodeBase) << 10) + uint32(raw.Row)),
			Name:         itemFileNameToAPIString(raw.Name, nameEncoding, forceNameEncoding),
			NPCPrice:     uint32Ptr(raw.NPCPrice),
			Levels:       itemFileLevelsToAPIData(raw.Levels[:], 1),
		}
	}

	return ItemFileAPIData{
		ItemFileType: itemFileTypeIT0,
		NameEncoding: itemFileNameEncodingLabel(nameEncoding),
		Items:        items,
	}
}

func it0ExItemFileToAPIData(data itemfile.IT0ExFile, baseData itemfile.IT0File, requestedNameEncoding string) (ItemFileAPIData, error) {
	nameEncoding, forceNameEncoding := itemFileNameEncodingSelection(requestedNameEncoding, it0ItemFileNameEncoding(baseData))
	baseByRow := make(map[uint16]itemfile.IT0Raw, len(baseData))
	for _, raw := range baseData {
		baseByRow[raw.Row] = raw
	}

	items := make([]ItemFileItemAPIData, len(data))
	for i, raw := range data {
		baseRaw, ok := baseByRow[raw.Row]
		if !ok {
			return ItemFileAPIData{}, fmt.Errorf("IT0Ex row %d references missing IT0 base item", raw.Row)
		}

		items[i] = ItemFileItemAPIData{
			ItemCodeBase: uint16Ptr(baseRaw.ItemCodeBase),
			Row:          uint16Ptr(raw.Row),
			Slot:         uint16Ptr(baseRaw.Slot),
			Type:         uint16Ptr(baseRaw.Type),
			ItemCode:     uint32Ptr((uint32(baseRaw.ItemCodeBase) << 10) + uint32(baseRaw.Row)),
			Name:         itemFileNameToAPIString(baseRaw.Name, nameEncoding, forceNameEncoding),
			Levels:       itemFileLevelsToAPIData(raw.Levels[:], 11),
		}
	}

	return ItemFileAPIData{
		ItemFileType: itemFileTypeIT0Ex,
		NameEncoding: itemFileNameEncodingLabel(nameEncoding),
		Items:        items,
		BaseItems:    it0BaseItemsToAPIData(baseData, nameEncoding, forceNameEncoding),
	}, nil
}

func it1ItemFileToAPIData(data itemfile.IT1File, requestedNameEncoding string) ItemFileAPIData {
	nameEncoding, forceNameEncoding := itemFileNameEncodingSelection(requestedNameEncoding, it1ItemFileNameEncoding(data))
	items := make([]ItemFileItemAPIData, len(data))
	for i, raw := range data {
		items[i] = ItemFileItemAPIData{
			Row:           uint16Ptr(raw.Row),
			Type:          uint16Ptr(raw.Type),
			ItemCode:      uint32Ptr((uint32(raw.Type) << 10) + uint32(raw.Row)),
			Name:          itemFileNameToAPIString(raw.Name, nameEncoding, forceNameEncoding),
			NPCPrice:      uint32Ptr(raw.NPCPrice),
			RequiredLevel: uint16Ptr(raw.RequiredLevel),
			Attribute:     uint16Ptr(raw.Attribute),
			BlueOption:    uint16Ptr(raw.BlueOption),
			RedOption:     uint16Ptr(raw.RedOption),
			GreyOption:    uint16Ptr(raw.GreyOption),
		}
	}

	return ItemFileAPIData{
		ItemFileType: itemFileTypeIT1,
		NameEncoding: itemFileNameEncodingLabel(nameEncoding),
		Items:        items,
	}
}

func it2ItemFileToAPIData(data itemfile.IT2File, requestedNameEncoding string) ItemFileAPIData {
	nameEncoding, forceNameEncoding := itemFileNameEncodingSelection(requestedNameEncoding, it2ItemFileNameEncoding(data))
	items := make([]ItemFileItemAPIData, len(data))
	for i, raw := range data {
		items[i] = ItemFileItemAPIData{
			Row:           uint16Ptr(raw.Row),
			Type:          uint16Ptr(raw.Type),
			ItemCode:      uint32Ptr((uint32(raw.Type) << 10) + uint32(raw.Row)),
			Name:          itemFileNameToAPIString(raw.Name, nameEncoding, forceNameEncoding),
			NPCPrice:      uint32Ptr(raw.NPCPrice),
			Class:         uint16Ptr(raw.Class),
			RequiredLevel: uint16Ptr(raw.RequiredLevel),
			SkillLevel:    uint16Ptr(raw.SkillLevel),
		}
	}

	return ItemFileAPIData{
		ItemFileType: itemFileTypeIT2,
		NameEncoding: itemFileNameEncodingLabel(nameEncoding),
		Items:        items,
	}
}

func it3ItemFileToAPIData(data itemfile.IT3File, requestedNameEncoding string) ItemFileAPIData {
	nameEncoding, forceNameEncoding := itemFileNameEncodingSelection(requestedNameEncoding, it3ItemFileNameEncoding(data))
	items := make([]ItemFileItemAPIData, len(data))
	for i, raw := range data {
		items[i] = ItemFileItemAPIData{
			Row:      uint16Ptr(raw.Row),
			Type:     uint16Ptr(raw.Type),
			ItemCode: uint32Ptr((uint32(raw.Type) << 10) + uint32(raw.Row)),
			Name:     itemFileNameToAPIString(raw.Name, nameEncoding, forceNameEncoding),
			NPCPrice: uint32Ptr(raw.NPCPrice),
		}
	}

	return ItemFileAPIData{
		ItemFileType: itemFileTypeIT3,
		NameEncoding: itemFileNameEncodingLabel(nameEncoding),
		Items:        items,
	}
}

func it0BaseItemsToAPIData(data itemfile.IT0File, nameEncoding *itemFileNameEncoding, forceNameEncoding bool) []ItemFileBaseItemAPIData {
	baseItems := make([]ItemFileBaseItemAPIData, len(data))
	for i, raw := range data {
		baseItems[i] = ItemFileBaseItemAPIData{
			Row:      uint16Ptr(raw.Row),
			ItemCode: uint32Ptr((uint32(raw.ItemCodeBase) << 10) + uint32(raw.Row)),
			Name:     itemFileNameToAPIString(raw.Name, nameEncoding, forceNameEncoding),
			Levels:   it0ExDefaultLevelsToAPIData(raw.Levels[9]),
		}
	}

	return baseItems
}

func it0ItemFileFromAPIData(req ItemFileAPIData, existing itemfile.IT0File) (itemfile.IT0File, []string) {
	errs := validateFixedItemRows(req.Items, len(existing), "IT0")
	if len(errs) > 0 {
		return nil, errs
	}

	data := append(itemfile.IT0File(nil), existing...)
	rowToIndex, rowErrs := it0RowIndexMap(existing, "IT0")
	errs = append(errs, rowErrs...)
	seenRows := map[uint16]bool{}
	nameEncoding, forceNameEncoding := itemFileNameEncodingSelection(req.NameEncoding, it0ItemFileNameEncoding(existing))
	for i, apiItem := range req.Items {
		row, ok := validateExistingItemRow(apiItem.Row, rowToIndex, seenRows)
		if !ok {
			errs = append(errs, fmt.Sprintf("items[%d].row must reference an existing IT0 row", i))
			continue
		}

		raw := data[rowToIndex[row]]
		if name, err := itemFileNameFromAPIData(raw.Name, apiItem.Name, nameEncoding, forceNameEncoding); err != nil {
			errs = append(errs, fmt.Sprintf("items[%d].name must fit selected name encoding and not exceed %d bytes", i, itemFileNameSize))
		} else {
			raw.Name = name
		}

		if apiItem.NPCPrice == nil {
			errs = append(errs, fmt.Sprintf("items[%d].npc_price is required", i))
		} else {
			raw.NPCPrice = *apiItem.NPCPrice
		}

		if apiItem.Slot == nil {
			errs = append(errs, fmt.Sprintf("items[%d].slot is required", i))
		} else {
			raw.Slot = *apiItem.Slot
		}

		if len(apiItem.Levels) != len(raw.Levels) {
			errs = append(errs, fmt.Sprintf("items[%d].levels must contain %d levels", i, len(raw.Levels)))
		} else {
			for j := range raw.Levels {
				raw.Levels[j] = itemFileLevelFromAPIData(apiItem.Levels[j], fmt.Sprintf("items[%d].levels[%d]", i, j), &errs)
			}
		}

		data[rowToIndex[row]] = raw
	}

	if len(errs) > 0 {
		return nil, errs
	}

	return data, nil
}

func it0ExItemFileFromAPIData(req ItemFileAPIData, baseData itemfile.IT0File) (itemfile.IT0ExFile, []string) {
	baseRows := make(map[uint16]bool, len(baseData))
	for _, raw := range baseData {
		baseRows[raw.Row] = true
	}

	errs := []string{}
	data := make(itemfile.IT0ExFile, len(req.Items))
	seenRows := map[uint16]bool{}
	for i, apiItem := range req.Items {
		if apiItem.Row == nil {
			errs = append(errs, fmt.Sprintf("items[%d].row is required", i))
			continue
		}

		row := *apiItem.Row
		if seenRows[row] {
			errs = append(errs, fmt.Sprintf("items[%d].row duplicates IT0Ex row %d", i, row))
			continue
		}

		seenRows[row] = true
		if !baseRows[row] {
			errs = append(errs, fmt.Sprintf("items[%d].row must reference an existing IT0 row", i))
			continue
		}

		raw := itemfile.IT0ExRaw{Row: row}
		if len(apiItem.Levels) != len(raw.Levels) {
			errs = append(errs, fmt.Sprintf("items[%d].levels must contain %d levels", i, len(raw.Levels)))
		} else {
			for j := range raw.Levels {
				raw.Levels[j] = itemFileLevelFromAPIData(apiItem.Levels[j], fmt.Sprintf("items[%d].levels[%d]", i, j), &errs)
			}
		}

		data[i] = raw
	}

	if len(errs) > 0 {
		return nil, errs
	}

	return data, nil
}

func it1ItemFileFromAPIData(req ItemFileAPIData, existing itemfile.IT1File) (itemfile.IT1File, []string) {
	errs := validateFixedItemRows(req.Items, len(existing), "IT1")
	if len(errs) > 0 {
		return nil, errs
	}

	data := append(itemfile.IT1File(nil), existing...)
	rowToIndex, rowErrs := it1RowIndexMap(existing, "IT1")
	errs = append(errs, rowErrs...)
	seenRows := map[uint16]bool{}
	nameEncoding, forceNameEncoding := itemFileNameEncodingSelection(req.NameEncoding, it1ItemFileNameEncoding(existing))
	for i, apiItem := range req.Items {
		row, ok := validateExistingItemRow(apiItem.Row, rowToIndex, seenRows)
		if !ok {
			errs = append(errs, fmt.Sprintf("items[%d].row must reference an existing IT1 row", i))
			continue
		}

		raw := data[rowToIndex[row]]
		if name, err := itemFileNameFromAPIData(raw.Name, apiItem.Name, nameEncoding, forceNameEncoding); err != nil {
			errs = append(errs, fmt.Sprintf("items[%d].name must fit selected name encoding and not exceed %d bytes", i, itemFileNameSize))
		} else {
			raw.Name = name
		}

		if apiItem.NPCPrice == nil {
			errs = append(errs, fmt.Sprintf("items[%d].npc_price is required", i))
		} else {
			raw.NPCPrice = *apiItem.NPCPrice
		}

		if apiItem.RequiredLevel == nil {
			errs = append(errs, fmt.Sprintf("items[%d].required_level is required", i))
		} else {
			raw.RequiredLevel = *apiItem.RequiredLevel
		}

		if apiItem.Attribute == nil {
			errs = append(errs, fmt.Sprintf("items[%d].attribute is required", i))
		} else {
			raw.Attribute = *apiItem.Attribute
		}

		if apiItem.BlueOption == nil {
			errs = append(errs, fmt.Sprintf("items[%d].blue_option is required", i))
		} else {
			raw.BlueOption = *apiItem.BlueOption
		}

		if apiItem.RedOption == nil {
			errs = append(errs, fmt.Sprintf("items[%d].red_option is required", i))
		} else {
			raw.RedOption = *apiItem.RedOption
		}

		if apiItem.GreyOption == nil {
			errs = append(errs, fmt.Sprintf("items[%d].grey_option is required", i))
		} else {
			raw.GreyOption = *apiItem.GreyOption
		}

		data[rowToIndex[row]] = raw
	}

	if len(errs) > 0 {
		return nil, errs
	}

	return data, nil
}

func it2ItemFileFromAPIData(req ItemFileAPIData, existing itemfile.IT2File) (itemfile.IT2File, []string) {
	errs := validateFixedItemRows(req.Items, len(existing), "IT2")
	if len(errs) > 0 {
		return nil, errs
	}

	data := append(itemfile.IT2File(nil), existing...)
	rowToIndex, rowErrs := it2RowIndexMap(existing, "IT2")
	errs = append(errs, rowErrs...)
	seenRows := map[uint16]bool{}
	nameEncoding, forceNameEncoding := itemFileNameEncodingSelection(req.NameEncoding, it2ItemFileNameEncoding(existing))
	for i, apiItem := range req.Items {
		row, ok := validateExistingItemRow(apiItem.Row, rowToIndex, seenRows)
		if !ok {
			errs = append(errs, fmt.Sprintf("items[%d].row must reference an existing IT2 row", i))
			continue
		}

		raw := data[rowToIndex[row]]
		if name, err := itemFileNameFromAPIData(raw.Name, apiItem.Name, nameEncoding, forceNameEncoding); err != nil {
			errs = append(errs, fmt.Sprintf("items[%d].name must fit selected name encoding and not exceed %d bytes", i, itemFileNameSize))
		} else {
			raw.Name = name
		}

		if apiItem.NPCPrice == nil {
			errs = append(errs, fmt.Sprintf("items[%d].npc_price is required", i))
		} else {
			raw.NPCPrice = *apiItem.NPCPrice
		}

		if apiItem.Class == nil {
			errs = append(errs, fmt.Sprintf("items[%d].class is required", i))
		} else {
			raw.Class = *apiItem.Class
		}

		if apiItem.RequiredLevel == nil {
			errs = append(errs, fmt.Sprintf("items[%d].required_level is required", i))
		} else {
			raw.RequiredLevel = *apiItem.RequiredLevel
		}

		if apiItem.SkillLevel == nil {
			errs = append(errs, fmt.Sprintf("items[%d].skill_level is required", i))
		} else {
			raw.SkillLevel = *apiItem.SkillLevel
		}

		data[rowToIndex[row]] = raw
	}

	if len(errs) > 0 {
		return nil, errs
	}

	return data, nil
}

func it3ItemFileFromAPIData(req ItemFileAPIData, existing itemfile.IT3File) (itemfile.IT3File, []string) {
	errs := validateFixedItemRows(req.Items, len(existing), "IT3")
	if len(errs) > 0 {
		return nil, errs
	}

	data := append(itemfile.IT3File(nil), existing...)
	rowToIndex, rowErrs := it3RowIndexMap(existing, "IT3")
	errs = append(errs, rowErrs...)
	seenRows := map[uint16]bool{}
	nameEncoding, forceNameEncoding := itemFileNameEncodingSelection(req.NameEncoding, it3ItemFileNameEncoding(existing))
	for i, apiItem := range req.Items {
		row, ok := validateExistingItemRow(apiItem.Row, rowToIndex, seenRows)
		if !ok {
			errs = append(errs, fmt.Sprintf("items[%d].row must reference an existing IT3 row", i))
			continue
		}

		raw := data[rowToIndex[row]]
		if name, err := itemFileNameFromAPIData(raw.Name, apiItem.Name, nameEncoding, forceNameEncoding); err != nil {
			errs = append(errs, fmt.Sprintf("items[%d].name must fit selected name encoding and not exceed %d bytes", i, itemFileNameSize))
		} else {
			raw.Name = name
		}

		if apiItem.NPCPrice == nil {
			errs = append(errs, fmt.Sprintf("items[%d].npc_price is required", i))
		} else {
			raw.NPCPrice = *apiItem.NPCPrice
		}

		data[rowToIndex[row]] = raw
	}

	if len(errs) > 0 {
		return nil, errs
	}

	return data, nil
}

func validateFixedItemRows(items []ItemFileItemAPIData, existingCount int, label string) []string {
	if len(items) != existingCount {
		return []string{fmt.Sprintf("%s item file must contain exactly %d items", label, existingCount)}
	}

	return nil
}

func validateExistingItemRow(rowValue *uint16, rowToIndex map[uint16]int, seenRows map[uint16]bool) (uint16, bool) {
	if rowValue == nil {
		return 0, false
	}

	row := *rowValue
	if _, ok := rowToIndex[row]; !ok {
		return row, false
	}

	if seenRows[row] {
		return row, false
	}

	seenRows[row] = true
	return row, true
}

func itemFileLevelsToAPIData(levels []itemfile.IT0RawLevelProperties, firstLevel uint8) []ItemFileLevelAPIData {
	apiLevels := make([]ItemFileLevelAPIData, len(levels))
	for i, level := range levels {
		apiLevels[i] = ItemFileLevelAPIData{
			Level:               uint8Ptr(firstLevel + uint8(i)),
			AdditionalAttribute: uint16Ptr(level.AdditionalAttribute),
			Strength:            uint16Ptr(level.Strength),
			Dexterity:           uint16Ptr(level.Dexterity),
			Intelligence:        uint16Ptr(level.Intelligence),
			Attribute:           uint16Ptr(level.Attribute),
			AttributeRange:      uint16Ptr(level.Range),
			BlueOption:          uint16Ptr(level.BlueOption),
			RedOption:           uint16Ptr(level.RedOption),
			GreyOption:          uint16Ptr(level.GreyOption),
		}
	}

	return apiLevels
}

func it0ExDefaultLevelsToAPIData(level itemfile.IT0RawLevelProperties) []ItemFileLevelAPIData {
	levels := make([]itemfile.IT0RawLevelProperties, 5)
	for i := range levels {
		levels[i] = level
	}

	return itemFileLevelsToAPIData(levels, 11)
}

func itemFileLevelFromAPIData(apiLevel ItemFileLevelAPIData, fieldPath string, errs *[]string) itemfile.IT0RawLevelProperties {
	level := itemfile.IT0RawLevelProperties{}
	if apiLevel.AdditionalAttribute == nil {
		*errs = append(*errs, fieldPath+".additional_attribute is required")
	} else {
		level.AdditionalAttribute = *apiLevel.AdditionalAttribute
	}

	if apiLevel.Strength == nil {
		*errs = append(*errs, fieldPath+".strength is required")
	} else {
		level.Strength = *apiLevel.Strength
	}

	if apiLevel.Dexterity == nil {
		*errs = append(*errs, fieldPath+".dexterity is required")
	} else {
		level.Dexterity = *apiLevel.Dexterity
	}

	if apiLevel.Intelligence == nil {
		*errs = append(*errs, fieldPath+".intelligence is required")
	} else {
		level.Intelligence = *apiLevel.Intelligence
	}

	if apiLevel.Attribute == nil {
		*errs = append(*errs, fieldPath+".attribute is required")
	} else {
		level.Attribute = *apiLevel.Attribute
	}

	if apiLevel.AttributeRange == nil {
		*errs = append(*errs, fieldPath+".attribute_range is required")
	} else {
		level.Range = *apiLevel.AttributeRange
	}

	if apiLevel.BlueOption == nil {
		*errs = append(*errs, fieldPath+".blue_option is required")
	} else {
		level.BlueOption = *apiLevel.BlueOption
	}

	if apiLevel.RedOption == nil {
		*errs = append(*errs, fieldPath+".red_option is required")
	} else {
		level.RedOption = *apiLevel.RedOption
	}

	if apiLevel.GreyOption == nil {
		*errs = append(*errs, fieldPath+".grey_option is required")
	} else {
		level.GreyOption = *apiLevel.GreyOption
	}

	return level
}

func itemFileNameToAPIString(name [itemFileNameSize]byte, nameEncoding *itemFileNameEncoding, forceNameEncoding bool) string {
	if forceNameEncoding {
		decoded, err := itemFileNameDecodeWithEncoding(name, nameEncoding)
		if err == nil {
			return decoded
		}
	}

	decoded, _ := itemFileNameDecode(name)
	return decoded
}

func itemFileNameFromAPIData(existingName [itemFileNameSize]byte, nextName string, nameEncoding *itemFileNameEncoding, forceNameEncoding bool) ([itemFileNameSize]byte, error) {
	existingDecoded, existingEncoding := itemFileNameDecode(existingName)
	if forceNameEncoding {
		decoded, err := itemFileNameDecodeWithEncoding(existingName, nameEncoding)
		if err == nil {
			existingDecoded = decoded
			existingEncoding = nameEncoding
		}
	}

	if nextName == existingDecoded {
		return existingName, nil
	}

	if !forceNameEncoding {
		nameEncoding = existingEncoding
	}

	encodedName, err := itemFileNameEncode(nextName, nameEncoding)
	if err != nil {
		return [itemFileNameSize]byte{}, err
	}

	if len(encodedName) > itemFileNameSize {
		return [itemFileNameSize]byte{}, fmt.Errorf("item name too long")
	}

	var name [itemFileNameSize]byte
	copy(name[:], encodedName)
	return name, nil
}

func itemFileNameDecodeWithEncoding(name [itemFileNameSize]byte, nameEncoding *itemFileNameEncoding) (string, error) {
	rawName := itemFileNameBytes(name[:])
	if len(rawName) == 0 {
		return "", nil
	}

	if nameEncoding == nil {
		if !utf8.Valid(rawName) {
			return externalUtils.ReadStringFromBytes(name[:]), nil
		}

		return string(rawName), nil
	}

	decoded, _, err := transform.String(nameEncoding.encoding.NewDecoder(), string(rawName))
	if err != nil {
		return "", err
	}

	return decoded, nil
}

func itemFileNameDecode(name [itemFileNameSize]byte) (string, *itemFileNameEncoding) {
	rawName := itemFileNameBytes(name[:])
	if len(rawName) == 0 {
		return "", nil
	}

	if utf8.Valid(rawName) {
		return string(rawName), nil
	}

	var (
		bestDecoded  string
		bestEncoding *itemFileNameEncoding
		bestScore    = -1
	)

	for _, candidate := range itemFileNameLegacyEncodings {
		decoded, _, err := transform.String(candidate.encoding.NewDecoder(), string(rawName))
		if err != nil {
			continue
		}

		score := itemFileNameDecodeScore(decoded)
		if score > bestScore {
			bestDecoded = decoded
			bestEncoding = candidate
			bestScore = score
		}
	}

	if bestScore >= 0 {
		return bestDecoded, bestEncoding
	}

	return externalUtils.ReadStringFromBytes(name[:]), nil
}

func itemFileNameEncodingSelection(requestedEncoding string, detectedEncoding *itemFileNameEncoding) (*itemFileNameEncoding, bool) {
	nameEncoding, hasRequestedEncoding, _ := requestedItemFileNameEncoding(requestedEncoding)
	if hasRequestedEncoding {
		return nameEncoding, true
	}

	if detectedEncoding != nil {
		return detectedEncoding, true
	}

	return nil, false
}

func requestedItemFileNameEncoding(value string) (*itemFileNameEncoding, bool, error) {
	normalizedValue := strings.ToLower(strings.TrimSpace(value))
	if normalizedValue == "" {
		return nil, false, nil
	}

	if normalizedValue == itemFileNameEncodingUTF8 {
		return nil, true, nil
	}

	for _, candidate := range itemFileNameLegacyEncodings {
		if normalizedValue == candidate.label {
			return candidate, true, nil
		}
	}

	return nil, false, fmt.Errorf("name_encoding must be one of %s", strings.Join(itemFileNameEncodingLabels(), ", "))
}

func itemFileNameEncodingLabel(nameEncoding *itemFileNameEncoding) string {
	if nameEncoding == nil {
		return itemFileNameEncodingUTF8
	}

	return nameEncoding.label
}

func itemFileNameEncodingLabels() []string {
	labels := []string{itemFileNameEncodingUTF8}
	for _, candidate := range itemFileNameLegacyEncodings {
		labels = append(labels, candidate.label)
	}

	return labels
}

func itemFileNameEncode(name string, nameEncoding *itemFileNameEncoding) ([]byte, error) {
	if nameEncoding == nil {
		return []byte(name), nil
	}

	encodedName, _, err := transform.String(nameEncoding.encoding.NewEncoder(), name)
	if err != nil {
		return nil, err
	}

	return []byte(encodedName), nil
}

func itemFileNameBytes(rawName []byte) []byte {
	if i := bytes.IndexByte(rawName, 0); i >= 0 {
		return rawName[:i]
	}

	return rawName
}

func itemFileNameDecodeScore(decoded string) int {
	if decoded == "" {
		return 0
	}

	score := 0
	for _, r := range decoded {
		switch {
		case r == utf8.RuneError || unicode.IsControl(r):
			return -1
		case r >= 0xff61 && r <= 0xff9f:
			score -= 4
		case unicode.In(r, unicode.Hangul, unicode.Han, unicode.Hiragana, unicode.Katakana):
			score += 4
		case r >= 0x20 && r <= 0x7e:
			score += 3
		case unicode.IsLetter(r) || unicode.IsNumber(r):
			score += 2
		case unicode.IsSpace(r):
			score++
		case unicode.IsPunct(r) || unicode.IsSymbol(r):
			continue
		default:
			score--
		}
	}

	return score
}

func it0ItemFileNameEncoding(data itemfile.IT0File) *itemFileNameEncoding {
	counts := map[*itemFileNameEncoding]int{}
	for _, raw := range data {
		countItemFileNameEncoding(counts, raw.Name)
	}

	return mostCommonItemFileNameEncoding(counts)
}

func it1ItemFileNameEncoding(data itemfile.IT1File) *itemFileNameEncoding {
	counts := map[*itemFileNameEncoding]int{}
	for _, raw := range data {
		countItemFileNameEncoding(counts, raw.Name)
	}

	return mostCommonItemFileNameEncoding(counts)
}

func it2ItemFileNameEncoding(data itemfile.IT2File) *itemFileNameEncoding {
	counts := map[*itemFileNameEncoding]int{}
	for _, raw := range data {
		countItemFileNameEncoding(counts, raw.Name)
	}

	return mostCommonItemFileNameEncoding(counts)
}

func it3ItemFileNameEncoding(data itemfile.IT3File) *itemFileNameEncoding {
	counts := map[*itemFileNameEncoding]int{}
	for _, raw := range data {
		countItemFileNameEncoding(counts, raw.Name)
	}

	return mostCommonItemFileNameEncoding(counts)
}

func countItemFileNameEncoding(counts map[*itemFileNameEncoding]int, name [itemFileNameSize]byte) {
	_, nameEncoding := itemFileNameDecode(name)
	if nameEncoding != nil {
		counts[nameEncoding]++
	}
}

func mostCommonItemFileNameEncoding(counts map[*itemFileNameEncoding]int) *itemFileNameEncoding {
	var (
		bestEncoding *itemFileNameEncoding
		bestCount    int
	)

	for _, candidate := range itemFileNameLegacyEncodings {
		if counts[candidate] > bestCount {
			bestEncoding = candidate
			bestCount = counts[candidate]
		}
	}

	return bestEncoding
}

func it0RowIndexMap(data itemfile.IT0File, label string) (map[uint16]int, []string) {
	rowToIndex := make(map[uint16]int, len(data))
	errs := []string{}
	for i, raw := range data {
		if _, exists := rowToIndex[raw.Row]; exists {
			errs = append(errs, fmt.Sprintf("%s row %d is duplicated in the existing file", label, raw.Row))
			continue
		}

		rowToIndex[raw.Row] = i
	}

	return rowToIndex, errs
}

func it1RowIndexMap(data itemfile.IT1File, label string) (map[uint16]int, []string) {
	rowToIndex := make(map[uint16]int, len(data))
	errs := []string{}
	for i, raw := range data {
		if _, exists := rowToIndex[raw.Row]; exists {
			errs = append(errs, fmt.Sprintf("%s row %d is duplicated in the existing file", label, raw.Row))
			continue
		}

		rowToIndex[raw.Row] = i
	}

	return rowToIndex, errs
}

func it2RowIndexMap(data itemfile.IT2File, label string) (map[uint16]int, []string) {
	rowToIndex := make(map[uint16]int, len(data))
	errs := []string{}
	for i, raw := range data {
		if _, exists := rowToIndex[raw.Row]; exists {
			errs = append(errs, fmt.Sprintf("%s row %d is duplicated in the existing file", label, raw.Row))
			continue
		}

		rowToIndex[raw.Row] = i
	}

	return rowToIndex, errs
}

func it3RowIndexMap(data itemfile.IT3File, label string) (map[uint16]int, []string) {
	rowToIndex := make(map[uint16]int, len(data))
	errs := []string{}
	for i, raw := range data {
		if _, exists := rowToIndex[raw.Row]; exists {
			errs = append(errs, fmt.Sprintf("%s row %d is duplicated in the existing file", label, raw.Row))
			continue
		}

		rowToIndex[raw.Row] = i
	}

	return rowToIndex, errs
}

func serializeIT0ItemFile(data itemfile.IT0File) ([]byte, error) {
	var buf bytes.Buffer
	if err := itemfile.WriteIT0(&buf, data); err != nil {
		return nil, err
	}

	if _, err := itemfile.ReadIT0(bytes.NewReader(buf.Bytes())); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func serializeIT0ExItemFile(data itemfile.IT0ExFile) ([]byte, error) {
	var buf bytes.Buffer
	if err := itemfile.WriteIT0Ex(&buf, data); err != nil {
		return nil, err
	}

	if _, err := itemfile.ReadIT0Ex(bytes.NewReader(buf.Bytes())); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func serializeIT1ItemFile(data itemfile.IT1File) ([]byte, error) {
	var buf bytes.Buffer
	if err := itemfile.WriteIT1(&buf, data); err != nil {
		return nil, err
	}

	if _, err := itemfile.ReadIT1(bytes.NewReader(buf.Bytes())); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func serializeIT2ItemFile(data itemfile.IT2File) ([]byte, error) {
	var buf bytes.Buffer
	if err := itemfile.WriteIT2(&buf, data); err != nil {
		return nil, err
	}

	if _, err := itemfile.ReadIT2(bytes.NewReader(buf.Bytes())); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func serializeIT3ItemFile(data itemfile.IT3File) ([]byte, error) {
	var buf bytes.Buffer
	if err := itemfile.WriteIT3(&buf, data); err != nil {
		return nil, err
	}

	if _, err := itemfile.ReadIT3(bytes.NewReader(buf.Bytes())); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func itemFileAPIType(fileType services.FileType) string {
	switch fileType {
	case services.FileTypeIT0Item:
		return itemFileTypeIT0
	case services.FileTypeIT0ExItem:
		return itemFileTypeIT0Ex
	case services.FileTypeIT1Item:
		return itemFileTypeIT1
	case services.FileTypeIT2Item:
		return itemFileTypeIT2
	case services.FileTypeIT3Item:
		return itemFileTypeIT3
	default:
		return ""
	}
}

func isItemFileType(fileType services.FileType) bool {
	switch fileType {
	case services.FileTypeIT0Item, services.FileTypeIT0ExItem, services.FileTypeIT1Item, services.FileTypeIT2Item, services.FileTypeIT3Item:
		return true
	default:
		return false
	}
}

func siblingIT0ItemFilePath(path string) string {
	return filepath.Join(filepath.Dir(path), services.IT0ItemFileName)
}

const (
	itemCombinationIngredientCount = 10
	maxItemCombinationSuccessRate  = 120
)

func itemCombinationDataToAPIData(data itemcombinationdata.ItemCombinationData) ItemCombinationDataFileAPIData {
	apiFormulas := make([]ItemCombinationFormulaAPIData, len(data))
	for i, formula := range data {
		successRate := formula.SuccessRate
		outcome := formula.Outcome
		apiFormulas[i] = ItemCombinationFormulaAPIData{
			Ingredients: []*uint16{
				uint16Ptr(formula.Item1),
				uint16Ptr(formula.Item2),
				uint16Ptr(formula.Item3),
				uint16Ptr(formula.Item4),
				uint16Ptr(formula.Item5),
				uint16Ptr(formula.Item6),
				uint16Ptr(formula.Item7),
				uint16Ptr(formula.Item8),
				uint16Ptr(formula.Item9),
				uint16Ptr(formula.Item10),
			},
			SuccessRate: &successRate,
			Outcome:     &outcome,
		}
	}

	return ItemCombinationDataFileAPIData{Formulas: apiFormulas}
}

func itemCombinationDataFromAPIData(req ItemCombinationDataFileAPIData, existingData itemcombinationdata.ItemCombinationData) itemcombinationdata.ItemCombinationData {
	data := make(itemcombinationdata.ItemCombinationData, len(req.Formulas))
	for i, formula := range req.Formulas {
		data[i] = itemCombinationFormulaFromAPIData(formula)
		if i < len(existingData) {
			data[i].Unknown1 = existingData[i].Unknown1
			data[i].Unknown2 = existingData[i].Unknown2
			data[i].Unknown3 = existingData[i].Unknown3
			data[i].Unknown4 = existingData[i].Unknown4
		}
	}

	return data
}

func itemCombinationFormulaFromAPIData(formula ItemCombinationFormulaAPIData) itemcombinationdata.CraftFormula {
	itemAt := func(index int) uint16 {
		if index >= len(formula.Ingredients) || formula.Ingredients[index] == nil {
			return 0
		}

		return *formula.Ingredients[index]
	}

	var successRate uint16
	if formula.SuccessRate != nil {
		successRate = *formula.SuccessRate
	}

	var outcome uint16
	if formula.Outcome != nil {
		outcome = *formula.Outcome
	}

	return itemcombinationdata.CraftFormula{
		Item1:       itemAt(0),
		Item2:       itemAt(1),
		Item3:       itemAt(2),
		Item4:       itemAt(3),
		Item5:       itemAt(4),
		Item6:       itemAt(5),
		Item7:       itemAt(6),
		Item8:       itemAt(7),
		Item9:       itemAt(8),
		Item10:      itemAt(9),
		SuccessRate: successRate,
		Outcome:     outcome,
	}
}

func (s *Server) handleUpdateItemCombinationDataFile(w http.ResponseWriter, r *http.Request) {
	if !s.requireUserPermission(w, r, permissions.ActionEditFiles) {
		return
	}

	ctx, ok := s.validateFileUpdateRequest(w, r, services.FileTypeItemCombinationData, "item combination data")
	if !ok {
		return
	}

	var req ItemCombinationDataFileAPIData
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

	if validationErrs := validateItemCombinationDataRequest(req); len(validationErrs) > 0 {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusBadRequest, map[string]interface{}{
			"errorCode": constants.ErrorCodeBadRequest,
			"context":   "file-system",
			"errors":    validationErrs,
		})
		return
	}

	revisionID, ok := s.updateFileWithRevision(w, ctx, func() ([]byte, func() error, bool) {
		existingData, err := s.fileEditor.ReadItemCombinationData(ctx.cleanPath)
		if err != nil {
			_ = utils.WriteJSONResponseWithStatus(w, http.StatusInternalServerError, map[string]interface{}{
				"errorCode": constants.ErrorCodeFileReadError,
				"context":   "file-system",
				"errors":    []string{"Failed to read item combination data: " + err.Error()},
			})
			return nil, nil, false
		}

		itemCombinationData := itemCombinationDataFromAPIData(req, existingData)

		var currentDataBuffer bytes.Buffer
		if err := itemcombinationdata.Write(&currentDataBuffer, itemCombinationData); err != nil {
			_ = utils.WriteJSONResponseWithStatus(w, http.StatusInternalServerError, map[string]interface{}{
				"errorCode": constants.ErrorCodeInternalServerError,
				"context":   "file-system",
				"errors":    []string{"Failed to serialize item combination data: " + err.Error()},
			})
			return nil, nil, false
		}

		currentData := currentDataBuffer.Bytes()
		if _, err := itemcombinationdata.Read(bytes.NewReader(currentData)); err != nil {
			_ = utils.WriteJSONResponseWithStatus(w, http.StatusInternalServerError, map[string]interface{}{
				"errorCode": constants.ErrorCodeInternalServerError,
				"context":   "file-system",
				"errors":    []string{"Failed to validate item combination data: " + err.Error()},
			})
			return nil, nil, false
		}

		return currentData, func() error {
			return s.fileEditor.WriteItemCombinationData(ctx.cleanPath, itemCombinationData)
		}, true
	})
	if !ok {
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
	if objAPI.Type == nil ||
		objAPI.Location == nil ||
		objAPI.Location.X == nil ||
		objAPI.Location.Y == nil {
		objective.Block = unusedQuestObjectiveBlock()
		objective.Name = nil
		return
	}

	objType := *objAPI.Type
	if objType == questfile.TypeUnused || objAPI.IsUnused {
		objective.Block = unusedQuestObjectiveBlock()
		objective.Name = nil
		return
	}

	blk := &objective.Block
	template := unusedQuestObjectiveBlock()
	copy(blk[:], template[:])
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

	revisionID, ok := s.updateFileWithRevision(w, ctx, func() ([]byte, func() error, bool) {
		qf, err := s.fileEditor.ReadQuestFileData(ctx.cleanPath)
		if err != nil {
			_ = utils.WriteJSONResponseWithStatus(w, http.StatusInternalServerError, map[string]interface{}{
				"errorCode": constants.ErrorCodeFileReadError,
				"context":   "file-system",
				"errors":    []string{"Failed to read quest file data: " + err.Error()},
			})
			return nil, nil, false
		}

		applyQuestFileAPIData(&qf, req)

		var currentDataBuffer bytes.Buffer
		if err := questfile.Write(&currentDataBuffer, qf); err != nil {
			_ = utils.WriteJSONResponseWithStatus(w, http.StatusInternalServerError, map[string]interface{}{
				"errorCode": constants.ErrorCodeInternalServerError,
				"context":   "file-system",
				"errors":    []string{"Failed to serialize quest data: " + err.Error()},
			})
			return nil, nil, false
		}

		currentData := currentDataBuffer.Bytes()
		if _, err := questfile.Read(bytes.NewReader(currentData)); err != nil {
			_ = utils.WriteJSONResponseWithStatus(w, http.StatusInternalServerError, map[string]interface{}{
				"errorCode": constants.ErrorCodeInternalServerError,
				"context":   "file-system",
				"errors":    []string{"Failed to validate quest data: " + err.Error()},
			})
			return nil, nil, false
		}

		return currentData, func() error {
			return s.fileEditor.WriteQuestFileData(ctx.cleanPath, qf)
		}, true
	})
	if !ok {
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

	currentData, err := s.fileEditor.ReadFile(cleanPath)
	if err != nil {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusInternalServerError, map[string]interface{}{
			"errorCode": constants.ErrorCodeFileReadError,
			"context":   "file-system",
			"errors":    []string{"Failed to read current file: " + err.Error()},
		})
		return
	}

	currentInfo, err := s.fileEditor.Stat(cleanPath)
	if err != nil {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusInternalServerError, map[string]interface{}{
			"errorCode": constants.ErrorCodeFileReadError,
			"context":   "file-system",
			"errors":    []string{"Failed to read current file metadata: " + err.Error()},
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

	fileWriteAttempted := false
	defer func() {
		if err != nil {
			if fileWriteAttempted {
				s.restoreFileAfterFailedWrite(cleanPath, currentData, currentInfo.Mode())
			}

			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				s.log.Error("Failed to rollback transaction", logger.Field{Key: "error", Value: rollbackErr})
			}
		}
	}()

	fileWriteAttempted = true
	if err = s.fileEditor.WriteFile(cleanPath, revisionData, currentInfo.Mode()); err != nil {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusInternalServerError, map[string]interface{}{
			"errorCode": constants.ErrorCodeInternalServerError,
			"context":   "file-system",
			"errors":    []string{"Failed to write file: " + err.Error()},
		})
		return
	}

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

func (s *Server) handleDuplicateFile(w http.ResponseWriter, r *http.Request) {
	if !s.requireUserPermission(w, r, permissions.ActionEditFiles) {
		return
	}

	var req DuplicateFileRequest
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
		var validationErrors []string
		for _, fieldErr := range err.(validator.ValidationErrors) {
			validationErrors = append(validationErrors, fieldErr.Field()+" is required")
		}

		_ = utils.WriteJSONResponseWithStatus(w, http.StatusBadRequest, map[string]interface{}{
			"errorCode": constants.ErrorCodeBadRequest,
			"context":   "file-system",
			"errors":    validationErrors,
		})
		return
	}

	sourcePath := filepath.Clean(req.SourcePath)
	sourceInfo, err := s.fileEditor.Stat(sourcePath)
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

	if sourceInfo.IsDir() {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusBadRequest, map[string]interface{}{
			"errorCode": constants.ErrorCodePathIsDirectory,
			"context":   "file-system",
			"errors":    []string{"Path is a directory, not a file"},
		})
		return
	}

	targetPath, err := resolveDuplicateTargetPath(sourcePath, req.NewFileName)
	if err != nil {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusBadRequest, map[string]interface{}{
			"errorCode": constants.ErrorCodeBadRequest,
			"context":   "file-system",
			"errors":    []string{err.Error()},
		})
		return
	}

	if _, err := s.fileEditor.Stat(targetPath); err == nil {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusBadRequest, map[string]interface{}{
			"errorCode": constants.ErrorCodeBadRequest,
			"context":   "file-system",
			"errors":    []string{"A file with this name already exists"},
		})
		return
	} else if !s.fileEditor.IsNotExist(err) {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusInternalServerError, map[string]interface{}{
			"errorCode": constants.ErrorCodeFileReadError,
			"context":   "file-system",
			"errors":    []string{"Cannot check destination file: " + err.Error()},
		})
		return
	}

	if err := duplicateFileContents(s.fileEditor, sourcePath, targetPath, sourceInfo.Mode().Perm()); err != nil {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusInternalServerError, map[string]interface{}{
			"errorCode": constants.ErrorCodeInternalServerError,
			"context":   "file-system",
			"errors":    []string{"Failed to duplicate file: " + err.Error()},
		})
		return
	}

	_ = utils.WriteJSONResponse(w, map[string]interface{}{
		"message":         "File duplicated successfully",
		"duplicated_path": targetPath,
	})
}

func (s *Server) releaseFileLock(lockPath string) {
	if lockPath != "" {
		if err := s.fileEditor.Remove(lockPath); err != nil {
			s.log.Error("Failed to remove lock file", logger.Field{Key: "error", Value: err})
		}
	}
}

func resolveDuplicateTargetPath(sourcePath string, newFileName string) (string, error) {
	if err := validateDuplicateFileName(newFileName); err != nil {
		return "", err
	}

	sourceDir := filepath.Clean(filepath.Dir(sourcePath))
	targetPath := filepath.Clean(filepath.Join(sourceDir, strings.TrimSpace(newFileName)))
	targetDir := filepath.Clean(filepath.Dir(targetPath))

	if targetDir != sourceDir {
		return "", errors.New("new file name must stay in the same directory")
	}

	return targetPath, nil
}

func validateDuplicateFileName(newFileName string) error {
	trimmedName := strings.TrimSpace(newFileName)
	if trimmedName == "" {
		return errors.New("new_file_name is required")
	}

	if strings.ContainsAny(trimmedName, `/\`) {
		return errors.New("new file name cannot contain path separators")
	}

	baseName := filepath.Base(trimmedName)
	if baseName != trimmedName || baseName == "." || baseName == ".." {
		return errors.New("new file name is invalid")
	}

	return nil
}

func duplicateFileContents(fileEditor duplicateFileEditor, sourcePath string, targetPath string, perm fs.FileMode) error {
	content, err := fileEditor.ReadFile(sourcePath)
	if err != nil {
		return err
	}

	return fileEditor.WriteFile(targetPath, content, perm)
}

func validateItemCombinationDataRequest(req ItemCombinationDataFileAPIData) []string {
	var errs []string
	if req.Formulas == nil {
		errs = append(errs, "formulas is required")
		return errs
	}

	for i, formula := range req.Formulas {
		if len(formula.Ingredients) != itemCombinationIngredientCount {
			errs = append(errs, fmt.Sprintf("formulas[%d]: ingredients must contain exactly %d items", i, itemCombinationIngredientCount))
		} else {
			for j, ingredient := range formula.Ingredients {
				if ingredient == nil {
					errs = append(errs, fmt.Sprintf("formulas[%d]: ingredients[%d] is required", i, j))
				}
			}
		}

		if formula.SuccessRate == nil {
			errs = append(errs, fmt.Sprintf("formulas[%d]: success_rate is required", i))
		} else if *formula.SuccessRate == 0 || *formula.SuccessRate > maxItemCombinationSuccessRate {
			errs = append(errs, fmt.Sprintf("formulas[%d]: success_rate must be between 1 and %d", i, maxItemCombinationSuccessRate))
		}

		if formula.Outcome == nil {
			errs = append(errs, fmt.Sprintf("formulas[%d]: outcome is required", i))
		}
	}

	return errs
}

func validateQuestFileRequest(req QuestFileAPIData) []string {
	var errs []string
	if *req.MinLevel != 0xff && *req.MaxLevel != 0xff && *req.MinLevel > *req.MaxLevel {
		errs = append(errs, "min_level must be less than or equal to max_level")
	}
	for i := 0; i < 7 && i < len(req.Objectives); i++ {
		obj := req.Objectives[i]
		if obj.Type == nil {
			errs = append(errs, fmt.Sprintf("objectives[%d]: type is required", i))
			continue
		}

		if obj.Location == nil || obj.Location.X == nil || obj.Location.Y == nil {
			errs = append(errs, fmt.Sprintf("objectives[%d]: location.x and location.y are required", i))
			continue
		}

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

func uint16Ptr(value uint16) *uint16 {
	return &value
}

func uint8Ptr(value uint8) *uint8 {
	return &value
}

func uint32Ptr(value uint32) *uint32 {
	return &value
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

type TextFileUpdateAPIData struct {
	Content *string `json:"content" validate:"required"`
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

type DropFileAPIData struct {
	Drops []DropAPIData `json:"drops" validate:"required,dive"`
}

type DropAPIData struct {
	ItemCode  *uint16 `json:"item_code" validate:"required"`
	DropRate  *uint16 `json:"drop_rate" validate:"required"`
	DropGroup *uint16 `json:"drop_group" validate:"required"`
}

type ItemFileAPIData struct {
	ItemFileType string                    `json:"item_file_type"`
	NameEncoding string                    `json:"name_encoding,omitempty"`
	Items        []ItemFileItemAPIData     `json:"items"`
	BaseItems    []ItemFileBaseItemAPIData `json:"base_items,omitempty"`
}

type ItemFileItemAPIData struct {
	ItemCodeBase  *uint16                `json:"item_code_base,omitempty"`
	Row           *uint16                `json:"row,omitempty"`
	Slot          *uint16                `json:"slot,omitempty"`
	Type          *uint16                `json:"type,omitempty"`
	ItemCode      *uint32                `json:"item_code,omitempty"`
	Name          string                 `json:"name"`
	NPCPrice      *uint32                `json:"npc_price,omitempty"`
	RequiredLevel *uint16                `json:"required_level,omitempty"`
	Attribute     *uint16                `json:"attribute,omitempty"`
	BlueOption    *uint16                `json:"blue_option,omitempty"`
	RedOption     *uint16                `json:"red_option,omitempty"`
	GreyOption    *uint16                `json:"grey_option,omitempty"`
	Class         *uint16                `json:"class,omitempty"`
	SkillLevel    *uint16                `json:"skill_level,omitempty"`
	Levels        []ItemFileLevelAPIData `json:"levels,omitempty"`
}

type ItemFileLevelAPIData struct {
	Level               *uint8  `json:"level,omitempty"`
	AdditionalAttribute *uint16 `json:"additional_attribute,omitempty"`
	Strength            *uint16 `json:"strength,omitempty"`
	Dexterity           *uint16 `json:"dexterity,omitempty"`
	Intelligence        *uint16 `json:"intelligence,omitempty"`
	Attribute           *uint16 `json:"attribute,omitempty"`
	AttributeRange      *uint16 `json:"attribute_range,omitempty"`
	BlueOption          *uint16 `json:"blue_option,omitempty"`
	RedOption           *uint16 `json:"red_option,omitempty"`
	GreyOption          *uint16 `json:"grey_option,omitempty"`
}

type ItemFileBaseItemAPIData struct {
	Row      *uint16                `json:"row,omitempty"`
	ItemCode *uint32                `json:"item_code,omitempty"`
	Name     string                 `json:"name"`
	Levels   []ItemFileLevelAPIData `json:"levels,omitempty"`
}

type itemFileNameEncoding struct {
	label    string
	encoding textencoding.Encoding
}

type ItemCombinationDataFileAPIData struct {
	Formulas []ItemCombinationFormulaAPIData `json:"formulas" validate:"required,dive"`
}

type ItemCombinationFormulaAPIData struct {
	Ingredients []*uint16 `json:"ingredients" validate:"required,len=10,dive,required"`
	SuccessRate *uint16   `json:"success_rate" validate:"required"`
	Outcome     *uint16   `json:"outcome" validate:"required"`
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
	Objectives    []ObjectiveAPIData `json:"objectives" validate:"required,len=7,dive"`
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

type DuplicateFileRequest struct {
	SourcePath  string `json:"source_path" validate:"required"`
	NewFileName string `json:"new_file_name" validate:"required"`
}

type duplicateFileEditor interface {
	ReadFile(name string) ([]byte, error)
	WriteFile(name string, data []byte, perm fs.FileMode) error
}
