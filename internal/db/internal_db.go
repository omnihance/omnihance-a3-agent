package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/doug-martin/goqu/v9"
	"github.com/omnihance/omnihance-a3-agent/internal/logger"
	"github.com/robfig/cron/v3"
	"modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"
)

const (
	sqliteBusyTimeoutMilliseconds = 5000
	sqliteMaxIdleConnections      = 1
	sqliteMaxOpenConnections      = 1
	sqlitePrimaryResultCodeMask   = 0xff
	walCheckpointSchedule         = "@every 5m"
	walCheckpointTimeout          = 10 * time.Second
)

type InternalDB interface {
	Connect() error
	Close() error
	MigrateUp() error
	MigrateDown() error
	GetSettings() ([]Settings, error)
	GetSetting(key string) (*Settings, error)
	CreateSetting(key string, value string, userID *int64) (*Settings, error)
	UpdateSetting(key string, value string, userID *int64) (*Settings, error)
	SetSetting(key string, value string, userID *int64) error
	SetSettingIfNotExists(key string, value string, userID *int64) error
	DeleteSetting(key string) error
	InsertMetric(metricName string, metricType MetricType, labels map[string]string, value float64, timestamp *int64, unit *string, description *string) error
	InsertMetricSample(seriesID int64, value float64, timestamp *int64) error
	GetSeriesWithLabels() ([]SeriesWithLabels, error)
	GetLatestSamples() ([]LatestSample, error)
	GetMetricSamplesByTimeRange(metricName string, startTime, endTime int64) ([]MetricSampleWithLabels, error)
	GetMetricSamplesByTimeWindow(metricName string, startTime, endTime, stepSeconds int64) ([]MetricSampleWithLabels, error)
	DeleteOldMetrics(retentionDays int) error
	BeginTx() (*goqu.TxDatabase, error)
	CreateFileRevision(tx *goqu.TxDatabase, fileID, originalPath, revisionPath string, previousHash, currentHash string, createdBy int64) (int64, error)
	UpdateFileRevisionStatus(tx *goqu.TxDatabase, revisionID int64, status string, updatedBy int64) error
	UpdateFileRevisionPath(tx *goqu.TxDatabase, revisionID int64, revisionPath string, updatedBy int64) error
	GetFileRevision(revisionID int64) (*FileRevision, error)
	GetLastCompletedFileRevision(fileID string) (*FileRevision, error)
	GetCompletedRevisionCount(fileID string) (int64, error)
	GetRevisionSummary(fileID string) (*RevisionSummary, error)
	CreateSession(userID int64, expiresAt time.Time, userAgent, ipAddress *string) (*Session, error)
	GetSession(sessionID string) (*Session, error)
	UpdateSessionLastAccessed(sessionID string) error
	DeleteSession(sessionID string) error
	DeleteUserSessions(userID int64) error
	DeleteUserSessionsExcept(userID int64, exceptSessionID string) error
	DeleteExpiredSessions() error
	GetUserByID(userID int64) (*User, error)
	GetActiveUserByID(userID int64) (*User, error)
	GetUserByEmail(email string) (*User, error)
	GetUserByIDIncludeDeleted(userID int64) (*User, error)
	GetUserByEmailIncludeDeleted(email string) (*User, error)
	GetUsers() ([]User, error)
	GetUsersPaginated(page, pageSize int, search string) ([]User, int64, error)
	CreateUser(email, password, roles string, createdBy *int64) (*User, error)
	CreateUserWithStatus(email, password, roles, status string, createdBy *int64) (*User, error)
	UpdateUserPassword(userID int64, newPassword string, updatedBy int64) error
	UpdateUserRoles(userID int64, roles string, updatedBy int64) error
	UpdateUserStatus(userID int64, status string, updatedBy int64) error
	DeleteUser(userID int64, deletedBy int64) error
	GetAdminUserCount() (int64, error)
	SetDefaultSettings() error
	BulkReplaceMonsterClientData(data []MonsterClientData) error
	GetAllMonsterClientData(search string) ([]MonsterClientData, error)
	GetMonsterClientDataCount() (int64, error)
	BulkReplaceMapClientData(data []MapClientData) error
	GetAllMapClientData(search string) ([]MapClientData, error)
	GetMapClientDataCount() (int64, error)
	BulkReplaceItemClientData(itemType ItemClientDataType, data []ItemClientData) error
	GetAllItemClientData(search string) ([]ItemClientData, error)
	GetItemClientDataCounts() (ItemClientDataCounts, error)
	GetServerProcesses() ([]ServerProcess, error)
	GetServerProcess(id int64) (*ServerProcess, error)
	GetServerProcessByPath(path string) (*ServerProcess, error)
	CreateServerProcess(name, path string, port *int, sequenceOrder int) (*ServerProcess, error)
	UpdateServerProcess(id int64, name, path string, port *int) error
	DeleteServerProcess(id int64) error
	ReorderServerProcesses(updates []ReorderUpdate) error
	GetMaxSequenceOrder() (int, error)
	UpdateProcessStartTime(id int64, startTime time.Time) error
	UpdateProcessEndTime(id int64, endTime time.Time) error
	GetDirectoryShortcuts(userID int64) ([]DirectoryShortcut, error)
	GetDirectoryShortcut(id int64) (*DirectoryShortcut, error)
	GetDirectoryShortcutCount(userID int64) (int64, error)
	GetDirectoryShortcutByNormalizedPath(userID int64, normalizedPath string) (*DirectoryShortcut, error)
	CreateDirectoryShortcut(userID int64, name, path string) (*DirectoryShortcut, error)
	DeleteDirectoryShortcut(id int64, userID int64) error
	GetBackupJobs() ([]BackupJob, error)
	GetSchedulableBackupJobs() ([]BackupJob, error)
	GetBackupJob(id int64) (*BackupJob, error)
	CreateBackupJob(payload BackupJobPayload, userID *int64) (*BackupJob, error)
	UpdateBackupJob(id int64, payload BackupJobPayload, userID *int64) (*BackupJob, error)
	UpdateBackupJobStatus(id int64, status string, userID *int64) error
	DeleteBackupJob(id int64, userID *int64) error
	CreateBackupRun(jobID int64, triggerType string, previousJobStatus string, userID *int64) (*BackupRun, error)
	CreateSkippedBackupRun(jobID int64, triggerType string, previousJobStatus string, output string, userID *int64) (*BackupRun, error)
	GetBackupRuns(jobID int64, page int, pageSize int) ([]BackupRun, int64, error)
	GetBackupRun(id int64) (*BackupRun, error)
	GetRunningBackupRunForJob(jobID int64) (*BackupRun, error)
	FinishBackupRun(runID int64, jobID int64, runStatus string, jobStatus string, output *string, errorDetails *string) error
	MarkBackupRunCancelRequested(runID int64) error
	CreateBackupRunFile(runID int64, itemName string, filePath string, fileSize int64) (*BackupRunFile, error)
	GetBackupRunFiles(runID int64) ([]BackupRunFile, error)
	GetBackupRunFile(id int64) (*BackupRunFile, error)
	MarkOrphanedBackupRunsFailed() error
	CreateServerViewSyncRun(userID *int64) (*ServerViewSyncRun, error)
	FinishServerViewSyncRun(runID int64, status string, warningCount int, errorDetails *string) error
	GetLatestServerViewSyncRun() (*ServerViewSyncRun, error)
	GetRunningServerViewSyncRun() (*ServerViewSyncRun, error)
	MarkOrphanedServerViewSyncRunsFailed() error
	AddServerViewSyncWarning(runID int64, source string, message string) error
	GetServerViewSyncWarnings(runID int64) ([]ServerViewSyncWarning, error)
	ReplaceServerViewSvrInfoRows(serverType string, rows []ServerViewSvrInfoRow) error
	GetServerViewSvrInfoRows() ([]ServerViewSvrInfoRow, error)
	ReplaceServerViewMapZones(serverType string, rows []ServerViewMapZone) error
	GetServerViewMapZones(serverType string) ([]ServerViewMapZone, error)
	ReplaceServerViewSpawnRowsForMap(mapID int64, rows []ServerViewSpawnRow) error
	GetServerViewSpawnRows() ([]ServerViewSpawnRow, error)
	ReplaceServerViewDropRowsForNPC(npcID int64, rows []ServerViewDropRow) error
	GetServerViewDropRows() ([]ServerViewDropRow, error)
	ReplaceServerViewShopRowsForNPC(npcID int64, rows []ServerViewShopRow) error
	GetServerViewShopRows() ([]ServerViewShopRow, error)
	ReplaceServerViewGameMasterRows(rows []ServerViewGameMasterRow) error
	GetServerViewGameMasterRows() ([]ServerViewGameMasterRow, error)
}

