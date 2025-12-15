package db

import (
	"fmt"

	"github.com/doug-martin/goqu/v9"
	"github.com/omnihance/omnihance-a3-agent/internal/logger"
)

type Settings struct {
	Key       string `db:"key" json:"key"`
	Value     string `db:"value" json:"value"`
	CreatedBy *int64 `db:"created_by" json:"created_by"`
	CreatedAt int64  `db:"created_at" json:"created_at"`
	UpdatedBy *int64 `db:"updated_by" json:"updated_by"`
	UpdatedAt *int64 `db:"updated_at" json:"updated_at"`
}

var defaultSettings = map[string]string{
	"DB_HOST": "127.0.0.1",
	"DB_PORT": "1433",
	"DB_USER": "sa",
	"DB_PASS": "ley",
	"DB_NAME": "ASD",
}

func (s *sqliteInternalDB) GetSettings() ([]Settings, error) {
	var settings []Settings
	err := s.goqu.From("settings").
		Prepared(true).
		ScanStructs(&settings)
	if err != nil {
		s.logger.Error(
			"failed to get settings",
			logger.Field{Key: "error", Value: err},
		)
		return nil, fmt.Errorf("failed to get settings: %w", err)
	}
	return settings, nil
}

func (s *sqliteInternalDB) GetSetting(key string) (*Settings, error) {
	var setting Settings
	found, err := s.goqu.From("settings").
		Prepared(true).
		Where(goqu.Ex{"key": key}).
		ScanStruct(&setting)
	if err != nil {
		s.logger.Error(
			"failed to get setting",
			logger.Field{Key: "key", Value: key},
			logger.Field{Key: "error", Value: err},
		)
		return nil, fmt.Errorf("failed to get setting %s: %w", key, err)
	}

	if !found {
		return nil, fmt.Errorf("setting %s not found", key)
	}

	return &setting, nil
}

func (s *sqliteInternalDB) SetSetting(key string, value string, userID *int64) error {
	insertRecord := goqu.Record{
		"key":   key,
		"value": value,
	}

	if userID != nil {
		insertRecord["created_by"] = *userID
	}

	updateRecord := goqu.Record{
		"value":      value,
		"updated_at": goqu.L("strftime('%s', 'now')"),
	}

	if userID != nil {
		updateRecord["updated_by"] = *userID
	}

	_, err := s.goqu.Insert("settings").
		Prepared(true).
		Rows(insertRecord).
		OnConflict(goqu.DoUpdate("key", updateRecord)).
		Executor().
		Exec()
	if err != nil {
		s.logger.Error(
			"failed to set setting",
			logger.Field{Key: "key", Value: key},
			logger.Field{Key: "user_id", Value: userID},
			logger.Field{Key: "error", Value: err},
		)
		return fmt.Errorf("failed to set setting %s: %w", key, err)
	}

	return nil
}

func (s *sqliteInternalDB) SetSettingIfNotExists(key string, value string, userID *int64) error {
	insertRecord := goqu.Record{
		"key":   key,
		"value": value,
	}

	if userID != nil {
		insertRecord["created_by"] = *userID
	}

	_, err := s.goqu.Insert("settings").
		Prepared(true).
		Rows(insertRecord).
		OnConflict(goqu.DoNothing()).
		Executor().
		Exec()
	if err != nil {
		s.logger.Error(
			"failed to set setting if not exists",
			logger.Field{Key: "key", Value: key},
			logger.Field{Key: "user_id", Value: userID},
			logger.Field{Key: "error", Value: err},
		)
		return fmt.Errorf("failed to set setting %s if not exists: %w", key, err)
	}

	return nil
}

func (s *sqliteInternalDB) DeleteSetting(key string) error {
	_, err := s.goqu.Delete("settings").
		Prepared(true).
		Where(goqu.Ex{"key": key}).
		Executor().
		Exec()
	if err != nil {
		s.logger.Error(
			"failed to delete setting",
			logger.Field{Key: "key", Value: key},
			logger.Field{Key: "error", Value: err},
		)
		return fmt.Errorf("failed to delete setting %s: %w", key, err)
	}

	return nil
}

func (s *sqliteInternalDB) SetDefaultSettings() error {
	for key, value := range defaultSettings {
		err := s.SetSettingIfNotExists(key, value, nil)
		if err != nil {
			return fmt.Errorf("failed to set default setting %s: %w", key, err)
		}
	}

	return nil
}
