package services

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/omnihance/omnihance-a3-agent/internal/constants"
	"github.com/omnihance/omnihance-a3-agent/internal/db"
	"github.com/omnihance/omnihance-a3-agent/internal/logger"
	"github.com/project-agonyl/agonyl-utils-go/dropfile"
	"gopkg.in/ini.v1"
)

const (
	serverInfoFileName    = "SvrInfo.ini"
	serverViewMapInfoName = "MapInfo.ini"
	serverViewGMInfoName  = "GMInfo.ini"
	serverViewMapDataDir  = "ZoneData/map"
	serverViewNPCDataDir  = "ZoneData/npc"
	serverViewShopDataDir = "ZoneData/shop"
	serverViewItemIDMask  = 0x3fff
	serverViewEmptyItemID = 0xffff
)

var ErrServerViewSyncRunning = errors.New("server view sync is already running")

type ServerViewService interface {
	Start() error
	GetOverview(ctx context.Context) (*ServerViewOverview, error)
	GetSyncStatus(ctx context.Context) (*ServerViewSyncStatus, error)
	StartSync(ctx context.Context, userID *int64) (*db.ServerViewSyncRun, error)
	GetMainMaps(ctx context.Context, query string) ([]ServerViewMapRow, error)
	GetZoneMaps(ctx context.Context, query string) ([]ServerViewMapRow, error)
	GetZoneSpawns(ctx context.Context, mapQuery string, npcQuery string) ([]ServerViewSpawnSummaryRow, error)
	GetZoneDrops(ctx context.Context, query string) ([]ServerViewDropSummaryRow, error)
	GetZoneDropDetails(ctx context.Context, npcID int64, query string) ([]ServerViewDropDetailRow, error)
	GetZoneShops(ctx context.Context, query string) ([]ServerViewShopSummaryRow, error)
	GetZoneShopDetails(ctx context.Context, npcID int64, query string) ([]ServerViewShopDetailRow, error)
}

type serverViewService struct {
	logger     logger.Logger
	internalDB db.InternalDB
	fileEditor FileEditorService
	mu         sync.Mutex
	running    bool
}

type ServerViewOverview struct {
	Servers []ServerViewServerInfo `json:"servers"`
	Sync    ServerViewSyncStatus   `json:"sync"`
}

type ServerViewServerInfo struct {
	ServerType        string                  `json:"server_type"`
	Label             string                  `json:"label"`
	SettingKey        string                  `json:"setting_key"`
	Configured        bool                    `json:"configured"`
	Path              *string                 `json:"path"`
	Sections          []ServerViewInfoSection `json:"sections"`
	AvailableActions  []ServerViewAction      `json:"available_actions"`
	UnavailableReason *string                 `json:"unavailable_reason"`
}

type ServerViewInfoSection struct {
	Name string              `json:"name"`
	Rows []ServerViewInfoRow `json:"rows"`
}

type ServerViewInfoRow struct {
	Key        string `json:"key"`
	Value      string `json:"value"`
	ValueIndex int    `json:"value_index"`
}

type ServerViewAction struct {
	Label string `json:"label"`
	Href  string `json:"href"`
}

type ServerViewSyncStatus struct {
	Running    bool                          `json:"running"`
	LatestRun  *db.ServerViewSyncRun         `json:"latest_run"`
	Warnings   []db.ServerViewSyncWarning    `json:"warnings"`
	Missing    []ServerViewMissingSetting    `json:"missing_settings"`
	Configured []ServerViewConfiguredSetting `json:"configured_settings"`
}

type ServerViewMissingSetting struct {
	ServerType string `json:"server_type"`
	SettingKey string `json:"setting_key"`
	Label      string `json:"label"`
}

type ServerViewConfiguredSetting struct {
	ServerType string `json:"server_type"`
	SettingKey string `json:"setting_key"`
	Label      string `json:"label"`
	Path       string `json:"path"`
}

type ServerViewMapRow struct {
	MapID      int64   `json:"map_id"`
	MapName    *string `json:"map_name"`
	MapDisplay string  `json:"map_display"`
	ZoneID     int64   `json:"zone_id"`
}

type ServerViewSpawnSummaryRow struct {
	MapID      int64   `json:"map_id"`
	MapName    *string `json:"map_name"`
	MapDisplay string  `json:"map_display"`
	NPCID      int64   `json:"npc_id"`
	NPCName    *string `json:"npc_name"`
	NPCDisplay string  `json:"npc_display"`
	Count      int64   `json:"count"`
}

type ServerViewDropSummaryRow struct {
	NPCID         int64   `json:"npc_id"`
	NPCName       *string `json:"npc_name"`
	NPCDisplay    string  `json:"npc_display"`
	DropItemCount int64   `json:"drop_item_count"`
}