type sqliteInternalDB struct {
	dsn    string
	db     *sql.DB
	goqu   *goqu.Database
	logger logger.Logger
	cron   *cron.Cron
}

func NewSQLiteDB(dsn string, log logger.Logger) InternalDB {
	return &sqliteInternalDB{dsn: dsn, logger: log}
}

func (s *sqliteInternalDB) Connect() error {
	db, err := sql.Open("sqlite", s.dsn)
	if err != nil {
		return err
	}

	db.SetMaxOpenConns(sqliteMaxOpenConnections)
	db.SetMaxIdleConns(sqliteMaxIdleConnections)

	pragmas := []string{
		`PRAGMA journal_mode=WAL;`,
		`PRAGMA synchronous=NORMAL;`,
		fmt.Sprintf(`PRAGMA busy_timeout=%d;`, sqliteBusyTimeoutMilliseconds),
		`PRAGMA foreign_keys=ON;`,
		`PRAGMA cache_size=-64000;`,
		`PRAGMA temp_store=MEMORY;`,
		`PRAGMA mmap_size=30000000000;`,
	}

	for _, pragma := range pragmas {
		if _, err := db.Exec(pragma); err != nil {
			if closeErr := db.Close(); closeErr != nil {
				s.logger.Error("Failed to close database during cleanup", logger.Field{Key: "error", Value: closeErr})
			}

			return fmt.Errorf("failed to execute pragma %s: %w", pragma, err)
		}
	}

	if err := db.Ping(); err != nil {
		if closeErr := db.Close(); closeErr != nil {
			s.logger.Error("Failed to close database during cleanup", logger.Field{Key: "error", Value: closeErr})
		}

		return fmt.Errorf("sqlite ping failed: %w", err)
	}

	s.db = db
	s.goqu = goqu.New("sqlite", db)

	s.runWALCheckpoint()

	s.cron = cron.New(cron.WithSeconds())
	_, err = s.cron.AddFunc(walCheckpointSchedule, s.runWALCheckpoint)
	if err != nil {
		if closeErr := db.Close(); closeErr != nil {
			s.logger.Error("Failed to close database during cleanup", logger.Field{Key: "error", Value: closeErr})
		}

		return fmt.Errorf("failed to schedule WAL checkpoint: %w", err)
	}

	s.cron.Start()

	s.logger.Info("WAL checkpoint scheduled to run every 5 minutes")

	return nil
}

func (s *sqliteInternalDB) Close() error {
	if s.cron != nil {
		ctx := s.cron.Stop()
		<-ctx.Done()
	}

	if s.db != nil {
		return s.db.Close()
	}

	return nil
}

func (s *sqliteInternalDB) MigrateUp() error {
	_, err := s.db.Exec(`CREATE TABLE IF NOT EXISTS migrations (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        name TEXT NOT NULL UNIQUE,
        applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
    );`)
	if err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	if err := s.migrate001UsersTable(); err != nil {
		return err
	}

	if err := s.migrate002SessionsTable(); err != nil {
		return err
	}

	if err := s.migrate003SettingsTable(); err != nil {
		return err
	}

	if err := s.migrate004FileRevisionsTable(); err != nil {
		return err
	}

	if err := s.migrate005MetricsTables(); err != nil {
		return err
	}

	if err := s.migrate006MonsterClientDataTable(); err != nil {
		return err
	}

	if err := s.migrate007MapClientDataTable(); err != nil {
		return err
	}

	if err := s.migrate008ItemClientDataTable(); err != nil {
		return err
	}

	if err := s.migrate009ServerProcessesTable(); err != nil {
		return err
	}

	if err := s.migrate010DirectoryShortcutsTable(); err != nil {
		return err
	}

	if err := s.migrate011ItemClientDataType(); err != nil {
		return err
	}

	if err := s.migrate012BackupJobsTables(); err != nil {
		return err
	}

	if err := s.migrate013ServerViewTables(); err != nil {
		return err
	}

	return nil
}

func (s *sqliteInternalDB) MigrateDown() error {
	if err := s.rollback013ServerViewTables(); err != nil {
		return err
	}

	if err := s.rollback012BackupJobsTables(); err != nil {
		return err
	}

	if err := s.rollback011ItemClientDataType(); err != nil {
		return err
	}

	if err := s.rollback010DirectoryShortcutsTable(); err != nil {
		return err
	}

	if err := s.rollback009ServerProcessesTable(); err != nil {
		return err
	}

	if err := s.rollback008ItemClientDataTable(); err != nil {
		return err
	}

	if err := s.rollback007MapClientDataTable(); err != nil {
		return err
	}

	if err := s.rollback006MonsterClientDataTable(); err != nil {
		return err
	}

	if err := s.rollback005MetricsTables(); err != nil {
		return err
	}

	if err := s.rollback004FileRevisionsTable(); err != nil {
		return err
	}

	if err := s.rollback003SettingsTable(); err != nil {
		return err
	}

	if err := s.rollback002SessionsTable(); err != nil {
		return err
	}

	if err := s.rollback001UsersTable(); err != nil {
		return err
	}

	return nil
}

func (s *sqliteInternalDB) BeginTx() (*goqu.TxDatabase, error) {
	tx, err := s.db.Begin()
	if err != nil {
		s.logger.Error(
			"failed to begin transaction",
			logger.Field{Key: "error", Value: err},
		)
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}

	return goqu.NewTx("sqlite", tx), nil
}

func (s *sqliteInternalDB) isMigrationApplied(migName string) (bool, error) {
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM migrations WHERE name = ?", migName).Scan(&count)
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

func (s *sqliteInternalDB) markMigrationApplied(migName string) error {
	_, err := s.db.Exec("INSERT INTO migrations (name) VALUES (?)", migName)
	return err
}

func (s *sqliteInternalDB) markMigrationRolledBack(migName string) error {
	_, err := s.db.Exec("DELETE FROM migrations WHERE name = ?", migName)
	return err
}

func (s *sqliteInternalDB) runWALCheckpoint() {
	if s.db == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), walCheckpointTimeout)
	defer cancel()

	_, err := s.db.ExecContext(ctx, `PRAGMA wal_checkpoint(TRUNCATE);`)
	if err != nil {
		if isSQLiteBusyOrLockedError(err) || errors.Is(err, context.DeadlineExceeded) {
			s.logger.Debug(
				"WAL checkpoint skipped because database is busy",
				logger.Field{Key: "error", Value: err},
			)
			return
		}

		s.logger.Error(
			"failed to run WAL checkpoint",
			logger.Field{Key: "error", Value: err},
		)
		return
	}

	s.logger.Debug("WAL checkpoint completed successfully")
}

