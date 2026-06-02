package db

import (
	"path/filepath"
	"testing"

	"github.com/omnihance/omnihance-a3-agent/internal/logger"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
)

func TestBulkReplaceItemClientDataReplacesOnlyMatchingType(t *testing.T) {
	internalDB := newItemClientDataTestDB(t)

	require.NoError(t, internalDB.BulkReplaceItemClientData(ItemClientDataTypeIT0, []ItemClientData{
		{ID: 1, Name: "IT0 Sword"},
		{ID: 2, Name: "IT0 Axe"},
	}))
	require.NoError(t, internalDB.BulkReplaceItemClientData(ItemClientDataTypeIT1, []ItemClientData{
		{ID: 1, Name: "IT1 Potion"},
	}))
	require.NoError(t, internalDB.BulkReplaceItemClientData(ItemClientDataTypeIT0, []ItemClientData{
		{ID: 3, Name: "IT0 Bow"},
	}))

	items, err := internalDB.GetAllItemClientData("")
	require.NoError(t, err)
	require.Equal(t, []ItemClientData{
		{ID: 3, Name: "IT0 Bow", ItemType: string(ItemClientDataTypeIT0)},
		{ID: 1, Name: "IT1 Potion", ItemType: string(ItemClientDataTypeIT1)},
	}, stripItemClientDataAuditFields(items))
}

func TestGetItemClientDataCounts(t *testing.T) {
	internalDB := newItemClientDataTestDB(t)

	require.NoError(t, internalDB.BulkReplaceItemClientData(ItemClientDataTypeIT0, []ItemClientData{
		{ID: 1, Name: "IT0 Sword"},
		{ID: 2, Name: "IT0 Axe"},
	}))
	require.NoError(t, internalDB.BulkReplaceItemClientData(ItemClientDataTypeIT2, []ItemClientData{
		{ID: 1, Name: "IT2 Skill"},
	}))

	counts, err := internalDB.GetItemClientDataCounts()
	require.NoError(t, err)
	require.Equal(t, ItemClientDataCounts{IT0: 2, IT2: 1}, counts)
}

func TestBulkReplaceMonsterClientDataRollsBackOnInsertFailure(t *testing.T) {
	internalDB := newItemClientDataTestDB(t)

	require.NoError(t, internalDB.BulkReplaceMonsterClientData([]MonsterClientData{
		{ID: 1, Name: "Wolf"},
	}))

	err := internalDB.BulkReplaceMonsterClientData([]MonsterClientData{
		{ID: 2, Name: "Bad Wolf"},
		{ID: 2, Name: "Duplicate Wolf"},
	})
	require.Error(t, err)

	monsters, err := internalDB.GetAllMonsterClientData("")
	require.NoError(t, err)
	require.Equal(t, []MonsterClientData{{ID: 1, Name: "Wolf"}}, stripMonsterClientDataAuditFields(monsters))
}

func TestBulkReplaceMapClientDataRollsBackOnInsertFailure(t *testing.T) {
	internalDB := newItemClientDataTestDB(t)

	require.NoError(t, internalDB.BulkReplaceMapClientData([]MapClientData{
		{ID: 1, Name: "Temerina"},
	}))

	err := internalDB.BulkReplaceMapClientData([]MapClientData{
		{ID: 2, Name: "Bad Map"},
		{ID: 2, Name: "Duplicate Map"},
	})
	require.Error(t, err)

	maps, err := internalDB.GetAllMapClientData("")
	require.NoError(t, err)
	require.Equal(t, []MapClientData{{ID: 1, Name: "Temerina"}}, stripMapClientDataAuditFields(maps))
}

func TestBulkReplaceItemClientDataRollsBackOnInsertFailure(t *testing.T) {
	internalDB := newItemClientDataTestDB(t)

	require.NoError(t, internalDB.BulkReplaceItemClientData(ItemClientDataTypeIT0, []ItemClientData{
		{ID: 1, Name: "Sword"},
	}))

	err := internalDB.BulkReplaceItemClientData(ItemClientDataTypeIT0, []ItemClientData{
		{ID: 2, Name: "Bad Item"},
		{ID: 2, Name: "Duplicate Item"},
	})
	require.Error(t, err)

	items, err := internalDB.GetAllItemClientData("")
	require.NoError(t, err)
	require.Equal(t, []ItemClientData{{ID: 1, Name: "Sword", ItemType: string(ItemClientDataTypeIT0)}}, stripItemClientDataAuditFields(items))
}

func newItemClientDataTestDB(t *testing.T) InternalDB {
	t.Helper()

	log := logger.NewZerologLogger(zerolog.Nop(), "test", zerolog.Disabled)
	internalDB := NewSQLiteDB(filepath.Join(t.TempDir(), "test.db"), log)
	require.NoError(t, internalDB.Connect())
	require.NoError(t, internalDB.MigrateUp())
	t.Cleanup(func() {
		_ = internalDB.Close()
	})

	return internalDB
}

func stripMonsterClientDataAuditFields(items []MonsterClientData) []MonsterClientData {
	stripped := make([]MonsterClientData, 0, len(items))
	for _, item := range items {
		stripped = append(stripped, MonsterClientData{
			ID:   item.ID,
			Name: item.Name,
		})
	}

	return stripped
}

func stripMapClientDataAuditFields(items []MapClientData) []MapClientData {
	stripped := make([]MapClientData, 0, len(items))
	for _, item := range items {
		stripped = append(stripped, MapClientData{
			ID:   item.ID,
			Name: item.Name,
		})
	}

	return stripped
}

func stripItemClientDataAuditFields(items []ItemClientData) []ItemClientData {
	stripped := make([]ItemClientData, 0, len(items))
	for _, item := range items {
		stripped = append(stripped, ItemClientData{
			ID:       item.ID,
			Name:     item.Name,
			ItemType: item.ItemType,
		})
	}

	return stripped
}
