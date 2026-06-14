package db

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/doug-martin/goqu/v9"
	"github.com/omnihance/omnihance-a3-agent/internal/constants"
	"github.com/omnihance/omnihance-a3-agent/internal/logger"
)

type DirectoryDownloadArchive struct {
	ID                int64     `db:"id" json:"id"`
	NormalizedPath    string    `db:"normalized_path" json:"normalized_path"`
	SourceFingerprint string    `db:"source_fingerprint" json:"source_fingerprint"`
	JobID             int64     `db:"job_id" json:"job_id"`
	RunID             int64     `db:"run_id" json:"run_id"`
	FileID            int64     `db:"file_id" json:"file_id"`
	ArchivePath       string    `db:"archive_path" json:"archive_path"`
	ArchiveSize       int64     `db:"archive_size" json:"archive_size"`
	CreatedAt         time.Time `db:"created_at" json:"created_at"`
	UpdatedAt         time.Time `db:"updated_at" json:"updated_at"`
}

type DirectoryDownloadArchivePayload struct {
	NormalizedPath    string
	SourceFingerprint string
	JobID             int64
	RunID             int64
	FileID            int64
	ArchivePath       string
	ArchiveSize       int64
}

func (s *sqliteInternalDB) GetBackupJobByTagAndSourcePath(tag string, sourcePath string) (*BackupJob, error) {
	var job BackupJob
	found, err := s.goqu.From("backup_jobs").
		Prepared(true).
		Where(
			goqu.Ex{
				"tag":         tag,
				"source_path": sourcePath,
			},
			goqu.C("status").Neq(BackupJobStatusDeleted),
		).
		Order(goqu.C("created_at").Desc(), goqu.C("id").Desc()).
		Limit(1).
		ScanStruct(&job)
	if err != nil {
		s.logger.Error(
			"failed to get backup job by tag and source path",
			logger.Field{Key: "tag", Value: tag},
			logger.Field{Key: "source_path", Value: sourcePath},
			logger.Field{Key: "error", Value: err},
		)
		return nil, fmt.Errorf("failed to get backup job by tag and source path: %w", err)
	}

	if !found {
		return nil, fmt.Errorf("%w: backup job with tag %s and source path %s", ErrBackupNotFound, tag, sourcePath)
	}

	return &job, nil
}

func (s *sqliteInternalDB) GetRunningBackupRunForTag(tag string) (*BackupRun, *BackupJob, error) {
	var run BackupRun
	err := s.db.QueryRow(`
		SELECT
			r.id,
			r.job_id,
			r.trigger_type,
			r.status,
			r.previous_job_status,
			r.started_at,
			r.finished_at,
			r.cancel_requested_at,
			r.output,
			r.error_details,
			r.created_by,
			r.created_at,
			r.updated_at
		FROM backup_runs r
		INNER JOIN backup_jobs j ON j.id = r.job_id
		WHERE j.tag = ? AND j.status <> ? AND r.status = ?
		ORDER BY r.started_at DESC, r.id DESC
		LIMIT 1
	`, tag, BackupJobStatusDeleted, BackupRunStatusRunning).Scan(
		&run.ID,
		&run.JobID,
		&run.TriggerType,
		&run.Status,
		&run.PreviousJobStatus,
		&run.StartedAt,
		&run.FinishedAt,
		&run.CancelRequestedAt,
		&run.Output,
		&run.ErrorDetails,
		&run.CreatedBy,
		&run.CreatedAt,
		&run.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil, nil
		}

		s.logger.Error(
			"failed to get running backup run by tag",
			logger.Field{Key: "tag", Value: tag},
			logger.Field{Key: "error", Value: err},
		)
		return nil, nil, fmt.Errorf("failed to get running backup run by tag: %w", err)
	}

	job, err := s.GetBackupJob(run.JobID)
	if err != nil {
		return nil, nil, err
	}

	return &run, job, nil
}

func (s *sqliteInternalDB) GetDirectoryDownloadArchive(normalizedPath string, sourceFingerprint string) (*DirectoryDownloadArchive, error) {
	var archive DirectoryDownloadArchive
	found, err := s.goqu.From("directory_download_archives").
		Prepared(true).
		Where(goqu.Ex{
			"normalized_path":    normalizedPath,
			"source_fingerprint": sourceFingerprint,
		}).
		Order(goqu.C("updated_at").Desc(), goqu.C("id").Desc()).
		Limit(1).
		ScanStruct(&archive)
	if err != nil {
		s.logger.Error(
			"failed to get directory download archive",
			logger.Field{Key: "normalized_path", Value: normalizedPath},
			logger.Field{Key: "error", Value: err},
		)
		return nil, fmt.Errorf("failed to get directory download archive: %w", err)
	}

	if !found {
		return nil, constants.ErrNotFound
	}

	return &archive, nil
}

func (s *sqliteInternalDB) UpsertDirectoryDownloadArchive(payload DirectoryDownloadArchivePayload) (*DirectoryDownloadArchive, error) {
	_, err := s.db.Exec(`
		INSERT INTO directory_download_archives (
			normalized_path,
			source_fingerprint,
			job_id,
			run_id,
			file_id,
			archive_path,
			archive_size
		) VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(normalized_path, source_fingerprint) DO UPDATE SET
			job_id = excluded.job_id,
			run_id = excluded.run_id,
			file_id = excluded.file_id,
			archive_path = excluded.archive_path,
			archive_size = excluded.archive_size,
			updated_at = CURRENT_TIMESTAMP
	`,
		payload.NormalizedPath,
		payload.SourceFingerprint,
		payload.JobID,
		payload.RunID,
		payload.FileID,
		payload.ArchivePath,
		payload.ArchiveSize,
	)
	if err != nil {
		s.logger.Error(
			"failed to upsert directory download archive",
			logger.Field{Key: "normalized_path", Value: payload.NormalizedPath},
			logger.Field{Key: "run_id", Value: payload.RunID},
			logger.Field{Key: "error", Value: err},
		)
		return nil, fmt.Errorf("failed to upsert directory download archive: %w", err)
	}

	return s.GetDirectoryDownloadArchive(payload.NormalizedPath, payload.SourceFingerprint)
}
