package services

import (
	"context"
	"crypto/rand"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/omnihance/omnihance-a3-agent/internal/config"
	"github.com/omnihance/omnihance-a3-agent/internal/constants"
	"github.com/omnihance/omnihance-a3-agent/internal/db"
	"github.com/omnihance/omnihance-a3-agent/internal/logger"
	"github.com/robfig/cron/v3"
	"github.com/yeka/zip"

	_ "github.com/microsoft/go-mssqldb"
)

const (
	backupArchiveExtension = ".zip"
	backupSearchLimit      = 10
	backupSkipRunningMsg   = "Skipped because backup job is already running."
)

var (
	ErrBackupInvalid       = errors.New("invalid backup job")
	ErrBackupJobRunning    = errors.New("backup job is currently running")
	ErrBackupNotFound      = errors.New("backup item not found")
	ErrBackupRemoteSQLHost = errors.New("remote SQL Server backups are not supported")
	ErrBackupNoRunningJob  = errors.New("backup job is not running")
)

type BackupService interface {
	Start() error
	Stop() error
	GetJobs() ([]db.BackupJob, error)
	GetJob(id int64) (*db.BackupJob, error)
	CreateJob(ctx context.Context, payload db.BackupJobPayload, userID *int64) (*db.BackupJob, error)
	UpdateJob(ctx context.Context, id int64, payload db.BackupJobPayload, userID *int64) (*db.BackupJob, error)
	DeleteJob(ctx context.Context, id int64, userID *int64) error
	RunJob(ctx context.Context, id int64, triggerType string, userID *int64) (*db.BackupRun, error)
	CancelJob(ctx context.Context, id int64) (*db.BackupRun, error)
	GetRuns(jobID int64, page int, pageSize int) ([]db.BackupRun, int64, error)
	GetRunDetails(runID int64) (*BackupRunDetails, error)
	GetRunFile(fileID int64) (*db.BackupRunFile, error)
	SearchPaths(query string, kind string) ([]PathSearchResult, error)
	GetSQLServerDefaults() SQLServerBackupDefaults
}

type BackupRunDetails struct {
	Run   db.BackupRun       `json:"run"`
	Files []db.BackupRunFile `json:"files"`
}

type PathSearchResult struct {
	Name string `json:"name"`
	Path string `json:"path"`
	Kind string `json:"kind"`
}

type SQLServerBackupDefaults struct {
	Host               string `json:"host"`
	Port               int    `json:"port"`
	Username           string `json:"username"`
	Password           string `json:"password"`
	LocalServerRunning bool   `json:"local_server_running"`
}

type backupService struct {
	cfg              *config.EnvVars
	logger           logger.Logger
	internalDB       db.InternalDB
	fileEditor       FileEditorService
	cron             *cron.Cron
	cronEntries      map[int64]cron.EntryID
	runningCancels   map[int64]context.CancelFunc
	runningRunIDs    map[int64]int64
	mu               sync.Mutex
	wg               sync.WaitGroup
	started          bool
	sqlServerChecker func() (bool, error)
}

func NewBackupService(
	cfg *config.EnvVars,
	logger logger.Logger,
	internalDB db.InternalDB,
	fileEditor FileEditorService,
) BackupService {
	return &backupService{
		cfg:              cfg,
		logger:           logger,
		internalDB:       internalDB,
		fileEditor:       fileEditor,
		cronEntries:      map[int64]cron.EntryID{},
		runningCancels:   map[int64]context.CancelFunc{},
		runningRunIDs:    map[int64]int64{},
		sqlServerChecker: localSQLServerServiceRunning,
	}
}

func (s *backupService) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.started {
		return nil
	}

	if err := s.internalDB.MarkOrphanedBackupRunsFailed(); err != nil {
		return err
	}

	if err := s.resetLocks(); err != nil {
		return err
	}

	s.cron = cron.New(cron.WithParser(backupCronParser()))
	jobs, err := s.internalDB.GetSchedulableBackupJobs()
	if err != nil {
		return err
	}

	for _, job := range jobs {
		if err := s.scheduleJobLocked(job); err != nil {
			return err
		}
	}

	s.cron.Start()
	s.started = true
	s.logger.Info("backup service started", logger.Field{Key: "scheduled_jobs", Value: len(s.cronEntries)})

	return nil
}

func (s *backupService) Stop() error {
	s.mu.Lock()
	if !s.started {
		s.mu.Unlock()
		return nil
	}

	scheduler := s.cron
	for _, cancel := range s.runningCancels {
		cancel()
	}

	s.started = false
	s.cron = nil
	s.cronEntries = map[int64]cron.EntryID{}
	s.mu.Unlock()

	if scheduler != nil {
		ctx := scheduler.Stop()
		<-ctx.Done()
	}

	s.wg.Wait()
	s.logger.Info("backup service stopped")

	return nil
}

