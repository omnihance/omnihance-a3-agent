package services

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
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
	directoryDownloadName  = "Directory download"
)

var (
	ErrBackupInvalid             = errors.New("invalid backup job")
	ErrBackupJobRunning          = errors.New("backup job is currently running")
	ErrBackupNotFound            = errors.New("backup item not found")
	ErrBackupRemoteSQLHost       = errors.New("remote SQL Server backups are not supported")
	ErrBackupNoRunningJob        = errors.New("backup job is not running")
	ErrDirectoryDownloadConflict = errors.New("another directory download job is in progress")
)

const (
	DirectoryDownloadStatusReady      = "ready"
	DirectoryDownloadStatusStarted    = "started"
	DirectoryDownloadStatusInProgress = "in_progress"
	DirectoryDownloadStatusFailed     = "failed"
	DirectoryDownloadStatusCancelled  = "cancelled"
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
	PrepareDirectoryDownload(ctx context.Context, path string, userID *int64) (*DirectoryDownloadResult, error)
	GetDirectoryDownloadStatus(ctx context.Context, runID int64, userID *int64) (*DirectoryDownloadResult, error)
	SearchPaths(query string, kind string) ([]PathSearchResult, error)
	GetSQLServerDefaults() SQLServerBackupDefaults
}

type BackupRunDetails struct {
	Run   db.BackupRun       `json:"run"`
	Files []db.BackupRunFile `json:"files"`
}

