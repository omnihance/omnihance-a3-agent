package db

import (
	"fmt"
	"time"

	"github.com/doug-martin/goqu/v9"
	"github.com/omnihance/omnihance-a3-agent/internal/logger"
)

const (
	ServerViewSyncStatusRunning   = "running"
	ServerViewSyncStatusSucceeded = "succeeded"
	ServerViewSyncStatusFailed    = "failed"
)

const (
	ServerViewServerTypeMain    = "main"
	ServerViewServerTypeAccount = "account"
	ServerViewServerTypeZone    = "zone"
	ServerViewServerTypeBattle  = "battle"
)

type ServerViewSvrInfoRow struct {
	ID         int64     `db:"id" json:"id"`
	ServerType string    `db:"server_type" json:"server_type"`
	Section    string    `db:"section" json:"section"`
	Key        string    `db:"key" json:"key"`
	Value      string    `db:"value" json:"value"`
	ValueIndex int       `db:"value_index" json:"value_index"`
	RowOrder   int       `db:"row_order" json:"row_order"`
	UpdatedAt  time.Time `db:"updated_at" json:"updated_at"`
}

type ServerViewMapZone struct {
	ID         int64     `db:"id" json:"id"`
	ServerType string    `db:"server_type" json:"server_type"`
	MapID      int64     `db:"map_id" json:"map_id"`
	ZoneID     int64     `db:"zone_id" json:"zone_id"`
	UpdatedAt  time.Time `db:"updated_at" json:"updated_at"`
}

type ServerViewSpawnRow struct {
	ID          int64     `db:"id" json:"id"`
	MapID       int64     `db:"map_id" json:"map_id"`
	RowIndex    int       `db:"row_index" json:"row_index"`
	NPCID       int64     `db:"npc_id" json:"npc_id"`
	X           int       `db:"x" json:"x"`
	Y           int       `db:"y" json:"y"`
	Unknown1    int       `db:"unknown1" json:"unknown1"`
	Orientation int       `db:"orientation" json:"orientation"`
	SpawnStep   int       `db:"spawn_step" json:"spawn_step"`
	UpdatedAt   time.Time `db:"updated_at" json:"updated_at"`
}

type ServerViewDropRow struct {
	ID        int64     `db:"id" json:"id"`
	NPCID     int64     `db:"npc_id" json:"npc_id"`
	RowIndex  int       `db:"row_index" json:"row_index"`
	ItemID    int64     `db:"item_id" json:"item_id"`
	DropRate  int       `db:"drop_rate" json:"drop_rate"`
	GroupCode int       `db:"group_code" json:"group_code"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

type ServerViewShopRow struct {
	ID         int64     `db:"id" json:"id"`
	NPCID      int64     `db:"npc_id" json:"npc_id"`
	LineNumber int       `db:"line_number" json:"line_number"`
	ItemID     int64     `db:"item_id" json:"item_id"`
	UpdatedAt  time.Time `db:"updated_at" json:"updated_at"`
}

type ServerViewGameMasterRow struct {
	GMIndex   int        `db:"gm_index" json:"gm_index"`
	Name      string     `db:"name" json:"name"`
	Level     *int       `db:"level" json:"level"`
	UpdatedAt *time.Time `db:"updated_at" json:"updated_at"`
}

type ServerViewSyncRun struct {
	ID           int64      `db:"id" json:"id"`
	Status       string     `db:"status" json:"status"`
	StartedAt    time.Time  `db:"started_at" json:"started_at"`
	FinishedAt   *time.Time `db:"finished_at" json:"finished_at"`
	WarningCount int        `db:"warning_count" json:"warning_count"`
	ErrorDetails *string    `db:"error_details" json:"error_details"`
	CreatedBy    *int64     `db:"created_by" json:"created_by"`
	CreatedAt    time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt    *time.Time `db:"updated_at" json:"updated_at"`
}

type ServerViewSyncWarning struct {
	ID        int64     `db:"id" json:"id"`
	RunID     int64     `db:"run_id" json:"run_id"`
	Source    string    `db:"source" json:"source"`
	Message   string    `db:"message" json:"message"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}

func (s *sqliteInternalDB) CreateServerViewSyncRun(userID *int64) (*ServerViewSyncRun, error) {
	record := goqu.Record{"status": ServerViewSyncStatusRunning}
	if userID != nil {
		record["created_by"] = *userID
	}

	result, err := s.goqu.Insert("server_view_sync_runs").
		Prepared(true).
		Rows(record).
		Executor().
		Exec()
	if err != nil {
		s.logger.Error("failed to create server view sync run", logger.Field{Key: "error", Value: err})
		return nil, fmt.Errorf("failed to create server view sync run: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get server view sync run id: %w", err)
	}

	return s.getServerViewSyncRun(id)
}

