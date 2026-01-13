package db

import (
	"fmt"
	"time"

	"github.com/doug-martin/goqu/v9"
	"github.com/omnihance/omnihance-a3-agent/internal/logger"
)

type ServerProcess struct {
	ID            int64      `db:"id" json:"id"`
	Name          string     `db:"name" json:"name"`
	Path          string     `db:"path" json:"path"`
	Port          *int       `db:"port" json:"port"`
	SequenceOrder int        `db:"sequence_order" json:"sequence_order"`
	StartTime     *time.Time `db:"start_time" json:"start_time"`
	EndTime       *time.Time `db:"end_time" json:"end_time"`
	CreatedAt     time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt     *time.Time `db:"updated_at" json:"updated_at"`
}

type ReorderUpdate struct {
	ID            int64 `json:"id"`
	SequenceOrder int   `json:"sequence_order"`
}

func (s *sqliteInternalDB) GetServerProcesses() ([]ServerProcess, error) {
	processes := make([]ServerProcess, 0)
	err := s.goqu.From("server_processes").
		Prepared(true).
		Order(goqu.C("sequence_order").Asc()).
		ScanStructs(&processes)
	if err != nil {
		s.logger.Error(
			"failed to get server processes",
			logger.Field{Key: "error", Value: err},
		)
		return nil, fmt.Errorf("failed to get server processes: %w", err)
	}

	return processes, nil
}

func (s *sqliteInternalDB) GetServerProcess(id int64) (*ServerProcess, error) {
	var process ServerProcess
	found, err := s.goqu.From("server_processes").
		Prepared(true).
		Where(goqu.Ex{"id": id}).
		ScanStruct(&process)
	if err != nil {
		s.logger.Error(
			"failed to get server process",
			logger.Field{Key: "id", Value: id},
			logger.Field{Key: "error", Value: err},
		)
		return nil, fmt.Errorf("failed to get server process %d: %w", id, err)
	}

	if !found {
		return nil, fmt.Errorf("server process %d not found", id)
	}

	return &process, nil
}

func (s *sqliteInternalDB) GetServerProcessByPath(path string) (*ServerProcess, error) {
	var process ServerProcess
	found, err := s.goqu.From("server_processes").
		Prepared(true).
		Where(goqu.Ex{"path": path}).
		ScanStruct(&process)
	if err != nil {
		s.logger.Error(
			"failed to get server process by path",
			logger.Field{Key: "path", Value: path},
			logger.Field{Key: "error", Value: err},
		)
		return nil, fmt.Errorf("failed to get server process by path: %w", err)
	}

	if !found {
		return nil, nil
	}

	return &process, nil
}

func (s *sqliteInternalDB) CreateServerProcess(name, path string, port *int, sequenceOrder int) (*ServerProcess, error) {
	insertRecord := goqu.Record{
		"name":           name,
		"path":           path,
		"sequence_order": sequenceOrder,
	}

	if port != nil {
		insertRecord["port"] = *port
	}

	result, err := s.goqu.Insert("server_processes").
		Prepared(true).
		Rows(insertRecord).
		Executor().
		Exec()
	if err != nil {
		s.logger.Error(
			"failed to create server process",
			logger.Field{Key: "name", Value: name},
			logger.Field{Key: "path", Value: path},
			logger.Field{Key: "error", Value: err},
		)
		return nil, fmt.Errorf("failed to create server process: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get last insert id: %w", err)
	}

	return s.GetServerProcess(id)
}

func (s *sqliteInternalDB) UpdateServerProcess(id int64, name, path string, port *int) error {
	updateRecord := goqu.Record{
		"name":       name,
		"path":       path,
		"updated_at": goqu.L("CURRENT_TIMESTAMP"),
	}

	if port != nil {
		updateRecord["port"] = *port
	} else {
		updateRecord["port"] = nil
	}

	_, err := s.goqu.Update("server_processes").
		Prepared(true).
		Set(updateRecord).
		Where(goqu.Ex{"id": id}).
		Executor().
		Exec()
	if err != nil {
		s.logger.Error(
			"failed to update server process",
			logger.Field{Key: "id", Value: id},
			logger.Field{Key: "error", Value: err},
		)
		return fmt.Errorf("failed to update server process %d: %w", id, err)
	}

	return nil
}

func (s *sqliteInternalDB) DeleteServerProcess(id int64) error {
	_, err := s.goqu.Delete("server_processes").
		Prepared(true).
		Where(goqu.Ex{"id": id}).
		Executor().
		Exec()
	if err != nil {
		s.logger.Error(
			"failed to delete server process",
			logger.Field{Key: "id", Value: id},
			logger.Field{Key: "error", Value: err},
		)
		return fmt.Errorf("failed to delete server process %d: %w", id, err)
	}

	return nil
}

func (s *sqliteInternalDB) ReorderServerProcesses(updates []ReorderUpdate) error {
	tx, err := s.BeginTx()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	for _, update := range updates {
		_, err := tx.Update("server_processes").
			Prepared(true).
			Set(goqu.Record{
				"sequence_order": update.SequenceOrder,
				"updated_at":     goqu.L("CURRENT_TIMESTAMP"),
			}).
			Where(goqu.Ex{"id": update.ID}).
			Executor().
			Exec()
		if err != nil {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				s.logger.Error(
					"failed to rollback transaction",
					logger.Field{Key: "error", Value: rollbackErr},
				)
			}
			s.logger.Error(
				"failed to reorder server process",
				logger.Field{Key: "id", Value: update.ID},
				logger.Field{Key: "error", Value: err},
			)
			return fmt.Errorf("failed to reorder server process %d: %w", update.ID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

func (s *sqliteInternalDB) GetMaxSequenceOrder() (int, error) {
	var maxOrder *int
	found, err := s.goqu.From("server_processes").
		Prepared(true).
		Select(goqu.MAX("sequence_order").As("max_order")).
		ScanVal(&maxOrder)
	if err != nil {
		s.logger.Error(
			"failed to get max sequence order",
			logger.Field{Key: "error", Value: err},
		)
		return 0, fmt.Errorf("failed to get max sequence order: %w", err)
	}

	if !found || maxOrder == nil {
		return 0, nil
	}

	return *maxOrder, nil
}

func (s *sqliteInternalDB) UpdateProcessStartTime(id int64, startTime time.Time) error {
	_, err := s.goqu.Update("server_processes").
		Prepared(true).
		Set(goqu.Record{
			"start_time": startTime,
			"updated_at": goqu.L("CURRENT_TIMESTAMP"),
		}).
		Where(goqu.Ex{"id": id}).
		Executor().
		Exec()
	if err != nil {
		s.logger.Error(
			"failed to update process start time",
			logger.Field{Key: "id", Value: id},
			logger.Field{Key: "error", Value: err},
		)
		return fmt.Errorf("failed to update process start time %d: %w", id, err)
	}

	return nil
}

func (s *sqliteInternalDB) UpdateProcessEndTime(id int64, endTime time.Time) error {
	_, err := s.goqu.Update("server_processes").
		Prepared(true).
		Set(goqu.Record{
			"end_time":   endTime,
			"updated_at": goqu.L("CURRENT_TIMESTAMP"),
		}).
		Where(goqu.Ex{"id": id}).
		Executor().
		Exec()
	if err != nil {
		s.logger.Error(
			"failed to update process end time",
			logger.Field{Key: "id", Value: id},
			logger.Field{Key: "error", Value: err},
		)
		return fmt.Errorf("failed to update process end time %d: %w", id, err)
	}

	return nil
}