func (s *backupService) GetJobs() ([]db.BackupJob, error) {
	return s.internalDB.GetBackupJobs()
}

func (s *backupService) GetJob(id int64) (*db.BackupJob, error) {
	job, err := s.internalDB.GetBackupJob(id)
	if err != nil {
		return nil, backupNotFoundError(err)
	}

	return job, nil
}

func (s *backupService) CreateJob(ctx context.Context, payload db.BackupJobPayload, userID *int64) (*db.BackupJob, error) {
	normalizedPayload, err := s.validateJobPayload(ctx, payload)
	if err != nil {
		return nil, err
	}

	job, err := s.internalDB.CreateBackupJob(normalizedPayload, userID)
	if err != nil {
		return nil, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.scheduleJobLocked(*job); err != nil {
		return nil, err
	}

	return job, nil
}

func (s *backupService) UpdateJob(ctx context.Context, id int64, payload db.BackupJobPayload, userID *int64) (*db.BackupJob, error) {
	job, err := s.GetJob(id)
	if err != nil {
		return nil, err
	}

	if job.Status == db.BackupJobStatusRunning {
		return nil, ErrBackupJobRunning
	}

	if running, err := s.internalDB.GetRunningBackupRunForJob(id); err != nil {
		return nil, err
	} else if running != nil {
		return nil, ErrBackupJobRunning
	}

	if payload.ArchivePassword == nil {
		payload.ArchivePassword = job.ArchivePassword
	}

	if payload.SQLPassword == nil {
		payload.SQLPassword = job.SQLPassword
	}

	normalizedPayload, err := s.validateJobPayload(ctx, payload)
	if err != nil {
		return nil, err
	}

	updatedJob, err := s.internalDB.UpdateBackupJob(id, normalizedPayload, userID)
	if err != nil {
		return nil, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.unscheduleJobLocked(id)
	if err := s.scheduleJobLocked(*updatedJob); err != nil {
		return nil, err
	}

	return updatedJob, nil
}

func (s *backupService) DeleteJob(ctx context.Context, id int64, userID *int64) error {
	_ = ctx

	job, err := s.GetJob(id)
	if err != nil {
		return err
	}

	if job.Status == db.BackupJobStatusRunning {
		return ErrBackupJobRunning
	}

	if running, err := s.internalDB.GetRunningBackupRunForJob(id); err != nil {
		return err
	} else if running != nil {
		return ErrBackupJobRunning
	}

	if err := s.internalDB.DeleteBackupJob(id, userID); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.unscheduleJobLocked(id)

	return nil
}

func (s *backupService) RunJob(ctx context.Context, id int64, triggerType string, userID *int64) (*db.BackupRun, error) {
	_ = ctx

	job, err := s.GetJob(id)
	if err != nil {
		return nil, err
	}

	if job.Status == db.BackupJobStatusDeleted {
		return nil, fmt.Errorf("%w: backup job is deleted", ErrBackupInvalid)
	}

	if triggerType == "" {
		triggerType = db.BackupRunTriggerManual
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.runningCancels[id]; ok {
		if triggerType == db.BackupRunTriggerCron {
			return s.createSkippedCronRun(*job)
		}

		return nil, ErrBackupJobRunning
	}

	if running, err := s.internalDB.GetRunningBackupRunForJob(id); err != nil {
		return nil, err
	} else if running != nil {
		if triggerType == db.BackupRunTriggerCron {
			return s.createSkippedCronRun(*job)
		}

		return nil, ErrBackupJobRunning
	}

	lockPath, err := s.acquireJobLock(id)
	if err != nil {
		if triggerType == db.BackupRunTriggerCron && errors.Is(err, ErrBackupJobRunning) {
			return s.createSkippedCronRun(*job)
		}

		return nil, err
	}

	run, err := s.internalDB.CreateBackupRun(id, triggerType, job.Status, userID)
	if err != nil {
		s.releaseJobLock(lockPath)
		return nil, err
	}

	runCtx, cancel := context.WithCancel(context.Background())
	s.runningCancels[id] = cancel
	s.runningRunIDs[id] = run.ID
	s.wg.Add(1)
	go s.executeRun(runCtx, *job, *run, lockPath)

	return run, nil
}

func (s *backupService) createSkippedCronRun(job db.BackupJob) (*db.BackupRun, error) {
	return s.internalDB.CreateSkippedBackupRun(job.ID, db.BackupRunTriggerCron, job.Status, backupSkipRunningMsg, nil)
}

func (s *backupService) CancelJob(ctx context.Context, id int64) (*db.BackupRun, error) {
	_ = ctx

	run, err := s.internalDB.GetRunningBackupRunForJob(id)
	if err != nil {
		return nil, err
	}

	if run == nil {
		return nil, ErrBackupNoRunningJob
	}

	if err := s.internalDB.MarkBackupRunCancelRequested(run.ID); err != nil {
		return nil, err
	}

	s.mu.Lock()
	cancel := s.runningCancels[id]
	s.mu.Unlock()

	if cancel != nil {
		cancel()
	}

	return s.internalDB.GetBackupRun(run.ID)
}

func (s *backupService) GetRuns(jobID int64, page int, pageSize int) ([]db.BackupRun, int64, error) {
	if _, err := s.GetJob(jobID); err != nil {
		return nil, 0, err
	}

	return s.internalDB.GetBackupRuns(jobID, page, pageSize)
}

func (s *backupService) GetRunDetails(runID int64) (*BackupRunDetails, error) {
	run, err := s.internalDB.GetBackupRun(runID)
	if err != nil {
		return nil, backupNotFoundError(err)
	}

	files, err := s.internalDB.GetBackupRunFiles(run.ID)
	if err != nil {
		return nil, err
	}

	return &BackupRunDetails{
		Run:   *run,
		Files: files,
	}, nil
}

func (s *backupService) GetRunFile(fileID int64) (*db.BackupRunFile, error) {
	file, err := s.internalDB.GetBackupRunFile(fileID)
	if err != nil {
		return nil, backupNotFoundError(err)
	}

	return file, nil
}

func backupNotFoundError(err error) error {
	if errors.Is(err, db.ErrBackupNotFound) || errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("%w: %w", ErrBackupNotFound, err)
	}

	return err
}

func (s *backupService) SearchPaths(query string, kind string) ([]PathSearchResult, error) {
	query = strings.TrimSpace(query)
	directoryOnly := kind == "directory"
	if query == "" {
		return s.searchRootPaths(directoryOnly), nil
	}

	cleanQuery := filepath.Clean(query)
	info, err := s.fileEditor.Stat(cleanQuery)
	var baseDir string
	var prefix string
	if err == nil && info.IsDir() {
		baseDir = cleanQuery
		prefix = ""
	} else {
		baseDir = filepath.Dir(cleanQuery)
		prefix = filepath.Base(cleanQuery)
		if baseDir == "." {
			baseDir = ""
		}
	}

	if baseDir == "" {
		return s.searchRootPaths(directoryOnly), nil
	}

	entries, err := s.fileEditor.ReadDir(baseDir)
	if err != nil {
		return []PathSearchResult{}, nil
	}

	prefix = strings.ToLower(prefix)
	results := make([]PathSearchResult, 0, backupSearchLimit)
	for _, entry := range entries {
		isDir := entry.IsDir()
		if directoryOnly && !isDir {
			continue
		}

		if prefix != "" && !strings.Contains(strings.ToLower(entry.Name()), prefix) {
			continue
		}

		resultKind := "file"
		if isDir {
			resultKind = "directory"
		}

		results = append(results, PathSearchResult{
			Name: entry.Name(),
			Path: filepath.Join(baseDir, entry.Name()),
			Kind: resultKind,
		})
	}

	sort.SliceStable(results, func(i int, j int) bool {
		if results[i].Kind != results[j].Kind {
			return results[i].Kind == "directory"
		}

		return strings.ToLower(results[i].Name) < strings.ToLower(results[j].Name)
	})

	if len(results) > backupSearchLimit {
		return results[:backupSearchLimit], nil
	}

	return results, nil
}

func (s *backupService) GetSQLServerDefaults() SQLServerBackupDefaults {
	defaults := SQLServerBackupDefaults{
		Host: "127.0.0.1",
		Port: 1433,
	}

	if setting, err := s.internalDB.GetSetting(constants.SettingKeyDBHost); err == nil {
		defaults.Host = setting.Value
	}

	if setting, err := s.internalDB.GetSetting(constants.SettingKeyDBPort); err == nil {
		if port, parseErr := strconv.Atoi(setting.Value); parseErr == nil {
			defaults.Port = port
		}
	}

	if setting, err := s.internalDB.GetSetting(constants.SettingKeyDBUser); err == nil {
		defaults.Username = setting.Value
	}

	if setting, err := s.internalDB.GetSetting(constants.SettingKeyDBPass); err == nil {
		defaults.Password = setting.Value
	}

	running, err := s.sqlServerChecker()
	defaults.LocalServerRunning = err == nil && running

	return defaults
}

func (s *backupService) validateJobPayload(ctx context.Context, payload db.BackupJobPayload) (db.BackupJobPayload, error) {
	_ = ctx

	payload.Name = strings.TrimSpace(payload.Name)
	payload.JobType = strings.TrimSpace(payload.JobType)
	payload.Status = strings.TrimSpace(payload.Status)
	payload.DestinationDirectory = filepath.Clean(strings.TrimSpace(payload.DestinationDirectory))
	payload.CronExpression = normalizeOptionalString(payload.CronExpression, true)
	payload.ArchivePassword = normalizeOptionalString(payload.ArchivePassword, false)
	payload.SourcePath = normalizeOptionalPath(payload.SourcePath)
	payload.SQLHost = normalizeOptionalString(payload.SQLHost, true)
	payload.SQLUsername = normalizeOptionalString(payload.SQLUsername, true)
	payload.SQLPassword = normalizeOptionalString(payload.SQLPassword, false)
	payload.SQLDatabaseNames = normalizeOptionalString(payload.SQLDatabaseNames, true)

	if payload.Name == "" {
		return payload, fmt.Errorf("%w: job name is required", ErrBackupInvalid)
	}

	if payload.JobType != db.BackupJobTypeFile && payload.JobType != db.BackupJobTypeSQLServer {
		return payload, fmt.Errorf("%w: unsupported job type", ErrBackupInvalid)
	}

	if payload.Status == "" {
		payload.Status = db.BackupJobStatusActive
	}

	if payload.Status != db.BackupJobStatusActive && payload.Status != db.BackupJobStatusInactive {
		return payload, fmt.Errorf("%w: status must be active or inactive", ErrBackupInvalid)
	}

	if payload.DestinationDirectory == "" || payload.DestinationDirectory == "." {
		return payload, fmt.Errorf("%w: destination directory is required", ErrBackupInvalid)
	}

	if info, err := s.fileEditor.Stat(payload.DestinationDirectory); err == nil && !info.IsDir() {
		return payload, fmt.Errorf("%w: destination must be a directory", ErrBackupInvalid)
	} else if err != nil && !s.fileEditor.IsNotExist(err) {
		return payload, fmt.Errorf("%w: cannot access destination directory: %v", ErrBackupInvalid, err)
	}

	if payload.CronExpression != nil {
		if _, err := backupCronParser().Parse(*payload.CronExpression); err != nil {
			return payload, fmt.Errorf("%w: invalid cron expression: %v", ErrBackupInvalid, err)
		}
	}

	if payload.JobType == db.BackupJobTypeFile {
		if payload.SourcePath == nil || *payload.SourcePath == "" {
			return payload, fmt.Errorf("%w: source path is required", ErrBackupInvalid)
		}

		if _, err := s.fileEditor.Stat(*payload.SourcePath); err != nil {
			if s.fileEditor.IsNotExist(err) {
				return payload, fmt.Errorf("%w: source path does not exist", ErrBackupInvalid)
			}

			return payload, fmt.Errorf("%w: cannot access source path: %v", ErrBackupInvalid, err)
		}

		payload.SQLHost = nil
		payload.SQLPort = nil
		payload.SQLUsername = nil
		payload.SQLPassword = nil
		payload.SQLDatabaseNames = nil
		return payload, nil
	}

	if payload.SQLHost == nil || *payload.SQLHost == "" {
		return payload, fmt.Errorf("%w: SQL Server host is required", ErrBackupInvalid)
	}

	if !isLocalSQLHost(*payload.SQLHost) {
		return payload, fmt.Errorf("%w: %s", ErrBackupRemoteSQLHost, *payload.SQLHost)
	}

	if payload.SQLPort == nil || *payload.SQLPort < 1 || *payload.SQLPort > 65535 {
		return payload, fmt.Errorf("%w: SQL Server port must be between 1 and 65535", ErrBackupInvalid)
	}

	if payload.SQLUsername == nil || *payload.SQLUsername == "" {
		return payload, fmt.Errorf("%w: SQL Server username is required", ErrBackupInvalid)
	}

	if payload.SQLPassword == nil {
		empty := ""
		payload.SQLPassword = &empty
	}

	if payload.SQLDatabaseNames == nil || parseDatabaseNames(*payload.SQLDatabaseNames) == nil {
		return payload, fmt.Errorf("%w: at least one SQL Server database name is required", ErrBackupInvalid)
	}

	payload.SourcePath = nil

	return payload, nil
}

func (s *backupService) executeRun(ctx context.Context, job db.BackupJob, run db.BackupRun, lockPath string) {
	defer s.wg.Done()
	defer s.releaseJobLock(lockPath)
	defer s.clearRunning(job.ID)

	var output strings.Builder
	runStatus := db.BackupRunStatusSucceeded
	var errorDetails *string

	if err := s.fileEditor.MkdirAll(job.DestinationDirectory, 0755); err != nil {
		runStatus = db.BackupRunStatusFailed
		msg := "Failed to create destination directory: " + err.Error()
		errorDetails = &msg
		output.WriteString(msg)
		_ = s.internalDB.FinishBackupRun(run.ID, job.ID, runStatus, run.PreviousJobStatus, stringPointer(output.String()), errorDetails)
		return
	}

	info, err := s.fileEditor.Stat(job.DestinationDirectory)
	if err != nil || !info.IsDir() {
		runStatus = db.BackupRunStatusFailed
		msg := "Destination path is not a directory"
		if err != nil {
			msg = "Cannot access destination directory: " + err.Error()
		}

		errorDetails = &msg
		output.WriteString(msg)
		_ = s.internalDB.FinishBackupRun(run.ID, job.ID, runStatus, run.PreviousJobStatus, stringPointer(output.String()), errorDetails)
		return
	}

	output.WriteString("Backup started.\n")
	switch job.JobType {
	case db.BackupJobTypeFile:
		err = s.runFileBackup(ctx, job, run, &output)
	case db.BackupJobTypeSQLServer:
		err = s.runSQLServerBackup(ctx, job, run, &output)
	default:
		err = fmt.Errorf("unsupported backup job type %s", job.JobType)
	}

	if err != nil {
		if ctx.Err() != nil {
			runStatus = db.BackupRunStatusCancelled
			msg := "Backup cancelled."
			errorDetails = &msg
			output.WriteString(msg)
		} else {
			runStatus = db.BackupRunStatusFailed
			msg := err.Error()
			errorDetails = &msg
			output.WriteString("Backup failed: " + msg)
		}
	} else if ctx.Err() != nil {
		runStatus = db.BackupRunStatusCancelled
		msg := "Backup cancelled."
		errorDetails = &msg
		output.WriteString(msg)
	} else {
		output.WriteString("Backup completed successfully.")
	}

	if err := s.internalDB.FinishBackupRun(run.ID, job.ID, runStatus, run.PreviousJobStatus, stringPointer(output.String()), errorDetails); err != nil {
		s.logger.Error("failed to finish backup run", logger.Field{Key: "run_id", Value: run.ID}, logger.Field{Key: "error", Value: err})
	}
}

func (s *backupService) runFileBackup(ctx context.Context, job db.BackupJob, run db.BackupRun, output *strings.Builder) error {
	if job.SourcePath == nil {
		return errors.New("source path is missing")
	}

	sourceInfo, err := s.fileEditor.Stat(*job.SourcePath)
	if err != nil {
		return err
	}

	itemName := filepath.Base(*job.SourcePath)
	if itemName == "." || itemName == string(filepath.Separator) {
		itemName = "files"
	}

	archivePath, err := uniqueBackupPath(job.DestinationDirectory, itemName, job.ID)
	if err != nil {
		return err
	}

	output.WriteString("Creating file archive " + archivePath + "\n")
	if err := createZipArchive(ctx, *job.SourcePath, archivePath, stringValue(job.ArchivePassword)); err != nil {
		_ = s.fileEditor.Remove(archivePath)
		return err
	}

	archiveInfo, err := s.fileEditor.Stat(archivePath)
	if err != nil {
		return err
	}

	if _, err := s.internalDB.CreateBackupRunFile(run.ID, itemName, archivePath, archiveInfo.Size()); err != nil {
		return err
	}

	_, _ = fmt.Fprintf(output, "Archived %s (%d bytes).\n", sourceInfo.Name(), archiveInfo.Size())

	return nil
}

func (s *backupService) runSQLServerBackup(ctx context.Context, job db.BackupJob, run db.BackupRun, output *strings.Builder) error {
	names := parseDatabaseNames(stringValue(job.SQLDatabaseNames))
	if len(names) == 0 {
		return errors.New("no database names configured")
	}

	if !isLocalSQLHost(stringValue(job.SQLHost)) {
		return fmt.Errorf("%w: %s", ErrBackupRemoteSQLHost, stringValue(job.SQLHost))
	}

	running, err := s.sqlServerChecker()
	if err != nil {
		return fmt.Errorf("failed to check local SQL Server service: %w", err)
	}

	if !running {
		return errors.New("local SQL Server service is not installed or running")
	}

	sqlDB, err := openSQLServerDatabase(ctx, job)
	if err != nil {
		return err
	}
	defer func() {
		_ = sqlDB.Close()
	}()

	for _, databaseName := range names {
		if err := ctx.Err(); err != nil {
			return err
		}

		bakPath, err := uniqueBackupPath(job.DestinationDirectory, databaseName, job.ID)
		if err != nil {
			return err
		}

		bakPath = strings.TrimSuffix(bakPath, backupArchiveExtension) + ".bak"
		archivePath := strings.TrimSuffix(bakPath, ".bak") + backupArchiveExtension
		query := fmt.Sprintf(
			"BACKUP DATABASE %s TO DISK = N'%s' WITH INIT, COPY_ONLY, CHECKSUM, COMPRESSION",
			quoteSQLIdentifier(databaseName),
			escapeSQLString(bakPath),
		)

		output.WriteString("Backing up database " + databaseName + " to " + bakPath + "\n")
		if _, err := sqlDB.ExecContext(ctx, query); err != nil {
			return fmt.Errorf("failed to backup database %s: %w", databaseName, err)
		}

		output.WriteString("Compressing " + databaseName + " backup to " + archivePath + "\n")
		if err := createZipArchive(ctx, bakPath, archivePath, stringValue(job.ArchivePassword)); err != nil {
			_ = s.fileEditor.Remove(archivePath)
			return err
		}

		archiveInfo, err := s.fileEditor.Stat(archivePath)
		if err != nil {
			return err
		}

		if _, err := s.internalDB.CreateBackupRunFile(run.ID, databaseName, archivePath, archiveInfo.Size()); err != nil {
			return err
		}

		if err := s.fileEditor.Remove(bakPath); err != nil {
			s.logger.Warn("failed to remove intermediate SQL backup", logger.Field{Key: "path", Value: bakPath}, logger.Field{Key: "error", Value: err})
		}
	}

	return nil
}

func (s *backupService) scheduleJobLocked(job db.BackupJob) error {
	if s.cron == nil || job.Status != db.BackupJobStatusActive || job.CronExpression == nil || *job.CronExpression == "" {
		return nil
	}

	entryID, err := s.cron.AddFunc(*job.CronExpression, func() {
		if _, err := s.RunJob(context.Background(), job.ID, db.BackupRunTriggerCron, nil); err != nil {
			s.logger.Warn("scheduled backup job did not start", logger.Field{Key: "job_id", Value: job.ID}, logger.Field{Key: "error", Value: err})
		}
	})
	if err != nil {
		return err
	}

	s.cronEntries[job.ID] = entryID

	return nil
}

func (s *backupService) unscheduleJobLocked(jobID int64) {
	if s.cron == nil {
		return
	}

	entryID, ok := s.cronEntries[jobID]
	if !ok {
		return
	}

	s.cron.Remove(entryID)
	delete(s.cronEntries, jobID)
}

func (s *backupService) acquireJobLock(jobID int64) (string, error) {
	locksDir := filepath.Join(s.cfg.BackupsDirectory, "locks")
	if err := s.fileEditor.MkdirAll(locksDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create backup locks directory: %w", err)
	}

	lockPath := filepath.Join(locksDir, fmt.Sprintf("job-%d.lock", jobID))
	lockFile, err := s.fileEditor.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if err != nil {
		if s.fileEditor.IsExist(err) {
			return "", ErrBackupJobRunning
		}

		return "", fmt.Errorf("failed to create backup lock file: %w", err)
	}

	if err := lockFile.Close(); err != nil {
		s.logger.Error("failed to close backup lock file", logger.Field{Key: "error", Value: err})
	}

	return lockPath, nil
}

func (s *backupService) releaseJobLock(lockPath string) {
	if lockPath == "" {
		return
	}

	if err := s.fileEditor.Remove(lockPath); err != nil && !s.fileEditor.IsNotExist(err) {
		s.logger.Error("failed to remove backup lock file", logger.Field{Key: "path", Value: lockPath}, logger.Field{Key: "error", Value: err})
	}
}

func (s *backupService) clearRunning(jobID int64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.runningCancels, jobID)
	delete(s.runningRunIDs, jobID)
}

func (s *backupService) resetLocks() error {
	locksDir := filepath.Join(s.cfg.BackupsDirectory, "locks")
	if err := s.fileEditor.RemoveAll(locksDir); err != nil && !s.fileEditor.IsNotExist(err) {
		return err
	}

	return s.fileEditor.MkdirAll(locksDir, 0755)
}

func (s *backupService) searchRootPaths(directoryOnly bool) []PathSearchResult {
	results := make([]PathSearchResult, 0, backupSearchLimit)
	if runtime.GOOS == "windows" {
		for _, drive := range "ABCDEFGHIJKLMNOPQRSTUVWXYZ" {
			path := string(drive) + ":\\"
			if info, err := s.fileEditor.Stat(path); err == nil && info.IsDir() {
				results = append(results, PathSearchResult{Name: string(drive) + ":", Path: path, Kind: "directory"})
			}

			if len(results) >= backupSearchLimit {
				break
			}
		}

		return results
	}

	entries, err := s.fileEditor.ReadDir("/")
	if err != nil {
		return []PathSearchResult{{Name: "/", Path: "/", Kind: "directory"}}
	}

	results = append(results, PathSearchResult{Name: "/", Path: "/", Kind: "directory"})
	for _, entry := range entries {
		if directoryOnly && !entry.IsDir() {
			continue
		}

		kind := "file"
		if entry.IsDir() {
			kind = "directory"
		}

		results = append(results, PathSearchResult{Name: entry.Name(), Path: filepath.Join("/", entry.Name()), Kind: kind})
		if len(results) >= backupSearchLimit {
			break
		}
	}

	return results
}

func backupCronParser() cron.Parser {
	return cron.NewParser(cron.SecondOptional | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)
}

func openSQLServerDatabase(ctx context.Context, job db.BackupJob) (*sql.DB, error) {
	host := stringValue(job.SQLHost)
	port := 0
	if job.SQLPort != nil {
		port = *job.SQLPort
	}

	address := host
	if port > 0 && !strings.Contains(host, `\`) && !strings.Contains(host, ",") {
		address = net.JoinHostPort(host, strconv.Itoa(port))
	}

	connectionString := fmt.Sprintf(
		"sqlserver://%s:%s@%s?database=master&encrypt=disable",
		url.QueryEscape(stringValue(job.SQLUsername)),
		url.QueryEscape(stringValue(job.SQLPassword)),
		address,
	)

	sqlDB, err := sql.Open("sqlserver", connectionString)
	if err != nil {
		return nil, err
	}

	pingCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	if err := sqlDB.PingContext(pingCtx); err != nil {
		_ = sqlDB.Close()
		return nil, err
	}

	return sqlDB, nil
}

func createZipArchive(ctx context.Context, sourcePath string, archivePath string, password string) error {
	archiveFile, err := os.Create(archivePath)
	if err != nil {
		return err
	}

	writer := zip.NewWriter(archiveFile)
	closeArchive := func() error {
		if err := writer.Close(); err != nil {
			_ = archiveFile.Close()
			return err
		}

		return archiveFile.Close()
	}

	sourceInfo, err := os.Stat(sourcePath)
	if err != nil {
		_ = archiveFile.Close()
		return err
	}

	baseDir := filepath.Dir(sourcePath)
	if sourceInfo.IsDir() {
		baseDir = filepath.Dir(sourcePath)
	}

	err = filepath.WalkDir(sourcePath, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		if err := ctx.Err(); err != nil {
			return err
		}

		info, err := entry.Info()
		if err != nil {
			return err
		}

		name, err := filepath.Rel(baseDir, path)
		if err != nil {
			return err
		}

		name = filepath.ToSlash(name)
		if entry.IsDir() {
			if name == "." {
				return nil
			}

			header := &zip.FileHeader{Name: strings.TrimSuffix(name, "/") + "/"}
			header.SetMode(info.Mode())
			_, err := writer.CreateHeader(header)
			return err
		}

		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}

		header.Name = name
		header.Method = zip.Deflate
		if password != "" {
			header.SetPassword(password)
			header.SetEncryptionMethod(zip.AES256Encryption)
		}

		zipFile, err := writer.CreateHeader(header)
		if err != nil {
			return err
		}

		sourceFile, err := os.Open(path)
		if err != nil {
			return err
		}

		_, copyErr := copyWithContext(ctx, zipFile, sourceFile)
		closeErr := sourceFile.Close()
		if copyErr != nil {
			return copyErr
		}

		return closeErr
	})
	if err != nil {
		_ = archiveFile.Close()
		return err
	}

	return closeArchive()
}

func copyWithContext(ctx context.Context, dst io.Writer, src io.Reader) (int64, error) {
	buffer := make([]byte, 1024*128)
	var written int64
	for {
		if err := ctx.Err(); err != nil {
			return written, err
		}

		nr, er := src.Read(buffer)
		if nr > 0 {
			nw, ew := dst.Write(buffer[0:nr])
			if nw > 0 {
				written += int64(nw)
			}

			if ew != nil {
				return written, ew
			}

			if nr != nw {
				return written, io.ErrShortWrite
			}
		}

		if er != nil {
			if er == io.EOF {
				break
			}

			return written, er
		}
	}

	return written, nil
}

func uniqueBackupPath(directory string, itemName string, jobID int64) (string, error) {
	timestamp := time.Now().Format("20060102150405")
	baseName := fmt.Sprintf("%s-%d-%s", safeBackupName(itemName), jobID, timestamp)
	path := filepath.Join(directory, baseName+backupArchiveExtension)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return path, nil
	} else if err != nil {
		return "", err
	}

	for i := 0; i < 10; i++ {
		suffix, err := randomNumericSuffix()
		if err != nil {
			return "", err
		}

		path = filepath.Join(directory, baseName+"-"+suffix+backupArchiveExtension)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			return path, nil
		} else if err != nil {
			return "", err
		}
	}

	return "", errors.New("failed to generate unique backup file name")
}

func randomNumericSuffix() (string, error) {
	value, err := rand.Int(rand.Reader, big.NewInt(900000))
	if err != nil {
		return "", err
	}

	return strconv.FormatInt(value.Int64()+100000, 10), nil
}

func safeBackupName(name string) string {
	name = strings.TrimSpace(name)
	var builder strings.Builder
	for _, char := range name {
		if unicode.IsLetter(char) || unicode.IsDigit(char) || char == '-' || char == '_' || char == '.' {
			builder.WriteRune(char)
			continue
		}

		builder.WriteRune('_')
	}

	value := strings.Trim(builder.String(), "._-")
	if value == "" {
		return "backup"
	}

	return value
}

func parseDatabaseNames(value string) []string {
	parts := strings.Split(value, ",")
	names := make([]string, 0, len(parts))
	for _, part := range parts {
		name := strings.TrimSpace(part)
		if name == "" {
			continue
		}

		names = append(names, name)
	}

	return names
}

func quoteSQLIdentifier(identifier string) string {
	return "[" + strings.ReplaceAll(identifier, "]", "]]") + "]"
}

func escapeSQLString(value string) string {
	return strings.ReplaceAll(value, "'", "''")
}

func isLocalSQLHost(host string) bool {
	host = strings.TrimSpace(strings.ToLower(host))
	host = strings.TrimPrefix(host, "tcp:")
	if slashIndex := strings.Index(host, `\`); slashIndex >= 0 {
		host = host[:slashIndex]
	}

	if commaIndex := strings.Index(host, ","); commaIndex >= 0 {
		host = host[:commaIndex]
	}

	host = strings.Trim(host, "[] ")
	if host == "" || host == "." || host == "(local)" || host == "localhost" || host == "127.0.0.1" || host == "::1" {
		return true
	}

	if ip := net.ParseIP(host); ip != nil {
		if ip.IsLoopback() {
			return true
		}

		return isLocalInterfaceIP(ip)
	}

	hostname, err := os.Hostname()
	if err == nil {
		normalizedHostname := strings.ToLower(hostname)
		if host == normalizedHostname || strings.HasPrefix(host, normalizedHostname+".") {
			return true
		}
	}

	ips, err := net.LookupIP(host)
	if err != nil {
		return false
	}

	for _, ip := range ips {
		if ip.IsLoopback() || isLocalInterfaceIP(ip) {
			return true
		}
	}

	return false
}

func isLocalInterfaceIP(ip net.IP) bool {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return false
	}

	for _, addr := range addrs {
		var currentIP net.IP
		switch value := addr.(type) {
		case *net.IPNet:
			currentIP = value.IP
		case *net.IPAddr:
			currentIP = value.IP
		}

		if currentIP != nil && currentIP.Equal(ip) {
			return true
		}
	}

	return false
}

func normalizeOptionalString(value *string, trim bool) *string {
	if value == nil {
		return nil
	}

	normalized := *value
	if trim {
		normalized = strings.TrimSpace(normalized)
	}

	if normalized == "" {
		return nil
	}

	return &normalized
}

func normalizeOptionalPath(value *string) *string {
	if value == nil {
		return nil
	}

	normalized := filepath.Clean(strings.TrimSpace(*value))
	if normalized == "" || normalized == "." {
		return nil
	}

	return &normalized
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}

	return *value
}

func stringPointer(value string) *string {
	return &value
}