func isSQLiteBusyOrLockedError(err error) bool {
	var sqliteErr *sqlite.Error
	if !errors.As(err, &sqliteErr) {
		return false
	}

	switch sqliteErr.Code() & sqlitePrimaryResultCodeMask {
	case sqlite3.SQLITE_BUSY, sqlite3.SQLITE_LOCKED:
		return true
	default:
		return false
	}
}

func (s *sqliteInternalDB) migrate001UsersTable() error {
	const migName = "001_users_table"

	applied, err := s.isMigrationApplied(migName)
	if err != nil {
		s.logger.Error(
			"failed to check migration status",
			logger.Field{Key: "migration", Value: migName},
			logger.Field{Key: "error", Value: err},
		)
		return fmt.Errorf("failed to check migration status for %s: %w", migName, err)
	}

	if applied {
		return nil
	}

	s.logger.Info("Applying migration", logger.Field{Key: "migration", Value: migName})

	migrationSQL := `
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
		email TEXT NOT NULL UNIQUE,
		password TEXT NOT NULL,
		roles TEXT NOT NULL DEFAULT 'viewer',
		status TEXT NOT NULL DEFAULT 'pending',
		created_by INTEGER REFERENCES users(id) ON DELETE SET NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_by INTEGER REFERENCES users(id) ON DELETE SET NULL,
		updated_at TIMESTAMP,
		deleted_by INTEGER REFERENCES users(id) ON DELETE SET NULL,
		deleted_at TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_users_email ON users (email);

	CREATE INDEX IF NOT EXISTS idx_users_is_deleted ON users (is_deleted);

	CREATE INDEX IF NOT EXISTS idx_users_roles ON users (roles);

	CREATE INDEX IF NOT EXISTS idx_users_status ON users (status);
	`
	_, err = s.db.Exec(migrationSQL)
	if err != nil {
		return fmt.Errorf("failed to create users table: %w", err)
	}

	if err := s.markMigrationApplied(migName); err != nil {
		s.logger.Error(
			"failed to mark migration as applied",
			logger.Field{Key: "migration", Value: migName},
			logger.Field{Key: "error", Value: err},
		)
		return fmt.Errorf("failed to mark migration as applied: %w", err)
	}

	return nil
}

func (s *sqliteInternalDB) rollback001UsersTable() error {
	const migName = "001_users_table"

	applied, err := s.isMigrationApplied(migName)
	if err != nil {
		s.logger.Error(
			"failed to check migration status",
			logger.Field{Key: "migration", Value: migName},
			logger.Field{Key: "error", Value: err},
		)
	}

	if !applied {
		return nil
	}

	s.logger.Info("Rolling back migration", logger.Field{Key: "migration", Value: migName})

	migrationSQL := `
	DROP TABLE IF EXISTS users;
	`
	_, err = s.db.Exec(migrationSQL)
	if err != nil {
		return fmt.Errorf("failed to rollback users table: %w", err)
	}

	if err := s.markMigrationRolledBack(migName); err != nil {
		s.logger.Error(
			"failed to mark migration as rolled back",
			logger.Field{Key: "migration", Value: migName},
			logger.Field{Key: "error", Value: err},
		)
		return fmt.Errorf("failed to mark migration as rolled back: %w", err)
	}

	return nil
}

func (s *sqliteInternalDB) migrate002SessionsTable() error {
	const migName = "002_sessions_table"

	applied, err := s.isMigrationApplied(migName)
	if err != nil {
		s.logger.Error(
			"failed to check migration status",
			logger.Field{Key: "migration", Value: migName},
			logger.Field{Key: "error", Value: err},
		)
		return fmt.Errorf("failed to check migration status for %s: %w", migName, err)
	}

	if applied {
		return nil
	}

	s.logger.Info("Applying migration", logger.Field{Key: "migration", Value: migName})

	migrationSQL := `
	CREATE TABLE IF NOT EXISTS sessions (
		session_id TEXT PRIMARY KEY,
		user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		expires_at TIMESTAMP NOT NULL,
		last_accessed_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		user_agent TEXT,
		ip_address TEXT
	);

	CREATE INDEX IF NOT EXISTS idx_sessions_user_id ON sessions (user_id);

	CREATE INDEX IF NOT EXISTS idx_sessions_expires_at ON sessions (expires_at);
	`
	_, err = s.db.Exec(migrationSQL)
	if err != nil {
		return fmt.Errorf("failed to create sessions table: %w", err)
	}

	if err := s.markMigrationApplied(migName); err != nil {
		s.logger.Error(
			"failed to mark migration as applied",
			logger.Field{Key: "migration", Value: migName},
			logger.Field{Key: "error", Value: err},
		)
		return fmt.Errorf("failed to mark migration as applied: %w", err)
	}

	return nil
}

func (s *sqliteInternalDB) rollback002SessionsTable() error {
	const migName = "002_sessions_table"

	applied, err := s.isMigrationApplied(migName)
	if err != nil {
		s.logger.Error(
			"failed to check migration status",
			logger.Field{Key: "migration", Value: migName},
			logger.Field{Key: "error", Value: err},
		)
	}

	if !applied {
		return nil
	}

	s.logger.Info("Rolling back migration", logger.Field{Key: "migration", Value: migName})

	migrationSQL := `
	DROP TABLE IF EXISTS sessions;
	`
	_, err = s.db.Exec(migrationSQL)
	if err != nil {
		return fmt.Errorf("failed to rollback sessions table: %w", err)
	}

	if err := s.markMigrationRolledBack(migName); err != nil {
		s.logger.Error(
			"failed to mark migration as rolled back",
			logger.Field{Key: "migration", Value: migName},
			logger.Field{Key: "error", Value: err},
		)
		return fmt.Errorf("failed to mark migration as rolled back: %w", err)
	}

	return nil
}

func (s *sqliteInternalDB) migrate003SettingsTable() error {
	const migName = "003_settings_table"

	applied, err := s.isMigrationApplied(migName)
	if err != nil {
		s.logger.Error(
			"failed to check migration status",
			logger.Field{Key: "migration", Value: migName},
			logger.Field{Key: "error", Value: err},
		)
		return fmt.Errorf("failed to check migration status for %s: %w", migName, err)
	}

	if applied {
		return nil
	}

	s.logger.Info("Applying migration", logger.Field{Key: "migration", Value: migName})

	migrationSQL := `
	CREATE TABLE IF NOT EXISTS settings (
		key TEXT PRIMARY KEY,
		value TEXT NOT NULL,
		created_by INTEGER REFERENCES users(id) ON DELETE SET NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_by INTEGER REFERENCES users(id) ON DELETE SET NULL,
		updated_at TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_settings_created_by ON settings (created_by);

	CREATE INDEX IF NOT EXISTS idx_settings_updated_by ON settings (updated_by);
	`
	_, err = s.db.Exec(migrationSQL)
	if err != nil {
		return fmt.Errorf("failed to create settings table: %w", err)
	}

	if err := s.markMigrationApplied(migName); err != nil {
		s.logger.Error(
			"failed to mark migration as applied",
			logger.Field{Key: "migration", Value: migName},
			logger.Field{Key: "error", Value: err},
		)
		return fmt.Errorf("failed to mark migration as applied: %w", err)
	}

	return nil
}

func (s *sqliteInternalDB) rollback003SettingsTable() error {
	const migName = "003_settings_table"

	applied, err := s.isMigrationApplied(migName)
	if err != nil {
		s.logger.Error(
			"failed to check migration status",
			logger.Field{Key: "migration", Value: migName},
			logger.Field{Key: "error", Value: err},
		)
	}

	if !applied {
		return nil
	}

	s.logger.Info("Rolling back migration", logger.Field{Key: "migration", Value: migName})

	migrationSQL := `
	DROP TABLE IF EXISTS settings;
	`
	_, err = s.db.Exec(migrationSQL)
	if err != nil {
		return fmt.Errorf("failed to rollback settings table: %w", err)
	}

	if err := s.markMigrationRolledBack(migName); err != nil {
		s.logger.Error(
			"failed to mark migration as rolled back",
			logger.Field{Key: "migration", Value: migName},
			logger.Field{Key: "error", Value: err},
		)
		return fmt.Errorf("failed to mark migration as rolled back: %w", err)
	}

	return nil
}

