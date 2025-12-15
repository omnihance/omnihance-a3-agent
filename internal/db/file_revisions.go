package db

import (
	"fmt"

	"github.com/doug-martin/goqu/v9"
	"github.com/omnihance/omnihance-a3-agent/internal/logger"
)

type FileRevision struct {
	ID           int64  `db:"id" json:"id"`
	FileID       string `db:"file_id" json:"file_id"`
	OriginalPath string `db:"original_path" json:"original_path"`
	RevisionPath string `db:"revision_path" json:"revision_path"`
	PreviousHash string `db:"previous_hash" json:"previous_hash"`
	CurrentHash  string `db:"current_hash" json:"current_hash"`
	CreatedBy    int64  `db:"created_by" json:"created_by"`
	CreatedAt    int64  `db:"created_at" json:"created_at"`
	UpdatedBy    *int64 `db:"updated_by" json:"updated_by"`
	UpdatedAt    *int64 `db:"updated_at" json:"updated_at"`
	Status       string `db:"status" json:"status"`
}

func (s *sqliteInternalDB) CreateFileRevision(tx *goqu.TxDatabase, fileID, originalPath, revisionPath string, previousHash, currentHash string, createdBy int64) (int64, error) {
	result, err := tx.Insert("file_revisions").
		Prepared(true).
		Rows(goqu.Record{
			"file_id":       fileID,
			"original_path": originalPath,
			"revision_path": revisionPath,
			"previous_hash": previousHash,
			"current_hash":  currentHash,
			"created_by":    createdBy,
			"created_at":    goqu.L("strftime('%s', 'now')"),
			"status":        "draft",
		}).
		Executor().
		Exec()
	if err != nil {
		s.logger.Error(
			"failed to create file revision",
			logger.Field{Key: "file_id", Value: fileID},
			logger.Field{Key: "created_by", Value: createdBy},
			logger.Field{Key: "error", Value: err},
		)
		return 0, fmt.Errorf("failed to create file revision: %w", err)
	}

	revisionID, err := result.LastInsertId()
	if err != nil {
		s.logger.Error(
			"failed to get last insert id",
			logger.Field{Key: "error", Value: err},
		)
		return 0, fmt.Errorf("failed to get last insert id: %w", err)
	}

	return revisionID, nil
}

func (s *sqliteInternalDB) UpdateFileRevisionStatus(tx *goqu.TxDatabase, revisionID int64, status string, updatedBy int64) error {
	updateRecord := goqu.Record{
		"status":     status,
		"updated_at": goqu.L("strftime('%s', 'now')"),
		"updated_by": updatedBy,
	}

	_, err := tx.Update("file_revisions").
		Prepared(true).
		Set(updateRecord).
		Where(goqu.Ex{"id": revisionID}).
		Executor().
		Exec()
	if err != nil {
		s.logger.Error(
			"failed to update file revision status",
			logger.Field{Key: "revision_id", Value: revisionID},
			logger.Field{Key: "status", Value: status},
			logger.Field{Key: "updated_by", Value: updatedBy},
			logger.Field{Key: "error", Value: err},
		)
		return fmt.Errorf("failed to update file revision status: %w", err)
	}

	return nil
}

func (s *sqliteInternalDB) UpdateFileRevisionPath(tx *goqu.TxDatabase, revisionID int64, revisionPath string, updatedBy int64) error {
	updateRecord := goqu.Record{
		"revision_path": revisionPath,
		"updated_at":    goqu.L("strftime('%s', 'now')"),
		"updated_by":    updatedBy,
	}

	_, err := tx.Update("file_revisions").
		Prepared(true).
		Set(updateRecord).
		Where(goqu.Ex{"id": revisionID}).
		Executor().
		Exec()
	if err != nil {
		s.logger.Error(
			"failed to update file revision path",
			logger.Field{Key: "revision_id", Value: revisionID},
			logger.Field{Key: "updated_by", Value: updatedBy},
			logger.Field{Key: "error", Value: err},
		)
		return fmt.Errorf("failed to update file revision path: %w", err)
	}

	return nil
}