type ServerViewDropDetailRow struct {
	RowIndex    int     `json:"row_index"`
	ItemID      int64   `json:"item_id"`
	ItemName    *string `json:"item_name"`
	ItemDisplay string  `json:"item_display"`
	DropRate    int     `json:"drop_rate"`
	GroupCode   int     `json:"group_code"`
}

type ServerViewShopSummaryRow struct {
	NPCID      int64   `json:"npc_id"`
	NPCName    *string `json:"npc_name"`
	NPCDisplay string  `json:"npc_display"`
	ItemCount  int64   `json:"item_count"`
}

type ServerViewShopDetailRow struct {
	LineNumber  int     `json:"line_number"`
	ItemID      int64   `json:"item_id"`
	ItemName    *string `json:"item_name"`
	ItemDisplay string  `json:"item_display"`
}

type serverViewPathDefinition struct {
	ServerType string
	SettingKey string
	Label      string
}

type serverViewPaths struct {
	ByType     map[string]string
	Missing    []ServerViewMissingSetting
	Configured []ServerViewConfiguredSetting
}

func NewServerViewService(logger logger.Logger, internalDB db.InternalDB, fileEditor FileEditorService) ServerViewService {
	return &serverViewService{
		logger:     logger,
		internalDB: internalDB,
		fileEditor: fileEditor,
	}
}

func (s *serverViewService) Start() error {
	return s.internalDB.MarkOrphanedServerViewSyncRunsFailed()
}

func (s *serverViewService) GetOverview(ctx context.Context) (*ServerViewOverview, error) {
	_ = ctx

	paths, err := s.serverPaths()
	if err != nil {
		return nil, err
	}

	infoRows, err := s.internalDB.GetServerViewSvrInfoRows()
	if err != nil {
		return nil, err
	}

	rowsByServer := groupSvrInfoRows(infoRows)
	servers := make([]ServerViewServerInfo, 0, len(serverViewPathDefinitions()))
	for _, definition := range serverViewPathDefinitions() {
		path, configured := paths.ByType[definition.ServerType]
		sections := rowsByServer[definition.ServerType]
		if sections == nil {
			sections = []ServerViewInfoSection{}
		}

		serverInfo := ServerViewServerInfo{
			ServerType:       definition.ServerType,
			Label:            definition.Label,
			SettingKey:       definition.SettingKey,
			Configured:       configured,
			Sections:         sections,
			AvailableActions: serverViewActions(definition.ServerType),
		}

		if configured {
			serverInfo.Path = &path
		} else {
			reason := "Set " + definition.Label + " path in Settings before syncing this card."
			serverInfo.UnavailableReason = &reason
		}

		servers = append(servers, serverInfo)
	}

	syncStatus, err := s.GetSyncStatus(ctx)
	if err != nil {
		return nil, err
	}

	return &ServerViewOverview{
		Servers: servers,
		Sync:    *syncStatus,
	}, nil
}

func (s *serverViewService) GetSyncStatus(ctx context.Context) (*ServerViewSyncStatus, error) {
	_ = ctx

	paths, err := s.serverPaths()
	if err != nil {
		return nil, err
	}

	latestRun, err := s.internalDB.GetLatestServerViewSyncRun()
	if err != nil {
		return nil, err
	}

	runningRun, err := s.internalDB.GetRunningServerViewSyncRun()
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	instanceRunning := s.running
	s.mu.Unlock()

	warnings := make([]db.ServerViewSyncWarning, 0)
	if latestRun != nil {
		warnings, err = s.internalDB.GetServerViewSyncWarnings(latestRun.ID)
		if err != nil {
			return nil, err
		}
	}

	return &ServerViewSyncStatus{
		Running:    runningRun != nil || instanceRunning,
		LatestRun:  latestRun,
		Warnings:   warnings,
		Missing:    paths.Missing,
		Configured: paths.Configured,
	}, nil
}

func (s *serverViewService) StartSync(ctx context.Context, userID *int64) (*db.ServerViewSyncRun, error) {
	_ = ctx

	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return nil, ErrServerViewSyncRunning
	}

	runningRun, err := s.internalDB.GetRunningServerViewSyncRun()
	if err != nil {
		s.mu.Unlock()
		return nil, err
	}

	if runningRun != nil {
		s.mu.Unlock()
		return nil, ErrServerViewSyncRunning
	}

	run, err := s.internalDB.CreateServerViewSyncRun(userID)
	if err != nil {
		s.mu.Unlock()
		if errors.Is(err, db.ErrServerViewSyncAlreadyRunning) {
			return nil, ErrServerViewSyncRunning
		}

		return nil, err
	}

	s.running = true
	s.mu.Unlock()

	go s.executeSync(run.ID)

	return run, nil
}