func (s *sqliteInternalDB) migrate004FileRevisionsTable() error {
	const migName = "004_file_revisions_table"

	applied, err := s.isMigrationApplied(migName)
	if err != nil {
		s.logger.Error(
			"failed to check migration status",
			logger.Field{Key: "migration", Value: migName},
			logger.Field{Key: "error", Value: err},
		)
		return fmt.Errorf("failed to check migration status for %s: %w", migName, err)
	}

	if applied {
		return nil
	}

	s.logger.Info("Applying migration", logger.Field{Key: "migration", Value: migName})

	migrationSQL := `
	CREATE TABLE IF NOT EXISTS file_revisions (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		file_id TEXT NOT NULL,
		original_path TEXT NOT NULL,
		revision_path TEXT NOT NULL,
		previous_hash TEXT NOT NULL,
		current_hash TEXT NOT NULL,
		created_by INTEGER NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_by INTEGER REFERENCES users(id) ON DELETE SET NULL,
		updated_at TIMESTAMP,
		status TEXT NOT NULL DEFAULT 'draft'
	);

	CREATE INDEX IF NOT EXISTS idx_file_revisions_file_id ON file_revisions (file_id);

	CREATE INDEX IF NOT EXISTS idx_file_revisions_status ON file_revisions (status);

	CREATE INDEX IF NOT EXISTS idx_file_revisions_created_by ON file_revisions (created_by);

	CREATE INDEX IF NOT EXISTS idx_file_revisions_created_at ON file_revisions (created_at);
	`
	_, err = s.db.Exec(migrationSQL)
	if err != nil {
		return fmt.Errorf("failed to create file_revisions table: %w", err)
	}

	if err := s.markMigrationApplied(migName); err != nil {
		s.logger.Error(
			"failed to mark migration as applied",
			logger.Field{Key: "migration", Value: migName},
			logger.Field{Key: "error", Value: err},
		)
		return fmt.Errorf("failed to mark migration as applied: %w", err)
	}

	return nil
}

func (s *sqliteInternalDB) rollback004FileRevisionsTable() error {
	const migName = "004_file_revisions_table"

	applied, err := s.isMigrationApplied(migName)
	if err != nil {
		s.logger.Error(
			"failed to check migration status",
			logger.Field{Key: "migration", Value: migName},
			logger.Field{Key: "error", Value: err},
		)
	}

	if !applied {
		return nil
	}

	s.logger.Info("Rolling back migration", logger.Field{Key: "migration", Value: migName})

	migrationSQL := `
	DROP TABLE IF EXISTS file_revisions;
	`
	_, err = s.db.Exec(migrationSQL)
	if err != nil {
		return fmt.Errorf("failed to rollback file_revisions table: %w", err)
	}

	if err := s.markMigrationRolledBack(migName); err != nil {
		s.logger.Error(
			"failed to mark migration as rolled back",
			logger.Field{Key: "migration", Value: migName},
			logger.Field{Key: "error", Value: err},
		)
		return fmt.Errorf("failed to mark migration as rolled back: %w", err)
	}

	return nil
}

func (s *sqliteInternalDB) migrate005MetricsTables() error {
	const migName = "005_metrics_tables"

	applied, err := s.isMigrationApplied(migName)
	if err != nil {
		s.logger.Error(
			"failed to check migration status",
			logger.Field{Key: "migration", Value: migName},
			logger.Field{Key: "error", Value: err},
		)
		return fmt.Errorf("failed to check migration status for %s: %w", migName, err)
	}

	if applied {
		return nil
	}

	s.logger.Info("applying migration", logger.Field{Key: "migration", Value: migName})

	migrationSQL := `
		CREATE TABLE IF NOT EXISTS metric_names (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL UNIQUE,
			type TEXT NOT NULL CHECK(type IN ('counter', 'gauge', 'histogram', 'summary')),
			unit TEXT,
			description TEXT,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		);

		CREATE UNIQUE INDEX IF NOT EXISTS idx_metric_names_name ON metric_names(name);

		CREATE TABLE IF NOT EXISTS labels (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			key TEXT NOT NULL,
			value TEXT NOT NULL,
			UNIQUE(key, value)
		);

		CREATE INDEX IF NOT EXISTS idx_labels_key ON labels(key);
		CREATE INDEX IF NOT EXISTS idx_labels_value ON labels(value);

		CREATE TABLE IF NOT EXISTS metric_series (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			metric_id INTEGER NOT NULL,
			label_hash TEXT NOT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			last_updated TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (metric_id) REFERENCES metric_names(id) ON DELETE CASCADE,
			UNIQUE(metric_id, label_hash)
		);

		CREATE INDEX IF NOT EXISTS idx_metric_series_metric ON metric_series(metric_id);
		CREATE INDEX IF NOT EXISTS idx_metric_series_hash ON metric_series(label_hash);
		CREATE INDEX IF NOT EXISTS idx_metric_series_updated ON metric_series(last_updated);

		CREATE TABLE IF NOT EXISTS series_labels (
			series_id INTEGER NOT NULL,
			label_id INTEGER NOT NULL,
			PRIMARY KEY (series_id, label_id),
			FOREIGN KEY (series_id) REFERENCES metric_series(id) ON DELETE CASCADE,
			FOREIGN KEY (label_id) REFERENCES labels(id) ON DELETE CASCADE
		);

		CREATE INDEX IF NOT EXISTS idx_series_labels_series ON series_labels(series_id);
		CREATE INDEX IF NOT EXISTS idx_series_labels_label ON series_labels(label_id);

		CREATE TABLE IF NOT EXISTS metric_samples (
			series_id INTEGER NOT NULL,
			timestamp INTEGER NOT NULL,
			value REAL NOT NULL,
			PRIMARY KEY (series_id, timestamp),
			FOREIGN KEY (series_id) REFERENCES metric_series(id) ON DELETE CASCADE
		) WITHOUT ROWID;

		CREATE INDEX IF NOT EXISTS idx_samples_time ON metric_samples(timestamp, series_id);
		CREATE INDEX IF NOT EXISTS idx_samples_series_time ON metric_samples(series_id, timestamp DESC);

		CREATE TRIGGER IF NOT EXISTS trg_update_series_timestamp
		AFTER INSERT ON metric_samples
		BEGIN
			UPDATE metric_series 
			SET last_updated = datetime(NEW.timestamp, 'unixepoch') 
			WHERE id = NEW.series_id 
			  AND last_updated < datetime(NEW.timestamp, 'unixepoch');
		END;
	`

	_, err = s.db.Exec(migrationSQL)
	if err != nil {
		return fmt.Errorf("failed to create metrics tables: %w", err)
	}

	if err := s.markMigrationApplied(migName); err != nil {
		s.logger.Error(
			"failed to mark migration as applied",
			logger.Field{Key: "migration", Value: migName},
			logger.Field{Key: "error", Value: err},
		)
		return fmt.Errorf("failed to mark migration %s as applied: %w", migName, err)
	}

	return nil
}