func (s *sqliteInternalDB) FinishServerViewSyncRun(runID int64, status string, warningCount int, errorDetails *string) error {
	result, err := s.goqu.Update("server_view_sync_runs").
		Prepared(true).
		Set(goqu.Record{
			"status":        status,
			"finished_at":   goqu.L("CURRENT_TIMESTAMP"),
			"warning_count": warningCount,
			"error_details": errorDetails,
			"updated_at":    goqu.L("CURRENT_TIMESTAMP"),
		}).
		Where(goqu.Ex{"id": runID}).
		Executor().
		Exec()
	if err != nil {
		s.logger.Error("failed to finish server view sync run", logger.Field{Key: "run_id", Value: runID}, logger.Field{Key: "error", Value: err})
		return fmt.Errorf("failed to finish server view sync run: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get server view sync rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("server view sync run %d not found", runID)
	}

	return nil
}

func (s *sqliteInternalDB) GetLatestServerViewSyncRun() (*ServerViewSyncRun, error) {
	var run ServerViewSyncRun
	found, err := s.goqu.From("server_view_sync_runs").
		Prepared(true).
		Order(goqu.C("started_at").Desc(), goqu.C("id").Desc()).
		Limit(1).
		ScanStruct(&run)
	if err != nil {
		s.logger.Error("failed to get latest server view sync run", logger.Field{Key: "error", Value: err})
		return nil, fmt.Errorf("failed to get latest server view sync run: %w", err)
	}

	if !found {
		return nil, nil
	}

	return &run, nil
}

func (s *sqliteInternalDB) GetRunningServerViewSyncRun() (*ServerViewSyncRun, error) {
	var run ServerViewSyncRun
	found, err := s.goqu.From("server_view_sync_runs").
		Prepared(true).
		Where(goqu.Ex{"status": ServerViewSyncStatusRunning}).
		Order(goqu.C("started_at").Desc(), goqu.C("id").Desc()).
		Limit(1).
		ScanStruct(&run)
	if err != nil {
		s.logger.Error("failed to get running server view sync run", logger.Field{Key: "error", Value: err})
		return nil, fmt.Errorf("failed to get running server view sync run: %w", err)
	}

	if !found {
		return nil, nil
	}

	return &run, nil
}

func (s *sqliteInternalDB) MarkOrphanedServerViewSyncRunsFailed() error {
	errorDetails := "Sync was marked failed because the service restarted before it finished."
	_, err := s.goqu.Update("server_view_sync_runs").
		Prepared(true).
		Set(goqu.Record{
			"status":        ServerViewSyncStatusFailed,
			"finished_at":   goqu.L("CURRENT_TIMESTAMP"),
			"updated_at":    goqu.L("CURRENT_TIMESTAMP"),
			"error_details": errorDetails,
		}).
		Where(goqu.Ex{"status": ServerViewSyncStatusRunning}).
		Executor().
		Exec()
	if err != nil {
		s.logger.Error("failed to mark orphaned server view sync runs failed", logger.Field{Key: "error", Value: err})
		return fmt.Errorf("failed to mark orphaned server view sync runs failed: %w", err)
	}

	return nil
}

func (s *sqliteInternalDB) AddServerViewSyncWarning(runID int64, source string, message string) error {
	_, err := s.goqu.Insert("server_view_sync_warnings").
		Prepared(true).
		Rows(goqu.Record{
			"run_id":  runID,
			"source":  source,
			"message": message,
		}).
		Executor().
		Exec()
	if err != nil {
		s.logger.Error("failed to add server view sync warning", logger.Field{Key: "run_id", Value: runID}, logger.Field{Key: "error", Value: err})
		return fmt.Errorf("failed to add server view sync warning: %w", err)
	}

	return nil
}

func (s *sqliteInternalDB) GetServerViewSyncWarnings(runID int64) ([]ServerViewSyncWarning, error) {
	warnings := make([]ServerViewSyncWarning, 0)
	err := s.goqu.From("server_view_sync_warnings").
		Prepared(true).
		Where(goqu.Ex{"run_id": runID}).
		Order(goqu.C("id").Asc()).
		ScanStructs(&warnings)
	if err != nil {
		s.logger.Error("failed to get server view sync warnings", logger.Field{Key: "run_id", Value: runID}, logger.Field{Key: "error", Value: err})
		return nil, fmt.Errorf("failed to get server view sync warnings: %w", err)
	}

	return warnings, nil
}

func (s *sqliteInternalDB) ReplaceServerViewSvrInfoRows(serverType string, rows []ServerViewSvrInfoRow) error {
	return s.replaceRows("server_view_svr_info_rows", goqu.Ex{"server_type": serverType}, serverViewSvrInfoRecords(rows))
}

func (s *sqliteInternalDB) GetServerViewSvrInfoRows() ([]ServerViewSvrInfoRow, error) {
	rows := make([]ServerViewSvrInfoRow, 0)
	err := s.goqu.From("server_view_svr_info_rows").
		Prepared(true).
		Order(goqu.C("server_type").Asc(), goqu.C("row_order").Asc(), goqu.C("id").Asc()).
		ScanStructs(&rows)
	if err != nil {
		s.logger.Error("failed to get server view svr info rows", logger.Field{Key: "error", Value: err})
		return nil, fmt.Errorf("failed to get server view svr info rows: %w", err)
	}

	return rows, nil
}

func (s *sqliteInternalDB) ReplaceServerViewMapZones(serverType string, rows []ServerViewMapZone) error {
	return s.replaceRows("server_view_map_zones", goqu.Ex{"server_type": serverType}, serverViewMapZoneRecords(rows))
}

func (s *sqliteInternalDB) GetServerViewMapZones(serverType string) ([]ServerViewMapZone, error) {
	rows := make([]ServerViewMapZone, 0)
	err := s.goqu.From("server_view_map_zones").
		Prepared(true).
		Where(goqu.Ex{"server_type": serverType}).
		Order(goqu.C("map_id").Asc()).
		ScanStructs(&rows)
	if err != nil {
		s.logger.Error("failed to get server view map zones", logger.Field{Key: "server_type", Value: serverType}, logger.Field{Key: "error", Value: err})
		return nil, fmt.Errorf("failed to get server view map zones: %w", err)
	}

	return rows, nil
}

func (s *sqliteInternalDB) ReplaceServerViewSpawnRowsForMap(mapID int64, rows []ServerViewSpawnRow) error {
	return s.replaceRows("server_view_spawn_rows", goqu.Ex{"map_id": mapID}, serverViewSpawnRecords(rows))
}

func (s *sqliteInternalDB) GetServerViewSpawnRows() ([]ServerViewSpawnRow, error) {
	rows := make([]ServerViewSpawnRow, 0)
	err := s.goqu.From("server_view_spawn_rows").
		Prepared(true).
		Order(goqu.C("map_id").Asc(), goqu.C("row_index").Asc()).
		ScanStructs(&rows)
	if err != nil {
		s.logger.Error("failed to get server view spawn rows", logger.Field{Key: "error", Value: err})
		return nil, fmt.Errorf("failed to get server view spawn rows: %w", err)
	}

	return rows, nil
}

func (s *sqliteInternalDB) ReplaceServerViewDropRowsForNPC(npcID int64, rows []ServerViewDropRow) error {
	return s.replaceRows("server_view_drop_rows", goqu.Ex{"npc_id": npcID}, serverViewDropRecords(rows))
}

func (s *sqliteInternalDB) GetServerViewDropRows() ([]ServerViewDropRow, error) {
	rows := make([]ServerViewDropRow, 0)
	err := s.goqu.From("server_view_drop_rows").
		Prepared(true).
		Order(goqu.C("npc_id").Asc(), goqu.C("row_index").Asc()).
		ScanStructs(&rows)
	if err != nil {
		s.logger.Error("failed to get server view drop rows", logger.Field{Key: "error", Value: err})
		return nil, fmt.Errorf("failed to get server view drop rows: %w", err)
	}

	return rows, nil
}

func (s *sqliteInternalDB) ReplaceServerViewShopRowsForNPC(npcID int64, rows []ServerViewShopRow) error {
	return s.replaceRows("server_view_shop_rows", goqu.Ex{"npc_id": npcID}, serverViewShopRecords(rows))
}

func (s *sqliteInternalDB) GetServerViewShopRows() ([]ServerViewShopRow, error) {
	rows := make([]ServerViewShopRow, 0)
	err := s.goqu.From("server_view_shop_rows").
		Prepared(true).
		Order(goqu.C("npc_id").Asc(), goqu.C("line_number").Asc()).
		ScanStructs(&rows)
	if err != nil {
		s.logger.Error("failed to get server view shop rows", logger.Field{Key: "error", Value: err})
		return nil, fmt.Errorf("failed to get server view shop rows: %w", err)
	}

	return rows, nil
}

func (s *sqliteInternalDB) ReplaceServerViewGameMasterRows(rows []ServerViewGameMasterRow) error {
	return s.replaceRows("server_view_game_master_rows", nil, serverViewGameMasterRecords(rows))
}

func (s *sqliteInternalDB) GetServerViewGameMasterRows() ([]ServerViewGameMasterRow, error) {
	rows := make([]ServerViewGameMasterRow, 0)
	err := s.goqu.From("server_view_game_master_rows").
		Prepared(true).
		Order(goqu.C("gm_index").Asc()).
		ScanStructs(&rows)
	if err != nil {
		s.logger.Error("failed to get server view game master rows", logger.Field{Key: "error", Value: err})
		return nil, fmt.Errorf("failed to get server view game master rows: %w", err)
	}

	return rows, nil
}

func (s *sqliteInternalDB) getServerViewSyncRun(id int64) (*ServerViewSyncRun, error) {
	var run ServerViewSyncRun
	found, err := s.goqu.From("server_view_sync_runs").
		Prepared(true).
		Where(goqu.Ex{"id": id}).
		ScanStruct(&run)
	if err != nil {
		s.logger.Error("failed to get server view sync run", logger.Field{Key: "id", Value: id}, logger.Field{Key: "error", Value: err})
		return nil, fmt.Errorf("failed to get server view sync run: %w", err)
	}

	if !found {
		return nil, fmt.Errorf("server view sync run %d not found", id)
	}

	return &run, nil
}

func (s *sqliteInternalDB) replaceRows(table string, where goqu.Ex, records []goqu.Record) error {
	tx, err := s.BeginTx()
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	deleteQuery := tx.Delete(table).Prepared(true)
	if where != nil {
		deleteQuery = deleteQuery.Where(where)
	}

	if _, err := deleteQuery.Executor().Exec(); err != nil {
		s.logger.Error("failed to delete server view rows", logger.Field{Key: "table", Value: table}, logger.Field{Key: "error", Value: err})
		return fmt.Errorf("failed to delete server view rows from %s: %w", table, err)
	}

	if len(records) > 0 {
		if _, err := tx.Insert(table).Prepared(true).Rows(records).Executor().Exec(); err != nil {
			s.logger.Error("failed to insert server view rows", logger.Field{Key: "table", Value: table}, logger.Field{Key: "count", Value: len(records)}, logger.Field{Key: "error", Value: err})
			return fmt.Errorf("failed to insert server view rows into %s: %w", table, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit server view rows for %s: %w", table, err)
	}

	return nil
}

func serverViewSvrInfoRecords(rows []ServerViewSvrInfoRow) []goqu.Record {
	records := make([]goqu.Record, 0, len(rows))
	for _, row := range rows {
		records = append(records, goqu.Record{
			"server_type": row.ServerType,
			"section":     row.Section,
			"key":         row.Key,
			"value":       row.Value,
			"value_index": row.ValueIndex,
			"row_order":   row.RowOrder,
		})
	}

	return records
}

func serverViewMapZoneRecords(rows []ServerViewMapZone) []goqu.Record {
	records := make([]goqu.Record, 0, len(rows))
	for _, row := range rows {
		records = append(records, goqu.Record{
			"server_type": row.ServerType,
			"map_id":      row.MapID,
			"zone_id":     row.ZoneID,
		})
	}

	return records
}

func serverViewSpawnRecords(rows []ServerViewSpawnRow) []goqu.Record {
	records := make([]goqu.Record, 0, len(rows))
	for _, row := range rows {
		records = append(records, goqu.Record{
			"map_id":      row.MapID,
			"row_index":   row.RowIndex,
			"npc_id":      row.NPCID,
			"x":           row.X,
			"y":           row.Y,
			"unknown1":    row.Unknown1,
			"orientation": row.Orientation,
			"spawn_step":  row.SpawnStep,
		})
	}

	return records
}

func serverViewDropRecords(rows []ServerViewDropRow) []goqu.Record {
	records := make([]goqu.Record, 0, len(rows))
	for _, row := range rows {
		records = append(records, goqu.Record{
			"npc_id":     row.NPCID,
			"row_index":  row.RowIndex,
			"item_id":    row.ItemID,
			"drop_rate":  row.DropRate,
			"group_code": row.GroupCode,
		})
	}

	return records
}

func serverViewShopRecords(rows []ServerViewShopRow) []goqu.Record {
	records := make([]goqu.Record, 0, len(rows))
	for _, row := range rows {
		records = append(records, goqu.Record{
			"npc_id":      row.NPCID,
			"line_number": row.LineNumber,
			"item_id":     row.ItemID,
		})
	}

	return records
}

func serverViewGameMasterRecords(rows []ServerViewGameMasterRow) []goqu.Record {
	records := make([]goqu.Record, 0, len(rows))
	for _, row := range rows {
		record := goqu.Record{
			"gm_index": row.GMIndex,
			"name":     row.Name,
		}
		if row.Level != nil {
			record["level"] = *row.Level
		}

		records = append(records, record)
	}

	return records
}