func (s *serverViewService) GetMainMaps(ctx context.Context, query string) ([]ServerViewMapRow, error) {
	return s.getMaps(ctx, db.ServerViewServerTypeMain, query)
}

func (s *serverViewService) GetZoneMaps(ctx context.Context, query string) ([]ServerViewMapRow, error) {
	return s.getMaps(ctx, db.ServerViewServerTypeZone, query)
}

func (s *serverViewService) GetZoneSpawns(ctx context.Context, mapQuery string, npcQuery string) ([]ServerViewSpawnSummaryRow, error) {
	_ = ctx

	rows, err := s.internalDB.GetServerViewSpawnRows()
	if err != nil {
		return nil, err
	}

	mapLookup, monsterLookup, _, err := s.clientLookups()
	if err != nil {
		return nil, err
	}

	grouped := make(map[string]ServerViewSpawnSummaryRow)
	for _, row := range rows {
		key := strconv.FormatInt(row.MapID, 10) + ":" + strconv.FormatInt(row.NPCID, 10)
		summary := grouped[key]
		if summary.Count == 0 {
			mapName := mapLookup[row.MapID]
			npcName := monsterLookup[row.NPCID]
			summary = ServerViewSpawnSummaryRow{
				MapID:      row.MapID,
				MapName:    stringPointerIfNotEmpty(mapName),
				MapDisplay: displayNameID(row.MapID, mapName),
				NPCID:      row.NPCID,
				NPCName:    stringPointerIfNotEmpty(npcName),
				NPCDisplay: displayNameID(row.NPCID, npcName),
			}
		}

		summary.Count++
		grouped[key] = summary
	}

	result := make([]ServerViewSpawnSummaryRow, 0, len(grouped))
	for _, row := range grouped {
		if matchesIDOrName(mapQuery, row.MapID, stringValueOrEmpty(row.MapName)) && matchesIDOrName(npcQuery, row.NPCID, stringValueOrEmpty(row.NPCName)) {
			result = append(result, row)
		}
	}

	sort.Slice(result, func(i, j int) bool {
		if result[i].MapID == result[j].MapID {
			return result[i].NPCID < result[j].NPCID
		}

		return result[i].MapID < result[j].MapID
	})

	return result, nil
}

func (s *serverViewService) GetZoneDrops(ctx context.Context, query string) ([]ServerViewDropSummaryRow, error) {
	_ = ctx

	rows, err := s.internalDB.GetServerViewDropRows()
	if err != nil {
		return nil, err
	}

	_, monsterLookup, itemLookup, err := s.clientLookups()
	if err != nil {
		return nil, err
	}

	grouped := make(map[int64]ServerViewDropSummaryRow)
	itemNamesByNPC := make(map[int64][]string)
	itemIDsByNPC := make(map[int64][]int64)
	for _, row := range rows {
		summary := grouped[row.NPCID]
		if summary.DropItemCount == 0 {
			npcName := monsterLookup[row.NPCID]
			summary = ServerViewDropSummaryRow{
				NPCID:      row.NPCID,
				NPCName:    stringPointerIfNotEmpty(npcName),
				NPCDisplay: displayNameID(row.NPCID, npcName),
			}
		}

		summary.DropItemCount++
		grouped[row.NPCID] = summary
		itemIDsByNPC[row.NPCID] = append(itemIDsByNPC[row.NPCID], row.ItemID, itemBaseID(row.ItemID))
		if itemName := itemLookup[itemBaseID(row.ItemID)]; itemName != "" {
			itemNamesByNPC[row.NPCID] = append(itemNamesByNPC[row.NPCID], itemName)
		}
	}

	result := make([]ServerViewDropSummaryRow, 0, len(grouped))
	for npcID, row := range grouped {
		if matchesNPCOrItemQuery(query, row.NPCID, stringValueOrEmpty(row.NPCName), itemIDsByNPC[npcID], itemNamesByNPC[npcID]) {
			result = append(result, row)
		}
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].NPCID < result[j].NPCID
	})

	return result, nil
}

func (s *serverViewService) GetZoneDropDetails(ctx context.Context, npcID int64, query string) ([]ServerViewDropDetailRow, error) {
	_ = ctx

	rows, err := s.internalDB.GetServerViewDropRows()
	if err != nil {
		return nil, err
	}

	_, _, itemLookup, err := s.clientLookups()
	if err != nil {
		return nil, err
	}

	result := make([]ServerViewDropDetailRow, 0)
	for _, row := range rows {
		if row.NPCID != npcID {
			continue
		}

		itemName := itemLookup[itemBaseID(row.ItemID)]
		if !matchesIDOrName(query, row.ItemID, itemName) && !matchesIDOrName(query, itemBaseID(row.ItemID), itemName) {
			continue
		}

		result = append(result, ServerViewDropDetailRow{
			RowIndex:    row.RowIndex,
			ItemID:      row.ItemID,
			ItemName:    stringPointerIfNotEmpty(itemName),
			ItemDisplay: displayNameID(row.ItemID, itemName),
			DropRate:    row.DropRate,
			GroupCode:   row.GroupCode,
		})
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].RowIndex < result[j].RowIndex
	})

	return result, nil
}