type DirectoryDownloadResult struct {
	Status        string
	Message       string
	JobID         int64
	RunID         int64
	FileID        *int64
	ArchivePath   string
	ArchiveReused bool
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

	if payload.Tag == nil {
		payload.Tag = job.Tag
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

	return s.runJobLocked(*job, triggerType, userID)
}

func (s *backupService) runJobLocked(job db.BackupJob, triggerType string, userID *int64) (*db.BackupRun, error) {
	if _, ok := s.runningCancels[job.ID]; ok {
		if triggerType == db.BackupRunTriggerCron {
			return s.createSkippedCronRun(job)
		}

		return nil, ErrBackupJobRunning
	}

	if running, err := s.internalDB.GetRunningBackupRunForJob(job.ID); err != nil {
		return nil, err
	} else if running != nil {
		if triggerType == db.BackupRunTriggerCron {
			return s.createSkippedCronRun(job)
		}

		return nil, ErrBackupJobRunning
	}

	lockPath, err := s.acquireJobLock(job.ID)
	if err != nil {
		if triggerType == db.BackupRunTriggerCron && errors.Is(err, ErrBackupJobRunning) {
			return s.createSkippedCronRun(job)
		}

		return nil, err
	}

	run, err := s.internalDB.CreateBackupRun(job.ID, triggerType, job.Status, userID)
	if err != nil {
		s.releaseJobLock(lockPath)
		return nil, err
	}

	runCtx, cancel := context.WithCancel(context.Background())
	s.runningCancels[job.ID] = cancel
	s.runningRunIDs[job.ID] = run.ID
	s.wg.Add(1)
	go s.executeRun(runCtx, job, *run, lockPath)

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

func (s *backupService) PrepareDirectoryDownload(ctx context.Context, path string, userID *int64) (*DirectoryDownloadResult, error) {
	normalizedPath, err := s.validateDirectoryDownloadPath(path)
	if err != nil {
		return nil, err
	}

	runningResult, err := s.runningDirectoryDownloadResult(normalizedPath)
	if err != nil {
		return nil, err
	}

	if runningResult != nil {
		return runningResult, nil
	}

	sourceFingerprint, err := s.buildDirectoryDownloadFingerprint(ctx, normalizedPath)
	if err != nil {
		return nil, err
	}

	archive, err := s.internalDB.GetDirectoryDownloadArchive(normalizedPath, sourceFingerprint)
	if err == nil {
		if result := s.directoryDownloadArchiveResult(archive); result != nil {
			return result, nil
		}
	} else if !errors.Is(err, constants.ErrNotFound) {
		return nil, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	runningRun, runningJob, err := s.internalDB.GetRunningBackupRunForTag(db.BackupJobTagDirectoryDownload)
	if err != nil {
		return nil, err
	}

	if runningRun != nil && runningJob != nil {
		if runningJob.SourcePath != nil && filepath.Clean(*runningJob.SourcePath) == normalizedPath {
			return directoryDownloadInProgressResult(*runningJob, *runningRun), nil
		}

		return nil, ErrDirectoryDownloadConflict
	}

	job, err := s.internalDB.GetBackupJobByTagAndSourcePath(db.BackupJobTagDirectoryDownload, normalizedPath)
	if err != nil {
		if !errors.Is(err, db.ErrBackupNotFound) {
			return nil, err
		}

		tag := db.BackupJobTagDirectoryDownload
		sourcePath := normalizedPath
		payload := db.BackupJobPayload{
			JobType:              db.BackupJobTypeFile,
			Tag:                  &tag,
			Name:                 directoryDownloadJobName(normalizedPath),
			Status:               db.BackupJobStatusActive,
			DestinationDirectory: s.directoryDownloadsDirectory(),
			SourcePath:           &sourcePath,
		}

		job, err = s.internalDB.CreateBackupJob(payload, userID)
		if err != nil {
			return nil, err
		}
	}

	run, err := s.runJobLocked(*job, db.BackupRunTriggerDirectoryDownload, userID)
	if err != nil {
		return nil, err
	}

	return &DirectoryDownloadResult{
		Status:  DirectoryDownloadStatusStarted,
		Message: "Compressing directory. Keep this page open and do not refresh; the download will start automatically when ready.",
		JobID:   job.ID,
		RunID:   run.ID,
	}, nil
}

func (s *backupService) GetDirectoryDownloadStatus(ctx context.Context, runID int64, userID *int64) (*DirectoryDownloadResult, error) {
	_ = ctx
	_ = userID

	run, err := s.internalDB.GetBackupRun(runID)
	if err != nil {
		return nil, backupNotFoundError(err)
	}

	job, err := s.internalDB.GetBackupJob(run.JobID)
	if err != nil {
		return nil, backupNotFoundError(err)
	}

	if !isDirectoryDownloadJob(*job) {
		return nil, ErrBackupNotFound
	}

	switch run.Status {
	case db.BackupRunStatusRunning:
		return directoryDownloadInProgressResult(*job, *run), nil
	case db.BackupRunStatusFailed:
		return &DirectoryDownloadResult{
			Status:  DirectoryDownloadStatusFailed,
			Message: backupRunMessage(*run, "Directory download failed"),
			JobID:   job.ID,
			RunID:   run.ID,
		}, nil
	case db.BackupRunStatusCancelled:
		return &DirectoryDownloadResult{
			Status:  DirectoryDownloadStatusCancelled,
			Message: backupRunMessage(*run, "Directory download was cancelled"),
			JobID:   job.ID,
			RunID:   run.ID,
		}, nil
	case db.BackupRunStatusSucceeded:
		files, err := s.internalDB.GetBackupRunFiles(run.ID)
		if err != nil {
			return nil, err
		}

		if len(files) == 0 {
			return nil, fmt.Errorf("%w: directory download archive missing", ErrBackupNotFound)
		}

		file := files[0]
		if info, err := s.fileEditor.Stat(file.FilePath); err != nil || info.IsDir() {
			return nil, fmt.Errorf("%w: directory download archive missing", ErrBackupNotFound)
		}

		return directoryDownloadReadyResult(*job, run.ID, file.ID, file.FilePath, false), nil
	default:
		return nil, fmt.Errorf("%w: unsupported directory download run status %s", ErrBackupInvalid, run.Status)
	}
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
	payload.Tag = normalizeOptionalString(payload.Tag, true)
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

	if isDirectoryDownloadJob(job) {
		return s.runDirectoryDownloadBackup(ctx, job, run, output)
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

func (s *backupService) runDirectoryDownloadBackup(ctx context.Context, job db.BackupJob, run db.BackupRun, output *strings.Builder) error {
	normalizedPath, err := s.validateDirectoryDownloadPath(*job.SourcePath)
	if err != nil {
		return err
	}

	sourceFingerprint, err := s.buildDirectoryDownloadFingerprint(ctx, normalizedPath)
	if err != nil {
		return err
	}

	itemName := filepath.Base(normalizedPath)
	if itemName == "." || itemName == string(filepath.Separator) {
		itemName = "directory"
	}

	archivePath, err := uniqueBackupPath(job.DestinationDirectory, itemName, job.ID)
	if err != nil {
		return err
	}

	output.WriteString("Creating directory download archive " + archivePath + "\n")
	if err := createZipArchiveWithExclusions(ctx, normalizedPath, archivePath, "", []string{job.DestinationDirectory}); err != nil {
		_ = s.fileEditor.Remove(archivePath)
		return err
	}

	currentFingerprint, err := s.buildDirectoryDownloadFingerprint(ctx, normalizedPath)
	if err != nil {
		_ = s.fileEditor.Remove(archivePath)
		return err
	}

	if currentFingerprint != sourceFingerprint {
		_ = s.fileEditor.Remove(archivePath)
		return errors.New("directory changed while it was being compressed; please try again")
	}

	archiveInfo, err := s.fileEditor.Stat(archivePath)
	if err != nil {
		return err
	}

	file, err := s.internalDB.CreateBackupRunFile(run.ID, itemName, archivePath, archiveInfo.Size())
	if err != nil {
		return err
	}

	if _, err := s.internalDB.UpsertDirectoryDownloadArchive(db.DirectoryDownloadArchivePayload{
		NormalizedPath:    normalizedPath,
		SourceFingerprint: sourceFingerprint,
		JobID:             job.ID,
		RunID:             run.ID,
		FileID:            file.ID,
		ArchivePath:       archivePath,
		ArchiveSize:       archiveInfo.Size(),
	}); err != nil {
		return err
	}

	_, _ = fmt.Fprintf(output, "Archived %s (%d bytes).\n", filepath.Base(normalizedPath), archiveInfo.Size())

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
	return createZipArchiveWithExclusions(ctx, sourcePath, archivePath, password, nil)
}

func createZipArchiveWithExclusions(ctx context.Context, sourcePath string, archivePath string, password string, excludedPaths []string) error {
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

	excludedAbsolutePaths := cleanDescendantAbsolutePaths(sourcePath, excludedPaths)
	err = filepath.WalkDir(sourcePath, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		if shouldSkipZipPath(path, entry, sourcePath, excludedAbsolutePaths) {
			if entry.IsDir() {
				return filepath.SkipDir
			}

			return nil
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

func cleanDescendantAbsolutePaths(sourcePath string, paths []string) []string {
	results := make([]string, 0, len(paths))
	sourceAbsolutePath, err := filepath.Abs(filepath.Clean(sourcePath))
	if err != nil {
		return results
	}

	for _, path := range paths {
		path = strings.TrimSpace(path)
		if path == "" {
			continue
		}

		absolutePath, err := filepath.Abs(filepath.Clean(path))
		if err != nil {
			continue
		}

		if absolutePath == sourceAbsolutePath || !isPathWithin(sourceAbsolutePath, absolutePath) {
			continue
		}

		results = append(results, absolutePath)
	}

	return results
}

func shouldSkipZipPath(path string, entry os.DirEntry, sourcePath string, excludedAbsolutePaths []string) bool {
	if len(excludedAbsolutePaths) == 0 {
		return false
	}

	if filepath.Clean(path) == filepath.Clean(sourcePath) {
		return false
	}

	absolutePath, err := filepath.Abs(filepath.Clean(path))
	if err != nil {
		return false
	}

	for _, excludedPath := range excludedAbsolutePaths {
		if entry.IsDir() && absolutePath == excludedPath {
			return true
		}

		if isPathWithin(excludedPath, absolutePath) {
			return true
		}
	}

	return false
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

func (s *backupService) validateDirectoryDownloadPath(path string) (string, error) {
	path = filepath.Clean(strings.TrimSpace(path))
	if path == "" || path == "." {
		return "", fmt.Errorf("%w: directory path is required", ErrBackupInvalid)
	}

	info, err := s.fileEditor.Stat(path)
	if err != nil {
		if s.fileEditor.IsNotExist(err) {
			return "", fmt.Errorf("%w: directory path does not exist", ErrBackupInvalid)
		}

		return "", fmt.Errorf("%w: cannot access directory path: %v", ErrBackupInvalid, err)
	}

	if !info.IsDir() {
		return "", fmt.Errorf("%w: path is not a directory", ErrBackupInvalid)
	}

	return path, nil
}

func (s *backupService) buildDirectoryDownloadFingerprint(ctx context.Context, sourcePath string) (string, error) {
	hash := sha256.New()
	sourcePath = filepath.Clean(sourcePath)
	excludedPath := s.directoryDownloadsDirectory()
	excludedAbsolutePaths := cleanDescendantAbsolutePaths(sourcePath, []string{excludedPath})

	err := filepath.WalkDir(sourcePath, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		if shouldSkipZipPath(path, entry, sourcePath, excludedAbsolutePaths) {
			if entry.IsDir() {
				return filepath.SkipDir
			}

			return nil
		}

		if err := ctx.Err(); err != nil {
			return err
		}

		info, err := entry.Info()
		if err != nil {
			return err
		}

		name, err := filepath.Rel(sourcePath, path)
		if err != nil {
			return err
		}

		name = filepath.ToSlash(name)
		_, _ = fmt.Fprintf(
			hash,
			"path=%s\x00dir=%t\x00mode=%s\x00size=%d\x00mod=%d\x00",
			name,
			entry.IsDir(),
			info.Mode().String(),
			info.Size(),
			info.ModTime().UnixNano(),
		)

		if entry.IsDir() {
			return nil
		}

		file, err := os.Open(path)
		if err != nil {
			return err
		}

		_, copyErr := copyWithContext(ctx, hash, file)
		closeErr := file.Close()
		if copyErr != nil {
			return copyErr
		}

		return closeErr
	})
	if err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

func (s *backupService) runningDirectoryDownloadResult(normalizedPath string) (*DirectoryDownloadResult, error) {
	runningRun, runningJob, err := s.internalDB.GetRunningBackupRunForTag(db.BackupJobTagDirectoryDownload)
	if err != nil {
		return nil, err
	}

	if runningRun == nil || runningJob == nil {
		return nil, nil
	}

	if runningJob.SourcePath != nil && filepath.Clean(*runningJob.SourcePath) == normalizedPath {
		return directoryDownloadInProgressResult(*runningJob, *runningRun), nil
	}

	return nil, ErrDirectoryDownloadConflict
}

func (s *backupService) directoryDownloadArchiveResult(archive *db.DirectoryDownloadArchive) *DirectoryDownloadResult {
	if archive == nil {
		return nil
	}

	info, err := s.fileEditor.Stat(archive.ArchivePath)
	if err != nil || info.IsDir() {
		return nil
	}

	return &DirectoryDownloadResult{
		Status:        DirectoryDownloadStatusReady,
		JobID:         archive.JobID,
		RunID:         archive.RunID,
		FileID:        &archive.FileID,
		ArchivePath:   archive.ArchivePath,
		ArchiveReused: true,
	}
}

func directoryDownloadReadyResult(job db.BackupJob, runID int64, fileID int64, archivePath string, archiveReused bool) *DirectoryDownloadResult {
	return &DirectoryDownloadResult{
		Status:        DirectoryDownloadStatusReady,
		JobID:         job.ID,
		RunID:         runID,
		FileID:        &fileID,
		ArchivePath:   archivePath,
		ArchiveReused: archiveReused,
	}
}

func directoryDownloadInProgressResult(job db.BackupJob, run db.BackupRun) *DirectoryDownloadResult {
	return &DirectoryDownloadResult{
		Status:  DirectoryDownloadStatusInProgress,
		Message: "This directory download is already in progress. Keep this page open; the download will start when ready.",
		JobID:   job.ID,
		RunID:   run.ID,
	}
}

func backupRunMessage(run db.BackupRun, fallback string) string {
	if run.ErrorDetails != nil && strings.TrimSpace(*run.ErrorDetails) != "" {
		return *run.ErrorDetails
	}

	if run.Output != nil && strings.TrimSpace(*run.Output) != "" {
		return *run.Output
	}

	return fallback
}

func isDirectoryDownloadJob(job db.BackupJob) bool {
	return job.Tag != nil && *job.Tag == db.BackupJobTagDirectoryDownload
}

func directoryDownloadJobName(path string) string {
	baseName := filepath.Base(filepath.Clean(path))
	if baseName == "." || baseName == string(filepath.Separator) {
		return directoryDownloadName
	}

	return directoryDownloadName + ": " + baseName
}

func (s *backupService) directoryDownloadsDirectory() string {
	if s.cfg == nil || strings.TrimSpace(s.cfg.DirectoryDownloadsDirectory) == "" {
		return ".directory-download"
	}

	return filepath.Clean(strings.TrimSpace(s.cfg.DirectoryDownloadsDirectory))
}

func isPathWithin(parentPath string, childPath string) bool {
	parentPath = filepath.Clean(parentPath)
	childPath = filepath.Clean(childPath)
	if parentPath == childPath {
		return true
	}

	relativePath, err := filepath.Rel(parentPath, childPath)
	if err != nil {
		return false
	}

	return relativePath != "." && relativePath != ".." && !strings.HasPrefix(relativePath, ".."+string(filepath.Separator))
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