func (s *sqliteInternalDB) rollback005MetricsTables() error {
	const migName = "005_metrics_tables"

	applied, err := s.isMigrationApplied(migName)
	if err != nil {
		s.logger.Error(
			"failed to check migration status",
			logger.Field{Key: "migration", Value: migName},
			logger.Field{Key: "error", Value: err},
		)
		return fmt.Errorf("failed to check migration status for %s: %w", migName, err)
	}

	if !applied {
		return nil
	}

	s.logger.Info("rolling back migration", logger.Field{Key: "migration", Value: migName})

	rollbackSQL := `
		DROP TRIGGER IF EXISTS trg_update_series_timestamp;
		DROP TABLE IF EXISTS metric_samples;
		DROP TABLE IF EXISTS series_labels;
		DROP TABLE IF EXISTS metric_series;
		DROP TABLE IF EXISTS labels;
		DROP TABLE IF EXISTS metric_names;
	`

	_, err = s.db.Exec(rollbackSQL)
	if err != nil {
		return fmt.Errorf("failed to drop metrics tables: %w", err)
	}

	if err := s.markMigrationRolledBack(migName); err != nil {
		s.logger.Error(
			"failed to mark migration as rolled back",
			logger.Field{Key: "migration", Value: migName},
			logger.Field{Key: "error", Value: err},
		)
		return fmt.Errorf("failed to mark migration %s as rolled back: %w", migName, err)
	}

	return nil
}

func (s *sqliteInternalDB) migrate006MonsterClientDataTable() error {
	const migName = "006_monster_client_data_table"

	applied, err := s.isMigrationApplied(migName)
	if err != nil {
		s.logger.Error(
			"failed to check migration status",
			logger.Field{Key: "migration", Value: migName},
			logger.Field{Key: "error", Value: err},
		)
		return fmt.Errorf("failed to check migration status for %s: %w", migName, err)
	}

	if applied {
		return nil
	}

	s.logger.Info("Applying migration", logger.Field{Key: "migration", Value: migName})

	migrationSQL := `
	CREATE TABLE IF NOT EXISTS monster_client_data (
		id INTEGER PRIMARY KEY,
		name TEXT NOT NULL,
		created_by INTEGER REFERENCES users(id) ON DELETE SET NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_by INTEGER REFERENCES users(id) ON DELETE SET NULL,
		updated_at TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_monster_client_data_name ON monster_client_data (name);

	CREATE INDEX IF NOT EXISTS idx_monster_client_data_created_by ON monster_client_data (created_by);

	CREATE INDEX IF NOT EXISTS idx_monster_client_data_updated_by ON monster_client_data (updated_by);
	`
	_, err = s.db.Exec(migrationSQL)
	if err != nil {
		return fmt.Errorf("failed to create monster_client_data table: %w", err)
	}

	if err := s.markMigrationApplied(migName); err != nil {
		s.logger.Error(
			"failed to mark migration as applied",
			logger.Field{Key: "migration", Value: migName},
			logger.Field{Key: "error", Value: err},
		)
		return fmt.Errorf("failed to mark migration as applied: %w", err)
	}

	return nil
}

func (s *sqliteInternalDB) rollback006MonsterClientDataTable() error {
	const migName = "006_monster_client_data_table"

	applied, err := s.isMigrationApplied(migName)
	if err != nil {
		s.logger.Error(
			"failed to check migration status",
			logger.Field{Key: "migration", Value: migName},
			logger.Field{Key: "error", Value: err},
		)
		return fmt.Errorf("failed to check migration status for %s: %w", migName, err)
	}

	if !applied {
		return nil
	}

	s.logger.Info("Rolling back migration", logger.Field{Key: "migration", Value: migName})

	migrationSQL := `
	DROP TABLE IF EXISTS monster_client_data;
	`
	_, err = s.db.Exec(migrationSQL)
	if err != nil {
		return fmt.Errorf("failed to rollback monster_client_data table: %w", err)
	}

	if err := s.markMigrationRolledBack(migName); err != nil {
		s.logger.Error(
			"failed to mark migration as rolled back",
			logger.Field{Key: "migration", Value: migName},
			logger.Field{Key: "error", Value: err},
		)
		return fmt.Errorf("failed to mark migration as rolled back: %w", err)
	}

	return nil
}

func (s *sqliteInternalDB) migrate007MapClientDataTable() error {
	const migName = "007_map_client_data_table"

	applied, err := s.isMigrationApplied(migName)
	if err != nil {
		s.logger.Error(
			"failed to check migration status",
			logger.Field{Key: "migration", Value: migName},
			logger.Field{Key: "error", Value: err},
		)
		return fmt.Errorf("failed to check migration status for %s: %w", migName, err)
	}

	if applied {
		return nil
	}

	s.logger.Info("Applying migration", logger.Field{Key: "migration", Value: migName})

	migrationSQL := `
	CREATE TABLE IF NOT EXISTS map_client_data (
		id INTEGER PRIMARY KEY,
		name TEXT NOT NULL,
		created_by INTEGER REFERENCES users(id) ON DELETE SET NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_by INTEGER REFERENCES users(id) ON DELETE SET NULL,
		updated_at TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_map_client_data_name ON map_client_data (name);

	CREATE INDEX IF NOT EXISTS idx_map_client_data_created_by ON map_client_data (created_by);

	CREATE INDEX IF NOT EXISTS idx_map_client_data_updated_by ON map_client_data (updated_by);
	`
	_, err = s.db.Exec(migrationSQL)
	if err != nil {
		return fmt.Errorf("failed to create map_client_data table: %w", err)
	}

	if err := s.markMigrationApplied(migName); err != nil {
		s.logger.Error(
			"failed to mark migration as applied",
			logger.Field{Key: "migration", Value: migName},
			logger.Field{Key: "error", Value: err},
		)
		return fmt.Errorf("failed to mark migration as applied: %w", err)
	}

	return nil
}

func (s *sqliteInternalDB) rollback007MapClientDataTable() error {
	const migName = "007_map_client_data_table"

	applied, err := s.isMigrationApplied(migName)
	if err != nil {
		s.logger.Error(
			"failed to check migration status",
			logger.Field{Key: "migration", Value: migName},
			logger.Field{Key: "error", Value: err},
		)
		return fmt.Errorf("failed to check migration status for %s: %w", migName, err)
	}

	if !applied {
		return nil
	}

	s.logger.Info("Rolling back migration", logger.Field{Key: "migration", Value: migName})

	migrationSQL := `
	DROP TABLE IF EXISTS map_client_data;
	`
	_, err = s.db.Exec(migrationSQL)
	if err != nil {
		return fmt.Errorf("failed to rollback map_client_data table: %w", err)
	}

	if err := s.markMigrationRolledBack(migName); err != nil {
		s.logger.Error(
			"failed to mark migration as rolled back",
			logger.Field{Key: "migration", Value: migName},
			logger.Field{Key: "error", Value: err},
		)
		return fmt.Errorf("failed to mark migration as rolled back: %w", err)
	}

	return nil
}

func (s *sqliteInternalDB) migrate008ItemClientDataTable() error {
	const migName = "008_item_client_data_table"

	applied, err := s.isMigrationApplied(migName)
	if err != nil {
		s.logger.Error(
			"failed to check migration status",
			logger.Field{Key: "migration", Value: migName},
			logger.Field{Key: "error", Value: err},
		)
		return fmt.Errorf("failed to check migration status for %s: %w", migName, err)
	}

	if applied {
		return nil
	}

	s.logger.Info("Applying migration", logger.Field{Key: "migration", Value: migName})

	migrationSQL := `
	CREATE TABLE IF NOT EXISTS item_client_data (
		id INTEGER PRIMARY KEY,
		name TEXT NOT NULL,
		created_by INTEGER REFERENCES users(id) ON DELETE SET NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_by INTEGER REFERENCES users(id) ON DELETE SET NULL,
		updated_at TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_item_client_data_name ON item_client_data (name);

	CREATE INDEX IF NOT EXISTS idx_item_client_data_created_by ON item_client_data (created_by);

	CREATE INDEX IF NOT EXISTS idx_item_client_data_updated_by ON item_client_data (updated_by);
	`
	_, err = s.db.Exec(migrationSQL)
	if err != nil {
		return fmt.Errorf("failed to create item_client_data table: %w", err)
	}

	if err := s.markMigrationApplied(migName); err != nil {
		s.logger.Error(
			"failed to mark migration as applied",
			logger.Field{Key: "migration", Value: migName},
			logger.Field{Key: "error", Value: err},
		)
		return fmt.Errorf("failed to mark migration as applied: %w", err)
	}

	return nil
}