func (s *serverViewService) GetZoneShops(ctx context.Context, query string) ([]ServerViewShopSummaryRow, error) {
	_ = ctx

	rows, err := s.internalDB.GetServerViewShopRows()
	if err != nil {
		return nil, err
	}

	_, monsterLookup, itemLookup, err := s.clientLookups()
	if err != nil {
		return nil, err
	}

	grouped := make(map[int64]ServerViewShopSummaryRow)
	itemNamesByNPC := make(map[int64][]string)
	itemIDsByNPC := make(map[int64][]int64)
	for _, row := range rows {
		summary := grouped[row.NPCID]
		if summary.ItemCount == 0 {
			npcName := monsterLookup[row.NPCID]
			summary = ServerViewShopSummaryRow{
				NPCID:      row.NPCID,
				NPCName:    stringPointerIfNotEmpty(npcName),
				NPCDisplay: displayNameID(row.NPCID, npcName),
			}
		}

		summary.ItemCount++
		grouped[row.NPCID] = summary
		itemIDsByNPC[row.NPCID] = append(itemIDsByNPC[row.NPCID], row.ItemID, itemBaseID(row.ItemID))
		if itemName := itemLookup[itemBaseID(row.ItemID)]; itemName != "" {
			itemNamesByNPC[row.NPCID] = append(itemNamesByNPC[row.NPCID], itemName)
		}
	}

	result := make([]ServerViewShopSummaryRow, 0, len(grouped))
	for npcID, row := range grouped {
		if matchesNPCOrItemQuery(query, row.NPCID, stringValueOrEmpty(row.NPCName), itemIDsByNPC[npcID], itemNamesByNPC[npcID]) {
			result = append(result, row)
		}
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].NPCID < result[j].NPCID
	})

	return result, nil
}

func (s *serverViewService) GetZoneShopDetails(ctx context.Context, npcID int64, query string) ([]ServerViewShopDetailRow, error) {
	_ = ctx

	rows, err := s.internalDB.GetServerViewShopRows()
	if err != nil {
		return nil, err
	}

	_, _, itemLookup, err := s.clientLookups()
	if err != nil {
		return nil, err
	}

	result := make([]ServerViewShopDetailRow, 0)
	for _, row := range rows {
		if row.NPCID != npcID {
			continue
		}

		itemName := itemLookup[itemBaseID(row.ItemID)]
		if !matchesIDOrName(query, row.ItemID, itemName) && !matchesIDOrName(query, itemBaseID(row.ItemID), itemName) {
			continue
		}

		result = append(result, ServerViewShopDetailRow{
			LineNumber:  row.LineNumber,
			ItemID:      row.ItemID,
			ItemName:    stringPointerIfNotEmpty(itemName),
			ItemDisplay: displayNameID(row.ItemID, itemName),
		})
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].LineNumber < result[j].LineNumber
	})

	return result, nil
}

func (s *serverViewService) executeSync(runID int64) {
	defer func() {
		s.mu.Lock()
		s.running = false
		s.mu.Unlock()
	}()

	warningCount := 0
	warn := func(source string, err error) {
		if err == nil {
			return
		}

		warningCount++
		message := err.Error()
		if addErr := s.internalDB.AddServerViewSyncWarning(runID, source, message); addErr != nil {
			s.logger.Error("failed to record server view sync warning", logger.Field{Key: "source", Value: source}, logger.Field{Key: "error", Value: addErr})
		}
	}

	paths, err := s.serverPaths()
	if err != nil {
		warn("settings", err)
		errorDetails := err.Error()
		_ = s.internalDB.FinishServerViewSyncRun(runID, db.ServerViewSyncStatusFailed, warningCount, &errorDetails)
		return
	}

	for _, missing := range paths.Missing {
		warn(missing.Label, fmt.Errorf("%s is not configured", missing.SettingKey))
	}

	s.syncSvrInfo(paths, warn)
	s.syncMapInfo(paths, warn)
	s.syncSpawns(paths, warn)
	s.syncDrops(paths, warn)
	s.syncShops(paths, warn)
	s.syncGameMasters(paths, warn)

	if err := s.internalDB.FinishServerViewSyncRun(runID, db.ServerViewSyncStatusSucceeded, warningCount, nil); err != nil {
		s.logger.Error("failed to finish server view sync", logger.Field{Key: "run_id", Value: runID}, logger.Field{Key: "error", Value: err})
	}
}

