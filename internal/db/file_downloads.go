package db

import (
	"fmt"
	"time"

	"github.com/doug-martin/goqu/v9"
	"github.com/doug-martin/goqu/v9/exp"
	"github.com/omnihance/omnihance-a3-agent/internal/logger"
)

const (
	FileDownloadSourceFileBrowser = "file_browser"
	FileDownloadSourceBackup      = "backup"
)

type FileDownloadLink struct {
	ID               int64      `db:"id" json:"id"`
	PublicID         string     `db:"public_id" json:"public_id"`
	UserID           int64      `db:"user_id" json:"user_id"`
	FileID           string     `db:"file_id" json:"file_id"`
	SourceType       string     `db:"source_type" json:"source_type"`
	BackupRunID      *int64     `db:"backup_run_id" json:"backup_run_id"`
	BackupFileID     *int64     `db:"backup_file_id" json:"backup_file_id"`
	OriginalPath     string     `db:"original_path" json:"original_path"`
	FileName         string     `db:"file_name" json:"file_name"`
	FileSize         int64      `db:"file_size" json:"file_size"`
	FileHash         string     `db:"file_hash" json:"file_hash"`
	FileModifiedAt   int64      `db:"file_modified_at" json:"file_modified_at"`
	ExpiresAt        time.Time  `db:"expires_at" json:"expires_at"`
	DownloadCount    int64      `db:"download_count" json:"download_count"`
	CreatedAt        time.Time  `db:"created_at" json:"created_at"`
	LastDownloadedAt *time.Time `db:"last_downloaded_at" json:"last_downloaded_at"`
}

type FileDownloadLinkPayload struct {
	PublicID       string
	UserID         int64
	FileID         string
	SourceType     string
	BackupRunID    *int64
	BackupFileID   *int64
	OriginalPath   string
	FileName       string
	FileSize       int64
	FileHash       string
	FileModifiedAt int64
	ExpiresAt      time.Time
}

func (s *sqliteInternalDB) GetReusableFileDownloadLink(payload FileDownloadLinkPayload, now time.Time) (*FileDownloadLink, error) {
	var link FileDownloadLink
	query := s.goqu.From("file_download_links").
		Prepared(true).
		Where(
			goqu.Ex{
				"user_id":          payload.UserID,
				"file_id":          payload.FileID,
				"source_type":      payload.SourceType,
				"original_path":    payload.OriginalPath,
				"file_size":        payload.FileSize,
				"file_hash":        payload.FileHash,
				"file_modified_at": payload.FileModifiedAt,
			},
			goqu.C("expires_at").Gt(now),
			nullableInt64Condition("backup_run_id", payload.BackupRunID),
			nullableInt64Condition("backup_file_id", payload.BackupFileID),
		).
		Order(goqu.C("created_at").Desc(), goqu.C("id").Desc()).
		Limit(1)

	found, err := query.ScanStruct(&link)
	if err != nil {
		s.logger.Error(
			"failed to get reusable file download link",
			logger.Field{Key: "file_id", Value: payload.FileID},
			logger.Field{Key: "user_id", Value: payload.UserID},
			logger.Field{Key: "error", Value: err},
		)
		return nil, fmt.Errorf("failed to get reusable file download link: %w", err)
	}

	if !found {
		return nil, nil
	}

	return &link, nil
}

func (s *sqliteInternalDB) CreateFileDownloadLink(payload FileDownloadLinkPayload) (*FileDownloadLink, error) {
	result, err := s.goqu.Insert("file_download_links").
		Prepared(true).
		Rows(fileDownloadLinkRecord(payload)).
		Executor().
		Exec()
	if err != nil {
		s.logger.Error(
			"failed to create file download link",
			logger.Field{Key: "file_id", Value: payload.FileID},
			logger.Field{Key: "user_id", Value: payload.UserID},
			logger.Field{Key: "error", Value: err},
		)
		return nil, fmt.Errorf("failed to create file download link: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get file download link id: %w", err)
	}

	return s.getFileDownloadLink(goqu.Ex{"id": id})
}

func (s *sqliteInternalDB) GetFileDownloadLinkByPublicID(publicID string) (*FileDownloadLink, error) {
	return s.getFileDownloadLink(goqu.Ex{"public_id": publicID})
}

func (s *sqliteInternalDB) RecordFileDownload(link *FileDownloadLink, userID int64, userAgent *string, ipAddress *string) error {
	tx, err := s.BeginTx()
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	_, err = tx.Insert("file_download_events").
		Prepared(true).
		Rows(goqu.Record{
			"link_id":        link.ID,
			"user_id":        userID,
			"file_id":        link.FileID,
			"source_type":    link.SourceType,
			"backup_run_id":  link.BackupRunID,
			"backup_file_id": link.BackupFileID,
			"original_path":  link.OriginalPath,
			"file_hash":      link.FileHash,
			"user_agent":     userAgent,
			"ip_address":     ipAddress,
		}).
		Executor().
		Exec()
	if err != nil {
		s.logger.Error(
			"failed to record file download event",
			logger.Field{Key: "link_id", Value: link.ID},
			logger.Field{Key: "user_id", Value: userID},
			logger.Field{Key: "error", Value: err},
		)
		return fmt.Errorf("failed to record file download event: %w", err)
	}

	result, err := tx.Update("file_download_links").
		Prepared(true).
		Set(goqu.Record{
			"download_count":     goqu.L("download_count + 1"),
			"last_downloaded_at": goqu.L("CURRENT_TIMESTAMP"),
		}).
		Where(goqu.Ex{"id": link.ID}).
		Executor().
		Exec()
	if err != nil {
		s.logger.Error(
			"failed to update file download count",
			logger.Field{Key: "link_id", Value: link.ID},
			logger.Field{Key: "error", Value: err},
		)
		return fmt.Errorf("failed to update file download count: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get file download count rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("file download link %d not found", link.ID)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit file download event: %w", err)
	}

	return nil
}

func (s *sqliteInternalDB) getFileDownloadLink(where goqu.Ex) (*FileDownloadLink, error) {
	var link FileDownloadLink
	found, err := s.goqu.From("file_download_links").
		Prepared(true).
		Where(where).
		ScanStruct(&link)
	if err != nil {
		s.logger.Error(
			"failed to get file download link",
			logger.Field{Key: "where", Value: where},
			logger.Field{Key: "error", Value: err},
		)
		return nil, fmt.Errorf("failed to get file download link: %w", err)
	}

	if !found {
		return nil, fmt.Errorf("file download link not found")
	}

	return &link, nil
}

func fileDownloadLinkRecord(payload FileDownloadLinkPayload) goqu.Record {
	return goqu.Record{
		"public_id":        payload.PublicID,
		"user_id":          payload.UserID,
		"file_id":          payload.FileID,
		"source_type":      payload.SourceType,
		"backup_run_id":    payload.BackupRunID,
		"backup_file_id":   payload.BackupFileID,
		"original_path":    payload.OriginalPath,
		"file_name":        payload.FileName,
		"file_size":        payload.FileSize,
		"file_hash":        payload.FileHash,
		"file_modified_at": payload.FileModifiedAt,
		"expires_at":       payload.ExpiresAt,
	}
}

func nullableInt64Condition(column string, value *int64) exp.Expression {
	if value == nil {
		return goqu.C(column).IsNull()
	}

	return goqu.Ex{column: *value}
}