func (s *sqliteInternalDB) rollback008ItemClientDataTable() error {
	const migName = "008_item_client_data_table"

	applied, err := s.isMigrationApplied(migName)
	if err != nil {
		s.logger.Error(
			"failed to check migration status",
			logger.Field{Key: "migration", Value: migName},
			logger.Field{Key: "error", Value: err},
		)
		return fmt.Errorf("failed to check migration status for %s: %w", migName, err)
	}

	if !applied {
		return nil
	}

	s.logger.Info("Rolling back migration", logger.Field{Key: "migration", Value: migName})

	migrationSQL := `
	DROP TABLE IF EXISTS item_client_data;
	`
	_, err = s.db.Exec(migrationSQL)
	if err != nil {
		return fmt.Errorf("failed to rollback item_client_data table: %w", err)
	}

	if err := s.markMigrationRolledBack(migName); err != nil {
		s.logger.Error(
			"failed to mark migration as rolled back",
			logger.Field{Key: "migration", Value: migName},
			logger.Field{Key: "error", Value: err},
		)
		return fmt.Errorf("failed to mark migration as rolled back: %w", err)
	}

	return nil
}

func (s *sqliteInternalDB) migrate009ServerProcessesTable() error {
	const migName = "009_server_processes_table"

	applied, err := s.isMigrationApplied(migName)
	if err != nil {
		s.logger.Error(
			"failed to check migration status",
			logger.Field{Key: "migration", Value: migName},
			logger.Field{Key: "error", Value: err},
		)
		return fmt.Errorf("failed to check migration status for %s: %w", migName, err)
	}

	if applied {
		return nil
	}

	s.logger.Info("Applying migration", logger.Field{Key: "migration", Value: migName})

	migrationSQL := `
	CREATE TABLE IF NOT EXISTS server_processes (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		path TEXT NOT NULL,
		port INTEGER,
		sequence_order INTEGER NOT NULL,
		start_time TIMESTAMP,
		end_time TIMESTAMP,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_server_processes_sequence_order ON server_processes (sequence_order);
	`
	_, err = s.db.Exec(migrationSQL)
	if err != nil {
		return fmt.Errorf("failed to create server_processes table: %w", err)
	}

	if err := s.markMigrationApplied(migName); err != nil {
		s.logger.Error(
			"failed to mark migration as applied",
			logger.Field{Key: "migration", Value: migName},
			logger.Field{Key: "error", Value: err},
		)
		return fmt.Errorf("failed to mark migration as applied: %w", err)
	}

	return nil
}

func (s *sqliteInternalDB) rollback009ServerProcessesTable() error {
	const migName = "009_server_processes_table"

	applied, err := s.isMigrationApplied(migName)
	if err != nil {
		s.logger.Error(
			"failed to check migration status",
			logger.Field{Key: "migration", Value: migName},
			logger.Field{Key: "error", Value: err},
		)
	}

	if !applied {
		return nil
	}

	s.logger.Info("Rolling back migration", logger.Field{Key: "migration", Value: migName})

	migrationSQL := `
	DROP TABLE IF EXISTS server_processes;
	`
	_, err = s.db.Exec(migrationSQL)
	if err != nil {
		return fmt.Errorf("failed to rollback server_processes table: %w", err)
	}

	if err := s.markMigrationRolledBack(migName); err != nil {
		s.logger.Error(
			"failed to mark migration as rolled back",
			logger.Field{Key: "migration", Value: migName},
			logger.Field{Key: "error", Value: err},
		)
		return fmt.Errorf("failed to mark migration as rolled back: %w", err)
	}

	return nil
}

func (s *sqliteInternalDB) migrate010DirectoryShortcutsTable() error {
	const migName = "010_directory_shortcuts_table"

	applied, err := s.isMigrationApplied(migName)
	if err != nil {
		s.logger.Error(
			"failed to check migration status",
			logger.Field{Key: "migration", Value: migName},
			logger.Field{Key: "error", Value: err},
		)
		return fmt.Errorf("failed to check migration status for %s: %w", migName, err)
	}

	if applied {
		return nil
	}

	s.logger.Info("Applying migration", logger.Field{Key: "migration", Value: migName})

	migrationSQL := `
	CREATE TABLE IF NOT EXISTS directory_shortcuts (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		name TEXT NOT NULL,
		path TEXT NOT NULL,
		normalized_path TEXT NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP,
		UNIQUE(user_id, normalized_path)
	);

	CREATE INDEX IF NOT EXISTS idx_directory_shortcuts_user_id ON directory_shortcuts (user_id);

	CREATE INDEX IF NOT EXISTS idx_directory_shortcuts_normalized_path ON directory_shortcuts (normalized_path);
	`
	_, err = s.db.Exec(migrationSQL)
	if err != nil {
		return fmt.Errorf("failed to create directory_shortcuts table: %w", err)
	}

	if err := s.markMigrationApplied(migName); err != nil {
		s.logger.Error(
			"failed to mark migration as applied",
			logger.Field{Key: "migration", Value: migName},
			logger.Field{Key: "error", Value: err},
		)
		return fmt.Errorf("failed to mark migration as applied: %w", err)
	}

	return nil
}

func (s *sqliteInternalDB) rollback010DirectoryShortcutsTable() error {
	const migName = "010_directory_shortcuts_table"

	applied, err := s.isMigrationApplied(migName)
	if err != nil {
		s.logger.Error(
			"failed to check migration status",
			logger.Field{Key: "migration", Value: migName},
			logger.Field{Key: "error", Value: err},
		)
	}

	if !applied {
		return nil
	}

	s.logger.Info("Rolling back migration", logger.Field{Key: "migration", Value: migName})

	migrationSQL := `
	DROP TABLE IF EXISTS directory_shortcuts;
	`
	_, err = s.db.Exec(migrationSQL)
	if err != nil {
		return fmt.Errorf("failed to rollback directory_shortcuts table: %w", err)
	}

	if err := s.markMigrationRolledBack(migName); err != nil {
		s.logger.Error(
			"failed to mark migration as rolled back",
			logger.Field{Key: "migration", Value: migName},
			logger.Field{Key: "error", Value: err},
		)
		return fmt.Errorf("failed to mark migration as rolled back: %w", err)
	}

	return nil
}