func (s *serverViewService) syncSvrInfo(paths serverViewPaths, warn func(string, error)) {
	for _, definition := range serverViewPathDefinitions() {
		basePath, ok := paths.ByType[definition.ServerType]
		if !ok {
			continue
		}

		source := filepath.Join(basePath, serverInfoFileName)
		cfg, err := s.loadINI(source)
		if err != nil {
			warn(source, err)
			continue
		}

		rows := svrInfoRowsFromINI(definition.ServerType, cfg)
		if err := s.internalDB.ReplaceServerViewSvrInfoRows(definition.ServerType, rows); err != nil {
			warn(source, err)
		}
	}
}

func (s *serverViewService) syncMapInfo(paths serverViewPaths, warn func(string, error)) {
	if mainPath, ok := paths.ByType[db.ServerViewServerTypeMain]; ok {
		source := filepath.Join(mainPath, serverViewMapInfoName)
		rows, err := s.mapRowsFromINIFile(db.ServerViewServerTypeMain, source)
		if err != nil {
			warn(source, err)
		} else if err := s.internalDB.ReplaceServerViewMapZones(db.ServerViewServerTypeMain, rows); err != nil {
			warn(source, err)
		}
	}

	if zonePath, ok := paths.ByType[db.ServerViewServerTypeZone]; ok {
		source := filepath.Join(zonePath, filepath.FromSlash(serverViewMapDataDir), serverViewMapInfoName)
		rows, err := s.mapRowsFromINIFile(db.ServerViewServerTypeZone, source)
		if err != nil {
			warn(source, err)
		} else if err := s.internalDB.ReplaceServerViewMapZones(db.ServerViewServerTypeZone, rows); err != nil {
			warn(source, err)
		}
	}
}

func (s *serverViewService) syncSpawns(paths serverViewPaths, warn func(string, error)) {
	zonePath, ok := paths.ByType[db.ServerViewServerTypeZone]
	if !ok {
		return
	}

	dir := filepath.Join(zonePath, filepath.FromSlash(serverViewMapDataDir))
	entries, err := s.fileEditor.ReadDir(dir)
	if err != nil {
		warn(dir, err)
		return
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.EqualFold(filepath.Ext(entry.Name()), SpawnFileExtension) {
			continue
		}

		mapID, err := parseNumericStem(entry.Name())
		source := filepath.Join(dir, entry.Name())
		if err != nil {
			warn(source, err)
			continue
		}

		spawns, err := s.fileEditor.ReadSpawnFileData(source)
		if err != nil {
			warn(source, err)
			continue
		}

		rows := make([]db.ServerViewSpawnRow, 0, len(spawns))
		for index, spawn := range spawns {
			rows = append(rows, db.ServerViewSpawnRow{
				MapID:       mapID,
				RowIndex:    index + 1,
				NPCID:       int64(spawn.Id),
				X:           int(spawn.X),
				Y:           int(spawn.Y),
				Unknown1:    int(spawn.Unknown1),
				Orientation: int(spawn.Orientation),
				SpawnStep:   int(spawn.SpwanStep),
			})
		}

		if err := s.internalDB.ReplaceServerViewSpawnRowsForMap(mapID, rows); err != nil {
			warn(source, err)
		}
	}
}

func (s *serverViewService) syncDrops(paths serverViewPaths, warn func(string, error)) {
	zonePath, ok := paths.ByType[db.ServerViewServerTypeZone]
	if !ok {
		return
	}

	dir := filepath.Join(zonePath, filepath.FromSlash(serverViewNPCDataDir))
	entries, err := s.fileEditor.ReadDir(dir)
	if err != nil {
		warn(dir, err)
		return
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.EqualFold(filepath.Ext(entry.Name()), DropFileExtension) {
			continue
		}

		npcID, err := parseNumericStem(entry.Name())
		source := filepath.Join(dir, entry.Name())
		if err != nil {
			warn(source, err)
			continue
		}

		drops, err := s.fileEditor.ReadDropFileData(source)
		if err != nil {
			warn(source, err)
			continue
		}

		rows := make([]db.ServerViewDropRow, 0, len(drops))
		for index, drop := range drops {
			if drop.ItemID == dropfile.EmptyItemID || drop.ItemID == serverViewEmptyItemID {
				continue
			}

			rows = append(rows, db.ServerViewDropRow{
				NPCID:     npcID,
				RowIndex:  index + 1,
				ItemID:    int64(drop.ItemID),
				DropRate:  int(drop.DropRate),
				GroupCode: int(drop.DropGroup),
			})
		}

		if err := s.internalDB.ReplaceServerViewDropRowsForNPC(npcID, rows); err != nil {
			warn(source, err)
		}
	}
}

