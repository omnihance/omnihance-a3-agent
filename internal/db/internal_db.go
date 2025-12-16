package db

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/doug-martin/goqu/v9"
	"github.com/omnihance/omnihance-a3-agent/internal/logger"
	"github.com/robfig/cron/v3"
	_ "modernc.org/sqlite"
)

type InternalDB interface {
	Connect() error
	Close() error
	MigrateUp() error
	MigrateDown() error
	GetSettings() ([]Settings, error)
	GetSetting(key string) (*Settings, error)
	SetSetting(key string, value string, userID *int64) error
	SetSettingIfNotExists(key string, value string, userID *int64) error
	DeleteSetting(key string) error
	InsertMetric(metricName string, metricType MetricType, labels map[string]string, value float64, timestamp *int64, unit *string, description *string) error
	InsertMetricSample(seriesID int64, value float64, timestamp *int64) error
	GetSeriesWithLabels() ([]SeriesWithLabels, error)
	GetLatestSamples() ([]LatestSample, error)
	GetMetricSamplesByTimeRange(metricName string, startTime, endTime int64) ([]MetricSampleWithLabels, error)
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
	CreateUser(email, password, roles string, createdBy *int64) (*User, error)
	CreateUserWithStatus(email, password, roles, status string, createdBy *int64) (*User, error)
	UpdateUserPassword(userID int64, newPassword string, updatedBy int64) error
	UpdateUserRoles(userID int64, roles string, updatedBy int64) error
	DeleteUser(userID int64, deletedBy int64) error
	GetAdminUserCount() (int64, error)
	SetDefaultSettings() error
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

	pragmas := []string{
		`PRAGMA journal_mode=WAL;`,
		`PRAGMA synchronous=NORMAL;`,
		`PRAGMA busy_timeout=5000;`,
		`PRAGMA foreign_keys=ON;`,
		`PRAGMA cache_size=-64000;`,
		`PRAGMA temp_store=MEMORY;`,
		`PRAGMA mmap_size=30000000000;`,
		`PRAGMA wal_checkpoint(TRUNCATE);`,
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

	s.cron = cron.New(cron.WithSeconds())
	_, err = s.cron.AddFunc("@every 5m", s.runWALCheckpoint)
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

	return nil
}

func (s *sqliteInternalDB) MigrateDown() error {
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

	_, err := s.db.Exec(`PRAGMA wal_checkpoint(TRUNCATE);`)
	if err != nil {
		s.logger.Error(
			"failed to run WAL checkpoint",
			logger.Field{Key: "error", Value: err},
		)
		return
	}

	s.logger.Debug("WAL checkpoint completed successfully")
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
		created_at INTEGER NOT NULL DEFAULT (strftime('%s', 'now')),
		updated_by INTEGER REFERENCES users(id) ON DELETE SET NULL,
		updated_at INTEGER,
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
			created_at INTEGER NOT NULL DEFAULT (strftime('%s', 'now'))
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
			created_at INTEGER NOT NULL DEFAULT (strftime('%s', 'now')),
			last_updated INTEGER NOT NULL DEFAULT (strftime('%s', 'now')),
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
			SET last_updated = NEW.timestamp 
			WHERE id = NEW.series_id 
			  AND last_updated < NEW.timestamp;
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
