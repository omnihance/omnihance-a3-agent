package db

import (
	"fmt"
	"strings"
	"time"

	"github.com/doug-martin/goqu/v9"
	"github.com/omnihance/omnihance-a3-agent/internal/logger"
)

type ItemClientDataType string

const (
	ItemClientDataTypeIT0 ItemClientDataType = "it0"
	ItemClientDataTypeIT1 ItemClientDataType = "it1"
	ItemClientDataTypeIT2 ItemClientDataType = "it2"
	ItemClientDataTypeIT3 ItemClientDataType = "it3"
)

type ItemClientData struct {
	ID        int64      `db:"id" json:"id"`
	Name      string     `db:"name" json:"name"`
	ItemType  string     `db:"item_type" json:"item_type"`
	CreatedBy *int64     `db:"created_by" json:"created_by"`
	CreatedAt time.Time  `db:"created_at" json:"created_at"`
	UpdatedBy *int64     `db:"updated_by" json:"updated_by"`
	UpdatedAt *time.Time `db:"updated_at" json:"updated_at"`
}

type ItemClientDataCounts struct {
	IT0 int64 `json:"it0"`
	IT1 int64 `json:"it1"`
	IT2 int64 `json:"it2"`
	IT3 int64 `json:"it3"`
}

func (s *sqliteInternalDB) BulkReplaceItemClientData(itemType ItemClientDataType, data []ItemClientData) error {
	tx, err := s.BeginTx()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	committed := false
	defer func() {
		if !committed {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				s.logger.Error(
					"failed to rollback item client data transaction",
					logger.Field{Key: "error", Value: rollbackErr},
				)
			}
		}
	}()

	_, err = tx.Delete("item_client_data").
		Where(goqu.C("item_type").Eq(string(itemType))).
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
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("failed to commit item client data replacement: %w", err)
		}

		committed = true
		return nil
	}

	records := make([]goqu.Record, 0, len(data))
	for _, item := range data {
		record := goqu.Record{
			"id":        item.ID,
			"name":      item.Name,
			"item_type": string(itemType),
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

	_, err = tx.Insert("item_client_data").
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

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit item client data replacement: %w", err)
	}

	committed = true

	return nil
}

func (s *sqliteInternalDB) GetAllItemClientData(search string) ([]ItemClientData, error) {
	var data []ItemClientData

	query := s.goqu.From("item_client_data").
		Order(goqu.C("item_type").Asc(), goqu.C("id").Asc()).
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

func (s *sqliteInternalDB) GetItemClientDataCounts() (ItemClientDataCounts, error) {
	var data []itemClientDataCountRow

	err := s.goqu.From("item_client_data").
		Select("item_type", goqu.COUNT("*").As("count")).
		GroupBy("item_type").
		Prepared(true).
		ScanStructs(&data)
	if err != nil {
		s.logger.Error(
			"failed to get item client data counts",
			logger.Field{Key: "error", Value: err},
		)
		return ItemClientDataCounts{}, fmt.Errorf("failed to get item client data counts: %w", err)
	}

	counts := ItemClientDataCounts{}
	for _, row := range data {
		switch ItemClientDataType(row.ItemType) {
		case ItemClientDataTypeIT0:
			counts.IT0 = row.Count
		case ItemClientDataTypeIT1:
			counts.IT1 = row.Count
		case ItemClientDataTypeIT2:
			counts.IT2 = row.Count
		case ItemClientDataTypeIT3:
			counts.IT3 = row.Count
		}
	}

	return counts, nil
}

type itemClientDataCountRow struct {
	ItemType string `db:"item_type"`
	Count    int64  `db:"count"`
}
