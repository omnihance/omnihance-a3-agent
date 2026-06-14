package db

import (
	"errors"
	"fmt"
	"time"

	"github.com/doug-martin/goqu/v9"
	"github.com/omnihance/omnihance-a3-agent/internal/logger"
)

var ErrBackupNotFound = errors.New("backup item not found")

const (
	BackupJobTypeFile      = "file"
	BackupJobTypeSQLServer = "sql_server"

	BackupJobTagDirectoryDownload = "directory_download"

	BackupJobStatusActive   = "active"
	BackupJobStatusInactive = "inactive"
	BackupJobStatusRunning  = "running"
	BackupJobStatusDeleted  = "deleted"

	BackupRunStatusRunning   = "running"
	BackupRunStatusSucceeded = "succeeded"
	BackupRunStatusFailed    = "failed"
	BackupRunStatusCancelled = "cancelled"
	BackupRunStatusSkipped   = "skipped"

	BackupRunTriggerManual            = "manual"
	BackupRunTriggerCron              = "cron"
	BackupRunTriggerDirectoryDownload = "directory_download"
)

type BackupJob struct {
	ID                   int64      `db:"id" json:"id"`
	JobType              string     `db:"job_type" json:"job_type"`
	Tag                  *string    `db:"tag" json:"tag"`
	Name                 string     `db:"name" json:"name"`
	Status               string     `db:"status" json:"status"`
	CronExpression       *string    `db:"cron_expression" json:"cron_expression"`
	DestinationDirectory string     `db:"destination_directory" json:"destination_directory"`
	ArchivePassword      *string    `db:"archive_password" json:"archive_password"`
	SourcePath           *string    `db:"source_path" json:"source_path"`
	SQLHost              *string    `db:"sql_host" json:"sql_host"`
	SQLPort              *int       `db:"sql_port" json:"sql_port"`
	SQLUsername          *string    `db:"sql_username" json:"sql_username"`
	SQLPassword          *string    `db:"sql_password" json:"sql_password"`
	SQLDatabaseNames     *string    `db:"sql_database_names" json:"sql_database_names"`
	LastRunAt            *time.Time `db:"last_run_at" json:"last_run_at"`
	CreatedBy            *int64     `db:"created_by" json:"created_by"`
	CreatedAt            time.Time  `db:"created_at" json:"created_at"`
	UpdatedBy            *int64     `db:"updated_by" json:"updated_by"`
	UpdatedAt            *time.Time `db:"updated_at" json:"updated_at"`
	DeletedBy            *int64     `db:"deleted_by" json:"deleted_by"`
	DeletedAt            *time.Time `db:"deleted_at" json:"deleted_at"`
}

type BackupRun struct {
	ID                int64      `db:"id" json:"id"`
	JobID             int64      `db:"job_id" json:"job_id"`
	TriggerType       string     `db:"trigger_type" json:"trigger_type"`
	Status            string     `db:"status" json:"status"`
	PreviousJobStatus string     `db:"previous_job_status" json:"previous_job_status"`
	StartedAt         time.Time  `db:"started_at" json:"started_at"`
	FinishedAt        *time.Time `db:"finished_at" json:"finished_at"`
	CancelRequestedAt *time.Time `db:"cancel_requested_at" json:"cancel_requested_at"`
	Output            *string    `db:"output" json:"output"`
	ErrorDetails      *string    `db:"error_details" json:"error_details"`
	CreatedBy         *int64     `db:"created_by" json:"created_by"`
	CreatedAt         time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt         *time.Time `db:"updated_at" json:"updated_at"`
}