func (s *sqliteInternalDB) GetFileRevision(revisionID int64) (*FileRevision, error) {
	var revision FileRevision
	found, err := s.goqu.From("file_revisions").
		Prepared(true).
		Where(goqu.Ex{"id": revisionID}).
		ScanStruct(&revision)
	if err != nil {
		s.logger.Error(
			"failed to get file revision",
			logger.Field{Key: "revision_id", Value: revisionID},
			logger.Field{Key: "error", Value: err},
		)
		return nil, fmt.Errorf("failed to get file revision: %w", err)
	}

	if !found {
		return nil, fmt.Errorf("file revision %d not found", revisionID)
	}

	return &revision, nil
}

func (s *sqliteInternalDB) GetLastCompletedFileRevision(fileID string) (*FileRevision, error) {
	var revision FileRevision
	found, err := s.goqu.From("file_revisions").
		Prepared(true).
		Where(goqu.Ex{"file_id": fileID, "status": "completed"}).
		Order(goqu.I("created_at").Desc()).
		Limit(1).
		ScanStruct(&revision)
	if err != nil {
		s.logger.Error(
			"failed to get last completed file revision",
			logger.Field{Key: "file_id", Value: fileID},
			logger.Field{Key: "error", Value: err},
		)
		return nil, fmt.Errorf("failed to get last completed file revision: %w", err)
	}

	if !found {
		return nil, nil
	}

	return &revision, nil
}

type RevisionSummary struct {
	Count          int64  `json:"count"`
	LastRevisionAt *int64 `json:"last_revision_at,omitempty"`
}

func (s *sqliteInternalDB) GetCompletedRevisionCount(fileID string) (int64, error) {
	var count int64
	_, err := s.goqu.From("file_revisions").
		Prepared(true).
		Select(goqu.COUNT("*")).
		Where(goqu.Ex{"file_id": fileID, "status": "completed"}).
		ScanVal(&count)
	if err != nil {
		s.logger.Error(
			"failed to get completed revision count",
			logger.Field{Key: "file_id", Value: fileID},
			logger.Field{Key: "error", Value: err},
		)
		return 0, fmt.Errorf("failed to get completed revision count: %w", err)
	}

	return count, nil
}

func (s *sqliteInternalDB) GetRevisionSummary(fileID string) (*RevisionSummary, error) {
	var summary RevisionSummary

	_, err := s.goqu.From("file_revisions").
		Prepared(true).
		Select(goqu.COUNT("*")).
		Where(goqu.Ex{"file_id": fileID, "status": "completed"}).
		ScanVal(&summary.Count)
	if err != nil {
		s.logger.Error(
			"failed to get revision summary count",
			logger.Field{Key: "file_id", Value: fileID},
			logger.Field{Key: "error", Value: err},
		)
		return nil, fmt.Errorf("failed to get revision summary count: %w", err)
	}

	if summary.Count == 0 {
		return &RevisionSummary{Count: 0, LastRevisionAt: nil}, nil
	}

	var lastRevisionAt int64
	found, err := s.goqu.From("file_revisions").
		Prepared(true).
		Select(goqu.I("created_at")).
		Where(goqu.Ex{"file_id": fileID, "status": "completed"}).
		Order(goqu.I("created_at").Desc()).
		Limit(1).
		ScanVal(&lastRevisionAt)
	if err != nil {
		s.logger.Error(
			"failed to get last revision time",
			logger.Field{Key: "file_id", Value: fileID},
			logger.Field{Key: "error", Value: err},
		)
		return nil, fmt.Errorf("failed to get last revision time: %w", err)
	}

	if found {
		summary.LastRevisionAt = &lastRevisionAt
	}

	return &summary, nil
}