func (s *serverViewService) syncShops(paths serverViewPaths, warn func(string, error)) {
	zonePath, ok := paths.ByType[db.ServerViewServerTypeZone]
	if !ok {
		return
	}

	dir := filepath.Join(zonePath, filepath.FromSlash(serverViewShopDataDir))
	entries, err := s.fileEditor.ReadDir(dir)
	if err != nil {
		warn(dir, err)
		return
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.EqualFold(filepath.Ext(entry.Name()), ".txt") {
			continue
		}

		npcID, err := parseNumericStem(entry.Name())
		source := filepath.Join(dir, entry.Name())
		if err != nil {
			warn(source, err)
			continue
		}

		rows, err := s.shopRowsFromFile(npcID, source)
		if err != nil {
			warn(source, err)
			continue
		}

		if err := s.internalDB.ReplaceServerViewShopRowsForNPC(npcID, rows); err != nil {
			warn(source, err)
		}
	}
}

func (s *serverViewService) syncGameMasters(paths serverViewPaths, warn func(string, error)) {
	zonePath, ok := paths.ByType[db.ServerViewServerTypeZone]
	if !ok {
		return
	}

	source := filepath.Join(zonePath, serverViewGMInfoName)
	cfg, err := s.loadINI(source)
	if err != nil {
		warn(source, err)
		return
	}

	rows := gameMasterRowsFromINI(cfg)
	if err := s.internalDB.ReplaceServerViewGameMasterRows(rows); err != nil {
		warn(source, err)
	}
}

func (s *serverViewService) getMaps(ctx context.Context, serverType string, query string) ([]ServerViewMapRow, error) {
	_ = ctx

	rows, err := s.internalDB.GetServerViewMapZones(serverType)
	if err != nil {
		return nil, err
	}

	mapLookup, _, _, err := s.clientLookups()
	if err != nil {
		return nil, err
	}

	result := make([]ServerViewMapRow, 0, len(rows))
	for _, row := range rows {
		mapName := mapLookup[row.MapID]
		if !matchesIDOrName(query, row.MapID, mapName) {
			continue
		}

		result = append(result, ServerViewMapRow{
			MapID:      row.MapID,
			MapName:    stringPointerIfNotEmpty(mapName),
			MapDisplay: displayNameID(row.MapID, mapName),
			ZoneID:     row.ZoneID,
		})
	}

	return result, nil
}

func (s *serverViewService) loadINI(path string) (*ini.File, error) {
	data, err := s.fileEditor.ReadFile(path)
	if err != nil {
		return nil, err
	}

	cfg, err := ini.LoadSources(ini.LoadOptions{
		AllowShadows: true,
	}, data)
	if err != nil {
		return nil, err
	}

	return cfg, nil
}

func (s *serverViewService) mapRowsFromINIFile(serverType string, path string) ([]db.ServerViewMapZone, error) {
	cfg, err := s.loadINI(path)
	if err != nil {
		return nil, err
	}

	return mapRowsFromINI(serverType, cfg), nil
}

func mapRowsFromINI(serverType string, cfg *ini.File) []db.ServerViewMapZone {
	section := findINISection(cfg, "MapZone")
	if section == nil {
		return []db.ServerViewMapZone{}
	}

	mapZones := make(map[int64]int64)
	for _, key := range section.Keys() {
		keyName := strings.TrimSpace(key.Name())
		if !strings.HasPrefix(strings.ToLower(keyName), "map") {
			continue
		}

		mapID, err := strconv.ParseInt(strings.TrimSpace(keyName[3:]), 10, 64)
		if err != nil {
			continue
		}

		zoneID, err := strconv.ParseInt(strings.TrimSpace(key.String()), 10, 64)
		if err != nil {
			continue
		}

		mapZones[mapID] = zoneID
	}

	mapIDs := make([]int64, 0, len(mapZones))
	for mapID := range mapZones {
		mapIDs = append(mapIDs, mapID)
	}
	sort.Slice(mapIDs, func(i, j int) bool {
		return mapIDs[i] < mapIDs[j]
	})

	rows := make([]db.ServerViewMapZone, 0, len(mapIDs))
	for _, mapID := range mapIDs {
		rows = append(rows, db.ServerViewMapZone{
			ServerType: serverType,
			MapID:      mapID,
			ZoneID:     mapZones[mapID],
		})
	}

	return rows
}

func (s *serverViewService) shopRowsFromFile(npcID int64, path string) ([]db.ServerViewShopRow, error) {
	data, err := s.fileEditor.ReadFile(path)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n")
	rows := make([]db.ServerViewShopRow, 0, len(lines))
	for index, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		lineNumber := index + 1
		itemID, err := strconv.ParseInt(trimmed, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid shop item ID on line %d: %w", lineNumber, err)
		}

		rows = append(rows, db.ServerViewShopRow{
			NPCID:      npcID,
			LineNumber: lineNumber,
			ItemID:     itemID,
		})
	}

	return rows, nil
}

