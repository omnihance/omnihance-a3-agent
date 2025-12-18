package db

import (
	"fmt"
	"strings"
	"time"

	"github.com/doug-martin/goqu/v9"
	"github.com/omnihance/omnihance-a3-agent/internal/logger"
)

type ItemClientData struct {
	ID        int64      `db:"id" json:"id"`
	Name      string     `db:"name" json:"name"`
	CreatedBy *int64     `db:"created_by" json:"created_by"`
	CreatedAt time.Time  `db:"created_at" json:"created_at"`
	UpdatedBy *int64     `db:"updated_by" json:"updated_by"`
	UpdatedAt *time.Time `db:"updated_at" json:"updated_at"`
}

func (s *sqliteInternalDB) BulkReplaceItemClientData(data []ItemClientData) error {
	_, err := s.goqu.Delete("item_client_data").
		Prepared(true).
		Executor().
		Exec()
	if err != nil {
		s.logger.Error(
			"failed to delete existing item client data",
			logger.Field{Key: "error", Value: err},
		)
		return fmt.Errorf("failed to delete existing item client data: %w", err)
	}

	if len(data) == 0 {
		return nil
	}

	records := make([]goqu.Record, 0, len(data))
	for _, item := range data {
		record := goqu.Record{
			"id":   item.ID,
			"name": item.Name,
		}

		if item.CreatedBy != nil {
			record["created_by"] = *item.CreatedBy
		}

		if item.UpdatedBy != nil {
			record["updated_by"] = *item.UpdatedBy
		}

		if item.UpdatedAt != nil {
			record["updated_at"] = *item.UpdatedAt
		}

		records = append(records, record)
	}

	_, err = s.goqu.Insert("item_client_data").
		Prepared(true).
		Rows(records).
		Executor().
		Exec()
	if err != nil {
		s.logger.Error(
			"failed to bulk insert item client data",
			logger.Field{Key: "count", Value: len(data)},
			logger.Field{Key: "error", Value: err},
		)
		return fmt.Errorf("failed to bulk insert item client data: %w", err)
	}

	return nil
}

func (s *sqliteInternalDB) GetAllItemClientData(search string) ([]ItemClientData, error) {
	var data []ItemClientData

	query := s.goqu.From("item_client_data").
		Prepared(true)

	if search != "" {
		query = query.Where(goqu.L("LOWER(name)").Like("%" + strings.ToLower(search) + "%"))
	}

	err := query.ScanStructs(&data)
	if err != nil {
		s.logger.Error(
			"failed to get item client data",
			logger.Field{Key: "search", Value: search},
			logger.Field{Key: "error", Value: err},
		)
		return nil, fmt.Errorf("failed to get item client data: %w", err)
	}

	return data, nil
}
