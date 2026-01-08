package db

import (
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/doug-martin/goqu/v9"
	"github.com/omnihance/omnihance-a3-agent/internal/logger"
)

type DirectoryShortcut struct {
	ID             int64      `db:"id" json:"id"`
	UserID         int64      `db:"user_id" json:"user_id"`
	Name           string     `db:"name" json:"name"`
	Path           string     `db:"path" json:"path"`
	NormalizedPath string     `db:"normalized_path" json:"normalized_path"`
	CreatedAt      time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt      *time.Time `db:"updated_at" json:"updated_at"`
}

func normalizePathForShortcut(path string) string {
	normalized := strings.ReplaceAll(path, "\\", "/")
	normalized = strings.TrimSpace(normalized)
	normalized = strings.ToLower(normalized)
	normalized = strings.TrimSuffix(normalized, "/")

	if runtime.GOOS == "windows" {
		if len(normalized) == 2 && normalized[1] == ':' {
			return normalized
		}
		if len(normalized) == 3 && normalized[1] == ':' && normalized[2] == '/' {
			return normalized[:2]
		}
	}

	return normalized
}

func (s *sqliteInternalDB) GetDirectoryShortcuts(userID int64) ([]DirectoryShortcut, error) {
	var shortcuts []DirectoryShortcut
	err := s.goqu.From("directory_shortcuts").
		Prepared(true).
		Where(goqu.Ex{"user_id": userID}).
		Order(goqu.C("created_at").Asc()).
		ScanStructs(&shortcuts)
	if err != nil {
		s.logger.Error(
			"failed to get directory shortcuts",
			logger.Field{Key: "user_id", Value: userID},
			logger.Field{Key: "error", Value: err},
		)
		return nil, fmt.Errorf("failed to get directory shortcuts: %w", err)
	}

	return shortcuts, nil
}

func (s *sqliteInternalDB) GetDirectoryShortcut(id int64) (*DirectoryShortcut, error) {
	var shortcut DirectoryShortcut
	found, err := s.goqu.From("directory_shortcuts").
		Prepared(true).
		Where(goqu.Ex{"id": id}).
		ScanStruct(&shortcut)
	if err != nil {
		s.logger.Error(
			"failed to get directory shortcut",
			logger.Field{Key: "id", Value: id},
			logger.Field{Key: "error", Value: err},
		)
		return nil, fmt.Errorf("failed to get directory shortcut: %w", err)
	}

	if !found {
		return nil, fmt.Errorf("directory shortcut not found")
	}

	return &shortcut, nil
}

func (s *sqliteInternalDB) GetDirectoryShortcutCount(userID int64) (int64, error) {
	count, err := s.goqu.From("directory_shortcuts").
		Prepared(true).
		Where(goqu.Ex{"user_id": userID}).
		Count()
	if err != nil {
		s.logger.Error(
			"failed to get directory shortcut count",
			logger.Field{Key: "user_id", Value: userID},
			logger.Field{Key: "error", Value: err},
		)
		return 0, fmt.Errorf("failed to get directory shortcut count: %w", err)
	}

	return count, nil
}

func (s *sqliteInternalDB) GetDirectoryShortcutByNormalizedPath(userID int64, normalizedPath string) (*DirectoryShortcut, error) {
	var shortcut DirectoryShortcut
	found, err := s.goqu.From("directory_shortcuts").
		Prepared(true).
		Where(goqu.Ex{"user_id": userID, "normalized_path": normalizedPath}).
		ScanStruct(&shortcut)
	if err != nil {
		s.logger.Error(
			"failed to get directory shortcut by normalized path",
			logger.Field{Key: "user_id", Value: userID},
			logger.Field{Key: "normalized_path", Value: normalizedPath},
			logger.Field{Key: "error", Value: err},
		)
		return nil, fmt.Errorf("failed to get directory shortcut by normalized path: %w", err)
	}

	if !found {
		return nil, nil
	}

	return &shortcut, nil
}

func (s *sqliteInternalDB) CreateDirectoryShortcut(userID int64, name, path string) (*DirectoryShortcut, error) {
	normalizedPath := normalizePathForShortcut(path)
	cleanPath := filepath.Clean(path)

	insertRecord := goqu.Record{
		"user_id":         userID,
		"name":            name,
		"path":            cleanPath,
		"normalized_path": normalizedPath,
		"created_at":      goqu.L("CURRENT_TIMESTAMP"),
	}

	result, err := s.goqu.Insert("directory_shortcuts").
		Prepared(true).
		Rows(insertRecord).
		Executor().
		Exec()
	if err != nil {
		s.logger.Error(
			"failed to create directory shortcut",
			logger.Field{Key: "user_id", Value: userID},
			logger.Field{Key: "name", Value: name},
			logger.Field{Key: "path", Value: path},
			logger.Field{Key: "error", Value: err},
		)
		return nil, fmt.Errorf("failed to create directory shortcut: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get last insert id: %w", err)
	}

	return s.GetDirectoryShortcut(id)
}

func (s *sqliteInternalDB) DeleteDirectoryShortcut(id int64, userID int64) error {
	_, err := s.goqu.Delete("directory_shortcuts").
		Prepared(true).
		Where(goqu.Ex{"id": id, "user_id": userID}).
		Executor().
		Exec()
	if err != nil {
		s.logger.Error(
			"failed to delete directory shortcut",
			logger.Field{Key: "id", Value: id},
			logger.Field{Key: "user_id", Value: userID},
			logger.Field{Key: "error", Value: err},
		)
		return fmt.Errorf("failed to delete directory shortcut: %w", err)
	}

	return nil
}