func (s *serverViewService) serverPaths() (serverViewPaths, error) {
	paths := serverViewPaths{
		ByType:     map[string]string{},
		Missing:    []ServerViewMissingSetting{},
		Configured: []ServerViewConfiguredSetting{},
	}

	for _, definition := range serverViewPathDefinitions() {
		setting, err := s.internalDB.GetSetting(definition.SettingKey)
		if err != nil {
			if errors.Is(err, db.ErrSettingNotFound) {
				paths.Missing = append(paths.Missing, ServerViewMissingSetting(definition))
				continue
			}

			return paths, err
		}

		value := strings.TrimSpace(setting.Value)
		if value == "" {
			paths.Missing = append(paths.Missing, ServerViewMissingSetting(definition))
			continue
		}

		cleanPath := filepath.Clean(value)
		paths.ByType[definition.ServerType] = cleanPath
		paths.Configured = append(paths.Configured, ServerViewConfiguredSetting{
			ServerType: definition.ServerType,
			SettingKey: definition.SettingKey,
			Label:      definition.Label,
			Path:       cleanPath,
		})
	}

	return paths, nil
}

func (s *serverViewService) clientLookups() (map[int64]string, map[int64]string, map[int64]string, error) {
	mapData, err := s.internalDB.GetAllMapClientData("")
	if err != nil {
		return nil, nil, nil, err
	}

	monsterData, err := s.internalDB.GetAllMonsterClientData("")
	if err != nil {
		return nil, nil, nil, err
	}

	itemData, err := s.internalDB.GetAllItemClientData("")
	if err != nil {
		return nil, nil, nil, err
	}

	mapLookup := make(map[int64]string, len(mapData))
	for _, item := range mapData {
		mapLookup[item.ID] = item.Name
	}

	monsterLookup := make(map[int64]string, len(monsterData))
	for _, item := range monsterData {
		monsterLookup[item.ID] = item.Name
	}

	itemLookup := make(map[int64]string, len(itemData))
	for _, item := range itemData {
		baseID := item.ID & serverViewItemIDMask
		if _, ok := itemLookup[baseID]; !ok {
			itemLookup[baseID] = item.Name
		}
	}

	return mapLookup, monsterLookup, itemLookup, nil
}

func serverViewPathDefinitions() []serverViewPathDefinition {
	return []serverViewPathDefinition{
		{ServerType: db.ServerViewServerTypeMain, SettingKey: constants.SettingKeyMainServerPath, Label: "Main Server"},
		{ServerType: db.ServerViewServerTypeAccount, SettingKey: constants.SettingKeyAccountServerPath, Label: "Account Server"},
		{ServerType: db.ServerViewServerTypeZone, SettingKey: constants.SettingKeyZoneServerPath, Label: "Zone Server"},
		{ServerType: db.ServerViewServerTypeBattle, SettingKey: constants.SettingKeyBattleServerPath, Label: "Battle Server"},
	}
}

func serverViewActions(serverType string) []ServerViewAction {
	switch serverType {
	case db.ServerViewServerTypeMain:
		return []ServerViewAction{{Label: "Map Info", Href: "/server-view/main/maps"}}
	case db.ServerViewServerTypeZone:
		return []ServerViewAction{
			{Label: "Loaded Maps", Href: "/server-view/zone/maps"},
			{Label: "Monster Spawns", Href: "/server-view/zone/spawns"},
			{Label: "Monster Drops", Href: "/server-view/zone/drops"},
			{Label: "Shops", Href: "/server-view/zone/shops"},
		}
	default:
		return []ServerViewAction{}
	}
}

func svrInfoRowsFromINI(serverType string, cfg *ini.File) []db.ServerViewSvrInfoRow {
	rows := []db.ServerViewSvrInfoRow{}
	rowOrder := 0
	for _, sectionName := range cfg.SectionStrings() {
		section := cfg.Section(sectionName)
		if section == nil {
			continue
		}

		displaySection := sectionName
		if strings.EqualFold(displaySection, ini.DefaultSection) {
			displaySection = "DEFAULT"
		}

		for _, key := range section.Keys() {
			keyName := strings.TrimSpace(key.Name())
			if keyName == "" {
				continue
			}

			values := key.ValueWithShadows()
			if len(values) == 0 {
				values = []string{key.String()}
			}

			for index, value := range values {
				rows = append(rows, db.ServerViewSvrInfoRow{
					ServerType: serverType,
					Section:    displaySection,
					Key:        keyName,
					Value:      strings.TrimSpace(value),
					ValueIndex: index,
					RowOrder:   rowOrder,
				})
				rowOrder++
			}
		}
	}

	return rows
}