func (s *sqliteInternalDB) migrate011ItemClientDataType() error {
	const migName = "011_item_client_data_type"

	applied, err := s.isMigrationApplied(migName)
	if err != nil {
		s.logger.Error(
			"failed to check migration status",
			logger.Field{Key: "migration", Value: migName},
			logger.Field{Key: "error", Value: err},
		)
		return fmt.Errorf("failed to check migration status for %s: %w", migName, err)
	}

	if applied {
		return nil
	}

	s.logger.Info("Applying migration", logger.Field{Key: "migration", Value: migName})

	migrationSQL := `
	CREATE TABLE IF NOT EXISTS item_client_data_new (
		id INTEGER NOT NULL,
		name TEXT NOT NULL,
		item_type TEXT NOT NULL DEFAULT 'it0' CHECK(item_type IN ('it0', 'it1', 'it2', 'it3')),
		created_by INTEGER REFERENCES users(id) ON DELETE SET NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_by INTEGER REFERENCES users(id) ON DELETE SET NULL,
		updated_at TIMESTAMP,
		PRIMARY KEY (id, item_type)
	);

	INSERT OR IGNORE INTO item_client_data_new (
		id,
		name,
		item_type,
		created_by,
		created_at,
		updated_by,
		updated_at
	)
	SELECT
		id,
		name,
		'it0',
		created_by,
		created_at,
		updated_by,
		updated_at
	FROM item_client_data;

	DROP TABLE item_client_data;

	ALTER TABLE item_client_data_new RENAME TO item_client_data;

	CREATE INDEX IF NOT EXISTS idx_item_client_data_name ON item_client_data (name);

	CREATE INDEX IF NOT EXISTS idx_item_client_data_item_type ON item_client_data (item_type);

	CREATE INDEX IF NOT EXISTS idx_item_client_data_created_by ON item_client_data (created_by);

	CREATE INDEX IF NOT EXISTS idx_item_client_data_updated_by ON item_client_data (updated_by);
	`
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin item client data type migration: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	_, err = tx.Exec(migrationSQL)
	if err != nil {
		return fmt.Errorf("failed to add item client data type: %w", err)
	}

	if _, err := tx.Exec("INSERT INTO migrations (name) VALUES (?)", migName); err != nil {
		s.logger.Error(
			"failed to mark migration as applied",
			logger.Field{Key: "migration", Value: migName},
			logger.Field{Key: "error", Value: err},
		)
		return fmt.Errorf("failed to mark migration as applied: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit item client data type migration: %w", err)
	}

	return nil
}

func (s *sqliteInternalDB) rollback011ItemClientDataType() error {
	const migName = "011_item_client_data_type"

	applied, err := s.isMigrationApplied(migName)
	if err != nil {
		s.logger.Error(
			"failed to check migration status",
			logger.Field{Key: "migration", Value: migName},
			logger.Field{Key: "error", Value: err},
		)
		return fmt.Errorf("failed to check migration status for %s: %w", migName, err)
	}

	if !applied {
		return nil
	}

	s.logger.Info("Rolling back migration", logger.Field{Key: "migration", Value: migName})

	migrationSQL := `
	CREATE TABLE IF NOT EXISTS item_client_data_old (
		id INTEGER PRIMARY KEY,
		name TEXT NOT NULL,
		created_by INTEGER REFERENCES users(id) ON DELETE SET NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_by INTEGER REFERENCES users(id) ON DELETE SET NULL,
		updated_at TIMESTAMP
	);

	INSERT OR IGNORE INTO item_client_data_old (
		id,
		name,
		created_by,
		created_at,
		updated_by,
		updated_at
	)
	SELECT
		id,
		name,
		created_by,
		created_at,
		updated_by,
		updated_at
	FROM item_client_data
	ORDER BY item_type;

	DROP TABLE item_client_data;

	ALTER TABLE item_client_data_old RENAME TO item_client_data;

	CREATE INDEX IF NOT EXISTS idx_item_client_data_name ON item_client_data (name);

	CREATE INDEX IF NOT EXISTS idx_item_client_data_created_by ON item_client_data (created_by);

	CREATE INDEX IF NOT EXISTS idx_item_client_data_updated_by ON item_client_data (updated_by);
	`
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin item client data type rollback: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	_, err = tx.Exec(migrationSQL)
	if err != nil {
		return fmt.Errorf("failed to rollback item client data type: %w", err)
	}

	if _, err := tx.Exec("DELETE FROM migrations WHERE name = ?", migName); err != nil {
		s.logger.Error(
			"failed to mark migration as rolled back",
			logger.Field{Key: "migration", Value: migName},
			logger.Field{Key: "error", Value: err},
		)
		return fmt.Errorf("failed to mark migration as rolled back: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit item client data type rollback: %w", err)
	}

	return nil
}

func (s *sqliteInternalDB) migrate012BackupJobsTables() error {
	const migName = "012_backup_jobs_tables"

	applied, err := s.isMigrationApplied(migName)
	if err != nil {
		s.logger.Error(
			"failed to check migration status",
			logger.Field{Key: "migration", Value: migName},
			logger.Field{Key: "error", Value: err},
		)
		return fmt.Errorf("failed to check migration status for %s: %w", migName, err)
	}

	if applied {
		return nil
	}

	s.logger.Info("Applying migration", logger.Field{Key: "migration", Value: migName})

	migrationSQL := `
	CREATE TABLE IF NOT EXISTS backup_jobs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		job_type TEXT NOT NULL,
		name TEXT NOT NULL,
		status TEXT NOT NULL DEFAULT 'active',
		cron_expression TEXT,
		destination_directory TEXT NOT NULL,
		archive_password TEXT,
		source_path TEXT,
		sql_host TEXT,
		sql_port INTEGER,
		sql_username TEXT,
		sql_password TEXT,
		sql_database_names TEXT,
		last_run_at TIMESTAMP,
		created_by INTEGER REFERENCES users(id) ON DELETE SET NULL,
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_by INTEGER REFERENCES users(id) ON DELETE SET NULL,
		updated_at TIMESTAMP,
		deleted_by INTEGER REFERENCES users(id) ON DELETE SET NULL,
		deleted_at TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_backup_jobs_status ON backup_jobs (status);

	CREATE INDEX IF NOT EXISTS idx_backup_jobs_job_type ON backup_jobs (job_type);

	CREATE INDEX IF NOT EXISTS idx_backup_jobs_created_at ON backup_jobs (created_at);

	CREATE INDEX IF NOT EXISTS idx_backup_jobs_last_run_at ON backup_jobs (last_run_at);

	CREATE TABLE IF NOT EXISTS backup_runs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		job_id INTEGER NOT NULL REFERENCES backup_jobs(id) ON DELETE RESTRICT,
		trigger_type TEXT NOT NULL,
		status TEXT NOT NULL DEFAULT 'running',
		previous_job_status TEXT NOT NULL,
		started_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		finished_at TIMESTAMP,
		cancel_requested_at TIMESTAMP,
		output TEXT,
		error_details TEXT,
		created_by INTEGER REFERENCES users(id) ON DELETE SET NULL,
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_backup_runs_job_id ON backup_runs (job_id);

	CREATE INDEX IF NOT EXISTS idx_backup_runs_status ON backup_runs (status);

	CREATE INDEX IF NOT EXISTS idx_backup_runs_started_at ON backup_runs (started_at);

	CREATE TABLE IF NOT EXISTS backup_run_files (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		run_id INTEGER NOT NULL REFERENCES backup_runs(id) ON DELETE CASCADE,
		item_name TEXT NOT NULL,
		file_path TEXT NOT NULL,
		file_size INTEGER NOT NULL DEFAULT 0,
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_backup_run_files_run_id ON backup_run_files (run_id);
	`

	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin backup jobs migration: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	_, err = tx.Exec(migrationSQL)
	if err != nil {
		return fmt.Errorf("failed to create backup jobs tables: %w", err)
	}

	if _, err := tx.Exec("INSERT INTO migrations (name) VALUES (?)", migName); err != nil {
		s.logger.Error(
			"failed to mark migration as applied",
			logger.Field{Key: "migration", Value: migName},
			logger.Field{Key: "error", Value: err},
		)
		return fmt.Errorf("failed to mark migration as applied: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit backup jobs migration: %w", err)
	}

	return nil
}