type BackupRunFile struct {
	ID        int64     `db:"id" json:"id"`
	RunID     int64     `db:"run_id" json:"run_id"`
	ItemName  string    `db:"item_name" json:"item_name"`
	FilePath  string    `db:"file_path" json:"file_path"`
	FileSize  int64     `db:"file_size" json:"file_size"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}

type BackupJobPayload struct {
	JobType              string
	Tag                  *string
	Name                 string
	Status               string
	CronExpression       *string
	DestinationDirectory string
	ArchivePassword      *string
	SourcePath           *string
	SQLHost              *string
	SQLPort              *int
	SQLUsername          *string
	SQLPassword          *string
	SQLDatabaseNames     *string
}

func (s *sqliteInternalDB) GetBackupJobs() ([]BackupJob, error) {
	jobs := make([]BackupJob, 0)
	err := s.goqu.From("backup_jobs").
		Prepared(true).
		Where(goqu.C("status").Neq(BackupJobStatusDeleted)).
		Order(goqu.C("created_at").Desc()).
		ScanStructs(&jobs)
	if err != nil {
		s.logger.Error("failed to get backup jobs", logger.Field{Key: "error", Value: err})
		return nil, fmt.Errorf("failed to get backup jobs: %w", err)
	}

	return jobs, nil
}

func (s *sqliteInternalDB) GetSchedulableBackupJobs() ([]BackupJob, error) {
	jobs := make([]BackupJob, 0)
	err := s.goqu.From("backup_jobs").
		Prepared(true).
		Where(
			goqu.Ex{
				"status": BackupJobStatusActive,
			},
			goqu.C("cron_expression").IsNotNull(),
			goqu.C("cron_expression").Neq(""),
		).
		Order(goqu.C("created_at").Asc()).
		ScanStructs(&jobs)
	if err != nil {
		s.logger.Error("failed to get schedulable backup jobs", logger.Field{Key: "error", Value: err})
		return nil, fmt.Errorf("failed to get schedulable backup jobs: %w", err)
	}

	return jobs, nil
}

func (s *sqliteInternalDB) GetBackupJob(id int64) (*BackupJob, error) {
	var job BackupJob
	found, err := s.goqu.From("backup_jobs").
		Prepared(true).
		Where(goqu.Ex{"id": id}).
		ScanStruct(&job)
	if err != nil {
		s.logger.Error("failed to get backup job", logger.Field{Key: "id", Value: id}, logger.Field{Key: "error", Value: err})
		return nil, fmt.Errorf("failed to get backup job %d: %w", id, err)
	}

	if !found {
		return nil, fmt.Errorf("%w: backup job %d", ErrBackupNotFound, id)
	}

	return &job, nil
}

func (s *sqliteInternalDB) CreateBackupJob(payload BackupJobPayload, userID *int64) (*BackupJob, error) {
	record := backupJobRecord(payload)
	if userID != nil {
		record["created_by"] = *userID
	}

	result, err := s.goqu.Insert("backup_jobs").
		Prepared(true).
		Rows(record).
		Executor().
		Exec()
	if err != nil {
		s.logger.Error("failed to create backup job", logger.Field{Key: "name", Value: payload.Name}, logger.Field{Key: "error", Value: err})
		return nil, fmt.Errorf("failed to create backup job: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get backup job id: %w", err)
	}

	return s.GetBackupJob(id)
}

func (s *sqliteInternalDB) UpdateBackupJob(id int64, payload BackupJobPayload, userID *int64) (*BackupJob, error) {
	record := backupJobRecord(payload)
	record["updated_at"] = goqu.L("CURRENT_TIMESTAMP")
	if userID != nil {
		record["updated_by"] = *userID
	}

	result, err := s.goqu.Update("backup_jobs").
		Prepared(true).
		Set(record).
		Where(goqu.Ex{"id": id}).
		Executor().
		Exec()
	if err != nil {
		s.logger.Error("failed to update backup job", logger.Field{Key: "id", Value: id}, logger.Field{Key: "error", Value: err})
		return nil, fmt.Errorf("failed to update backup job %d: %w", id, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("failed to get rows affected for backup job %d: %w", id, err)
	}

	if rowsAffected == 0 {
		return nil, fmt.Errorf("%w: backup job %d", ErrBackupNotFound, id)
	}

	return s.GetBackupJob(id)
}

func (s *sqliteInternalDB) UpdateBackupJobStatus(id int64, status string, userID *int64) error {
	record := goqu.Record{
		"status":     status,
		"updated_at": goqu.L("CURRENT_TIMESTAMP"),
	}
	if userID != nil {
		record["updated_by"] = *userID
	}

	result, err := s.goqu.Update("backup_jobs").
		Prepared(true).
		Set(record).
		Where(goqu.Ex{"id": id}).
		Executor().
		Exec()
	if err != nil {
		s.logger.Error("failed to update backup job status", logger.Field{Key: "id", Value: id}, logger.Field{Key: "error", Value: err})
		return fmt.Errorf("failed to update backup job status %d: %w", id, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected for backup job status %d: %w", id, err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("%w: backup job %d", ErrBackupNotFound, id)
	}

	return nil
}

func (s *sqliteInternalDB) DeleteBackupJob(id int64, userID *int64) error {
	record := goqu.Record{
		"status":     BackupJobStatusDeleted,
		"deleted_at": goqu.L("CURRENT_TIMESTAMP"),
		"updated_at": goqu.L("CURRENT_TIMESTAMP"),
	}
	if userID != nil {
		record["deleted_by"] = *userID
		record["updated_by"] = *userID
	}

	result, err := s.goqu.Update("backup_jobs").
		Prepared(true).
		Set(record).
		Where(goqu.Ex{"id": id}).
		Executor().
		Exec()
	if err != nil {
		s.logger.Error("failed to delete backup job", logger.Field{Key: "id", Value: id}, logger.Field{Key: "error", Value: err})
		return fmt.Errorf("failed to delete backup job %d: %w", id, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected for backup job delete %d: %w", id, err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("%w: backup job %d", ErrBackupNotFound, id)
	}

	return nil
}

func (s *sqliteInternalDB) CreateBackupRun(jobID int64, triggerType string, previousJobStatus string, userID *int64) (*BackupRun, error) {
	tx, err := s.BeginTx()
	if err != nil {
		return nil, err
	}

	rollback := func() { _ = tx.Rollback() }

	record := goqu.Record{
		"job_id":              jobID,
		"trigger_type":        triggerType,
		"status":              BackupRunStatusRunning,
		"previous_job_status": previousJobStatus,
	}
	if userID != nil {
		record["created_by"] = *userID
	}

	result, err := tx.Insert("backup_runs").
		Prepared(true).
		Rows(record).
		Executor().
		Exec()
	if err != nil {
		rollback()
		s.logger.Error("failed to create backup run", logger.Field{Key: "job_id", Value: jobID}, logger.Field{Key: "error", Value: err})
		return nil, fmt.Errorf("failed to create backup run: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		rollback()
		return nil, fmt.Errorf("failed to get backup run id: %w", err)
	}

	if _, err := tx.Update("backup_jobs").
		Prepared(true).
		Set(goqu.Record{
			"status":      BackupJobStatusRunning,
			"last_run_at": goqu.L("CURRENT_TIMESTAMP"),
			"updated_at":  goqu.L("CURRENT_TIMESTAMP"),
		}).
		Where(goqu.Ex{"id": jobID}).
		Executor().
		Exec(); err != nil {
		rollback()
		return nil, fmt.Errorf("failed to update backup job run state: %w", err)
	}

	if err := tx.Commit(); err != nil {
		rollback()
		return nil, fmt.Errorf("failed to commit backup run creation: %w", err)
	}

	return s.GetBackupRun(id)
}

func (s *sqliteInternalDB) CreateSkippedBackupRun(jobID int64, triggerType string, previousJobStatus string, output string, userID *int64) (*BackupRun, error) {
	record := goqu.Record{
		"job_id":              jobID,
		"trigger_type":        triggerType,
		"status":              BackupRunStatusSkipped,
		"previous_job_status": previousJobStatus,
		"finished_at":         goqu.L("CURRENT_TIMESTAMP"),
		"updated_at":          goqu.L("CURRENT_TIMESTAMP"),
		"output":              output,
	}
	if userID != nil {
		record["created_by"] = *userID
	}

	result, err := s.goqu.Insert("backup_runs").
		Prepared(true).
		Rows(record).
		Executor().
		Exec()
	if err != nil {
		s.logger.Error("failed to create skipped backup run", logger.Field{Key: "job_id", Value: jobID}, logger.Field{Key: "error", Value: err})
		return nil, fmt.Errorf("failed to create skipped backup run: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get skipped backup run id: %w", err)
	}

	return s.GetBackupRun(id)
}

func (s *sqliteInternalDB) GetBackupRuns(jobID int64, page int, pageSize int) ([]BackupRun, int64, error) {
	if page < 1 {
		page = 1
	}

	if pageSize < 1 {
		pageSize = 10
	}

	query := s.goqu.From("backup_runs").
		Prepared(true).
		Where(goqu.Ex{"job_id": jobID})

	var totalCount int64
	if _, err := query.Select(goqu.COUNT("*")).ScanVal(&totalCount); err != nil {
		s.logger.Error("failed to get backup runs count", logger.Field{Key: "job_id", Value: jobID}, logger.Field{Key: "error", Value: err})
		return nil, 0, fmt.Errorf("failed to get backup runs count: %w", err)
	}

	runs := make([]BackupRun, 0)
	offset := (page - 1) * pageSize
	err := query.
		Order(goqu.C("started_at").Desc(), goqu.C("id").Desc()).
		Limit(uint(pageSize)).
		Offset(uint(offset)).
		ScanStructs(&runs)
	if err != nil {
		s.logger.Error("failed to get backup runs", logger.Field{Key: "job_id", Value: jobID}, logger.Field{Key: "error", Value: err})
		return nil, 0, fmt.Errorf("failed to get backup runs: %w", err)
	}

	return runs, totalCount, nil
}

func (s *sqliteInternalDB) GetBackupRun(id int64) (*BackupRun, error) {
	var run BackupRun
	found, err := s.goqu.From("backup_runs").
		Prepared(true).
		Where(goqu.Ex{"id": id}).
		ScanStruct(&run)
	if err != nil {
		s.logger.Error("failed to get backup run", logger.Field{Key: "id", Value: id}, logger.Field{Key: "error", Value: err})
		return nil, fmt.Errorf("failed to get backup run %d: %w", id, err)
	}

	if !found {
		return nil, fmt.Errorf("%w: backup run %d", ErrBackupNotFound, id)
	}

	return &run, nil
}

func (s *sqliteInternalDB) GetRunningBackupRunForJob(jobID int64) (*BackupRun, error) {
	var run BackupRun
	found, err := s.goqu.From("backup_runs").
		Prepared(true).
		Where(goqu.Ex{"job_id": jobID, "status": BackupRunStatusRunning}).
		Order(goqu.C("started_at").Desc()).
		Limit(1).
		ScanStruct(&run)
	if err != nil {
		s.logger.Error("failed to get running backup run", logger.Field{Key: "job_id", Value: jobID}, logger.Field{Key: "error", Value: err})
		return nil, fmt.Errorf("failed to get running backup run: %w", err)
	}

	if !found {
		return nil, nil
	}

	return &run, nil
}

func (s *sqliteInternalDB) FinishBackupRun(runID int64, jobID int64, runStatus string, jobStatus string, output *string, errorDetails *string) error {
	tx, err := s.BeginTx()
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	result, err := tx.Update("backup_runs").
		Prepared(true).
		Set(goqu.Record{
			"status":        runStatus,
			"finished_at":   goqu.L("CURRENT_TIMESTAMP"),
			"updated_at":    goqu.L("CURRENT_TIMESTAMP"),
			"output":        output,
			"error_details": errorDetails,
		}).
		Where(goqu.Ex{"id": runID}).
		Executor().
		Exec()
	if err != nil {
		return fmt.Errorf("failed to finish backup run %d: %w", runID, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected for backup run finish %d: %w", runID, err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("failed to finish backup run %d: affected %d rows", runID, rowsAffected)
	}

	result, err = tx.Update("backup_jobs").
		Prepared(true).
		Set(goqu.Record{
			"status":     jobStatus,
			"updated_at": goqu.L("CURRENT_TIMESTAMP"),
		}).
		Where(goqu.Ex{"id": jobID}).
		Executor().
		Exec()
	if err != nil {
		return fmt.Errorf("failed to restore backup job status %d: %w", jobID, err)
	}

	rowsAffected, err = result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected for backup job restore %d: %w", jobID, err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("failed to restore backup job status %d: affected %d rows", jobID, rowsAffected)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit backup run finish: %w", err)
	}

	return nil
}

func (s *sqliteInternalDB) MarkBackupRunCancelRequested(runID int64) error {
	result, err := s.goqu.Update("backup_runs").
		Prepared(true).
		Set(goqu.Record{
			"cancel_requested_at": goqu.L("CURRENT_TIMESTAMP"),
			"updated_at":          goqu.L("CURRENT_TIMESTAMP"),
		}).
		Where(goqu.Ex{"id": runID, "status": BackupRunStatusRunning}).
		Executor().
		Exec()
	if err != nil {
		s.logger.Error("failed to mark backup run cancellation", logger.Field{Key: "run_id", Value: runID}, logger.Field{Key: "error", Value: err})
		return fmt.Errorf("failed to mark backup run cancellation: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected for backup run cancellation: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("backup run %d is not running", runID)
	}

	return nil
}

func (s *sqliteInternalDB) CreateBackupRunFile(runID int64, itemName string, filePath string, fileSize int64) (*BackupRunFile, error) {
	result, err := s.goqu.Insert("backup_run_files").
		Prepared(true).
		Rows(goqu.Record{
			"run_id":    runID,
			"item_name": itemName,
			"file_path": filePath,
			"file_size": fileSize,
		}).
		Executor().
		Exec()
	if err != nil {
		s.logger.Error("failed to create backup run file", logger.Field{Key: "run_id", Value: runID}, logger.Field{Key: "error", Value: err})
		return nil, fmt.Errorf("failed to create backup run file: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get backup run file id: %w", err)
	}

	return s.GetBackupRunFile(id)
}

func (s *sqliteInternalDB) GetBackupRunFiles(runID int64) ([]BackupRunFile, error) {
	files := make([]BackupRunFile, 0)
	err := s.goqu.From("backup_run_files").
		Prepared(true).
		Where(goqu.Ex{"run_id": runID}).
		Order(goqu.C("created_at").Asc(), goqu.C("id").Asc()).
		ScanStructs(&files)
	if err != nil {
		s.logger.Error("failed to get backup run files", logger.Field{Key: "run_id", Value: runID}, logger.Field{Key: "error", Value: err})
		return nil, fmt.Errorf("failed to get backup run files: %w", err)
	}

	return files, nil
}

func (s *sqliteInternalDB) GetBackupRunFile(id int64) (*BackupRunFile, error) {
	var file BackupRunFile
	found, err := s.goqu.From("backup_run_files").
		Prepared(true).
		Where(goqu.Ex{"id": id}).
		ScanStruct(&file)
	if err != nil {
		s.logger.Error("failed to get backup run file", logger.Field{Key: "id", Value: id}, logger.Field{Key: "error", Value: err})
		return nil, fmt.Errorf("failed to get backup run file %d: %w", id, err)
	}

	if !found {
		return nil, fmt.Errorf("%w: backup run file %d", ErrBackupNotFound, id)
	}

	return &file, nil
}

func (s *sqliteInternalDB) MarkOrphanedBackupRunsFailed() error {
	runs := make([]BackupRun, 0)
	err := s.goqu.From("backup_runs").
		Prepared(true).
		Where(goqu.Ex{"status": BackupRunStatusRunning}).
		ScanStructs(&runs)
	if err != nil {
		return fmt.Errorf("failed to get orphaned backup runs: %w", err)
	}

	for _, run := range runs {
		output := "Run was marked failed because the service restarted before it finished."
		errorDetails := output
		if err := s.FinishBackupRun(run.ID, run.JobID, BackupRunStatusFailed, run.PreviousJobStatus, &output, &errorDetails); err != nil {
			return err
		}
	}

	return nil
}

func backupJobRecord(payload BackupJobPayload) goqu.Record {
	return goqu.Record{
		"job_type":              payload.JobType,
		"tag":                   payload.Tag,
		"name":                  payload.Name,
		"status":                payload.Status,
		"cron_expression":       payload.CronExpression,
		"destination_directory": payload.DestinationDirectory,
		"archive_password":      payload.ArchivePassword,
		"source_path":           payload.SourcePath,
		"sql_host":              payload.SQLHost,
		"sql_port":              payload.SQLPort,
		"sql_username":          payload.SQLUsername,
		"sql_password":          payload.SQLPassword,
		"sql_database_names":    payload.SQLDatabaseNames,
	}
}