func gameMasterRowsFromINI(cfg *ini.File) []db.ServerViewGameMasterRow {
	section := findINISection(cfg, "GM")
	if section == nil {
		section = findINISection(cfg, "gminfo")
	}
	if section == nil {
		return []db.ServerViewGameMasterRow{}
	}

	indices := map[int]bool{}
	gmCount := section.Key("GMCount").MustInt(-1)
	if gmCount > 0 {
		for index := 0; index < gmCount; index++ {
			indices[index] = true
		}
	}

	for _, key := range section.Keys() {
		name := key.Name()
		if strings.HasPrefix(strings.ToLower(name), "gmname_") {
			index, err := strconv.Atoi(name[len("GMName_"):])
			if err == nil {
				indices[index] = true
			}
		}
	}

	ordered := make([]int, 0, len(indices))
	for index := range indices {
		ordered = append(ordered, index)
	}
	sort.Ints(ordered)

	rows := make([]db.ServerViewGameMasterRow, 0, len(ordered))
	for _, index := range ordered {
		name := strings.TrimSpace(section.Key(fmt.Sprintf("GMName_%d", index)).String())
		if name == "" {
			name = strings.TrimSpace(section.Key(fmt.Sprintf("gm%d", index+1)).String())
		}
		if name == "" {
			continue
		}

		var level *int
		levelKey := section.Key(fmt.Sprintf("GMLevel_%d", index))
		if levelKey != nil && strings.TrimSpace(levelKey.String()) != "" {
			value := levelKey.MustInt(0)
			level = &value
		}

		rows = append(rows, db.ServerViewGameMasterRow{
			GMIndex: index,
			Name:    name,
			Level:   level,
		})
	}

	return rows
}

func groupSvrInfoRows(rows []db.ServerViewSvrInfoRow) map[string][]ServerViewInfoSection {
	grouped := make(map[string][]ServerViewInfoSection)
	sectionIndexes := make(map[string]map[string]int)

	for _, row := range rows {
		if row.Section == "MapZone" {
			continue
		}

		if _, ok := sectionIndexes[row.ServerType]; !ok {
			sectionIndexes[row.ServerType] = map[string]int{}
		}

		sections := grouped[row.ServerType]
		index, ok := sectionIndexes[row.ServerType][row.Section]
		if !ok {
			index = len(sections)
			sectionIndexes[row.ServerType][row.Section] = index
			sections = append(sections, ServerViewInfoSection{Name: row.Section, Rows: []ServerViewInfoRow{}})
		}

		sections[index].Rows = append(sections[index].Rows, ServerViewInfoRow{
			Key:        row.Key,
			Value:      row.Value,
			ValueIndex: row.ValueIndex,
		})
		grouped[row.ServerType] = sections
	}

	return grouped
}

func findINISection(cfg *ini.File, name string) *ini.Section {
	for _, sectionName := range cfg.SectionStrings() {
		if strings.EqualFold(sectionName, name) {
			return cfg.Section(sectionName)
		}
	}

	return nil
}

func parseNumericStem(name string) (int64, error) {
	stem := strings.TrimSuffix(name, filepath.Ext(name))
	if stem == "" {
		return 0, fmt.Errorf("file name %s does not have a numeric stem", name)
	}

	value, err := strconv.ParseInt(stem, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("file name %s does not have a numeric stem: %w", name, err)
	}

	return value, nil
}

func matchesIDOrName(query string, id int64, name string) bool {
	normalized := strings.ToLower(strings.TrimSpace(query))
	if normalized == "" {
		return true
	}

	return strings.Contains(strconv.FormatInt(id, 10), normalized) || strings.Contains(strings.ToLower(name), normalized)
}

func matchesNPCOrItemQuery(query string, npcID int64, npcName string, itemIDs []int64, itemNames []string) bool {
	normalized := strings.ToLower(strings.TrimSpace(query))
	if normalized == "" {
		return true
	}

	if strings.Contains(strconv.FormatInt(npcID, 10), normalized) || strings.Contains(strings.ToLower(npcName), normalized) {
		return true
	}

	for _, itemID := range itemIDs {
		if strings.Contains(strconv.FormatInt(itemID, 10), normalized) {
			return true
		}
	}

	for _, itemName := range itemNames {
		if strings.Contains(strings.ToLower(itemName), normalized) {
			return true
		}
	}

	return false
}

func displayNameID(id int64, name string) string {
	if name != "" {
		return fmt.Sprintf("%s (%d)", name, id)
	}

	return strconv.FormatInt(id, 10)
}

func itemBaseID(itemID int64) int64 {
	return itemID & serverViewItemIDMask
}

func stringPointerIfNotEmpty(value string) *string {
	if value == "" {
		return nil
	}

	return &value
}

func stringValueOrEmpty(value *string) string {
	if value == nil {
		return ""
	}

	return *value
}