func (s *sqliteInternalDB) rollback012BackupJobsTables() error {
	const migName = "012_backup_jobs_tables"

	applied, err := s.isMigrationApplied(migName)
	if err != nil {
		s.logger.Error(
			"failed to check migration status",
			logger.Field{Key: "migration", Value: migName},
			logger.Field{Key: "error", Value: err},
		)
		return fmt.Errorf("failed to check migration status for %s: %w", migName, err)
	}

	if !applied {
		return nil
	}

	s.logger.Info("Rolling back migration", logger.Field{Key: "migration", Value: migName})

	rollbackSQL := `
	DROP TABLE IF EXISTS backup_run_files;
	DROP TABLE IF EXISTS backup_runs;
	DROP TABLE IF EXISTS backup_jobs;
	`

	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin backup jobs rollback: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	_, err = tx.Exec(rollbackSQL)
	if err != nil {
		return fmt.Errorf("failed to rollback backup jobs tables: %w", err)
	}

	if _, err := tx.Exec("DELETE FROM migrations WHERE name = ?", migName); err != nil {
		s.logger.Error(
			"failed to mark migration as rolled back",
			logger.Field{Key: "migration", Value: migName},
			logger.Field{Key: "error", Value: err},
		)
		return fmt.Errorf("failed to mark migration as rolled back: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit backup jobs rollback: %w", err)
	}

	return nil
}

func (s *sqliteInternalDB) migrate013ServerViewTables() error {
	const migName = "013_server_view_tables"

	applied, err := s.isMigrationApplied(migName)
	if err != nil {
		s.logger.Error(
			"failed to check migration status",
			logger.Field{Key: "migration", Value: migName},
			logger.Field{Key: "error", Value: err},
		)
		return fmt.Errorf("failed to check migration status for %s: %w", migName, err)
	}

	if applied {
		return nil
	}

	s.logger.Info("Applying migration", logger.Field{Key: "migration", Value: migName})

	migrationSQL := `
	CREATE TABLE IF NOT EXISTS server_view_svr_info_rows (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		server_type TEXT NOT NULL,
		section TEXT NOT NULL,
		key TEXT NOT NULL,
		value TEXT NOT NULL,
		value_index INTEGER NOT NULL DEFAULT 0,
		row_order INTEGER NOT NULL DEFAULT 0,
		updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(server_type, section, key, value_index)
	);

	CREATE INDEX IF NOT EXISTS idx_server_view_svr_info_server_type ON server_view_svr_info_rows (server_type);

	CREATE INDEX IF NOT EXISTS idx_server_view_svr_info_row_order ON server_view_svr_info_rows (server_type, row_order);

	CREATE TABLE IF NOT EXISTS server_view_map_zones (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		server_type TEXT NOT NULL,
		map_id INTEGER NOT NULL,
		zone_id INTEGER NOT NULL,
		updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(server_type, map_id)
	);

	CREATE INDEX IF NOT EXISTS idx_server_view_map_zones_server_type ON server_view_map_zones (server_type);

	CREATE INDEX IF NOT EXISTS idx_server_view_map_zones_map_id ON server_view_map_zones (map_id);

	CREATE TABLE IF NOT EXISTS server_view_spawn_rows (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		map_id INTEGER NOT NULL,
		row_index INTEGER NOT NULL,
		npc_id INTEGER NOT NULL,
		x INTEGER NOT NULL,
		y INTEGER NOT NULL,
		unknown1 INTEGER NOT NULL,
		orientation INTEGER NOT NULL,
		spawn_step INTEGER NOT NULL,
		updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(map_id, row_index)
	);

	CREATE INDEX IF NOT EXISTS idx_server_view_spawn_rows_map_id ON server_view_spawn_rows (map_id);

	CREATE INDEX IF NOT EXISTS idx_server_view_spawn_rows_npc_id ON server_view_spawn_rows (npc_id);

	CREATE TABLE IF NOT EXISTS server_view_drop_rows (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		npc_id INTEGER NOT NULL,
		row_index INTEGER NOT NULL,
		item_id INTEGER NOT NULL,
		drop_rate INTEGER NOT NULL,
		group_code INTEGER NOT NULL,
		updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(npc_id, row_index)
	);

	CREATE INDEX IF NOT EXISTS idx_server_view_drop_rows_npc_id ON server_view_drop_rows (npc_id);

	CREATE INDEX IF NOT EXISTS idx_server_view_drop_rows_item_id ON server_view_drop_rows (item_id);

	CREATE TABLE IF NOT EXISTS server_view_shop_rows (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		npc_id INTEGER NOT NULL,
		line_number INTEGER NOT NULL,
		item_id INTEGER NOT NULL,
		updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(npc_id, line_number)
	);

	CREATE INDEX IF NOT EXISTS idx_server_view_shop_rows_npc_id ON server_view_shop_rows (npc_id);

	CREATE INDEX IF NOT EXISTS idx_server_view_shop_rows_item_id ON server_view_shop_rows (item_id);

	CREATE TABLE IF NOT EXISTS server_view_game_master_rows (
		gm_index INTEGER PRIMARY KEY,
		name TEXT NOT NULL,
		level INTEGER,
		updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_server_view_game_master_rows_name ON server_view_game_master_rows (name);

	CREATE TABLE IF NOT EXISTS server_view_sync_runs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		status TEXT NOT NULL,
		started_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		finished_at TIMESTAMP,
		warning_count INTEGER NOT NULL DEFAULT 0,
		error_details TEXT,
		created_by INTEGER REFERENCES users(id) ON DELETE SET NULL,
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_server_view_sync_runs_status ON server_view_sync_runs (status);

	CREATE INDEX IF NOT EXISTS idx_server_view_sync_runs_started_at ON server_view_sync_runs (started_at);

	CREATE TABLE IF NOT EXISTS server_view_sync_warnings (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		run_id INTEGER NOT NULL REFERENCES server_view_sync_runs(id) ON DELETE CASCADE,
		source TEXT NOT NULL,
		message TEXT NOT NULL,
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_server_view_sync_warnings_run_id ON server_view_sync_warnings (run_id);
	`

	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin server view migration: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	_, err = tx.Exec(migrationSQL)
	if err != nil {
		return fmt.Errorf("failed to create server view tables: %w", err)
	}

	if _, err := tx.Exec("INSERT INTO migrations (name) VALUES (?)", migName); err != nil {
		s.logger.Error(
			"failed to mark migration as applied",
			logger.Field{Key: "migration", Value: migName},
			logger.Field{Key: "error", Value: err},
		)
		return fmt.Errorf("failed to mark migration as applied: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit server view migration: %w", err)
	}

	return nil
}

func (s *sqliteInternalDB) rollback013ServerViewTables() error {
	const migName = "013_server_view_tables"

	applied, err := s.isMigrationApplied(migName)
	if err != nil {
		s.logger.Error(
			"failed to check migration status",
			logger.Field{Key: "migration", Value: migName},
			logger.Field{Key: "error", Value: err},
		)
		return fmt.Errorf("failed to check migration status for %s: %w", migName, err)
	}

	if !applied {
		return nil
	}

	s.logger.Info("Rolling back migration", logger.Field{Key: "migration", Value: migName})

	rollbackSQL := `
	DROP TABLE IF EXISTS server_view_sync_warnings;
	DROP TABLE IF EXISTS server_view_sync_runs;
	DROP TABLE IF EXISTS server_view_game_master_rows;
	DROP TABLE IF EXISTS server_view_shop_rows;
	DROP TABLE IF EXISTS server_view_drop_rows;
	DROP TABLE IF EXISTS server_view_spawn_rows;
	DROP TABLE IF EXISTS server_view_map_zones;
	DROP TABLE IF EXISTS server_view_svr_info_rows;
	`

	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin server view rollback: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	_, err = tx.Exec(rollbackSQL)
	if err != nil {
		return fmt.Errorf("failed to rollback server view tables: %w", err)
	}

	if _, err := tx.Exec("DELETE FROM migrations WHERE name = ?", migName); err != nil {
		s.logger.Error(
			"failed to mark migration as rolled back",
			logger.Field{Key: "migration", Value: migName},
			logger.Field{Key: "error", Value: err},
		)
		return fmt.Errorf("failed to mark migration as rolled back: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit server view rollback: %w", err)
	}

	return nil
}
