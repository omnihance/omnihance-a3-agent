package services

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"path/filepath"
	"testing"

	"github.com/omnihance/omnihance-a3-agent/internal/db"
	"github.com/omnihance/omnihance-a3-agent/internal/logger"
	"github.com/project-agonyl/agonyl-utils-go/dropfile"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
	"gopkg.in/ini.v1"
)

func TestServerViewParsesSvrInfoDuplicateKeysAndComments(t *testing.T) {
	cfg := loadServerViewTestINI(t, `
; ignored
# ignored
[Server]
MaxMap=1
MaxMap=2
Name=Main
`)

	rows := svrInfoRowsFromINI(db.ServerViewServerTypeMain, cfg)

	require.Len(t, rows, 3)
	require.Equal(t, db.ServerViewServerTypeMain, rows[0].ServerType)
	require.Equal(t, "Server", rows[0].Section)
	require.Equal(t, "MaxMap", rows[0].Key)
	require.Equal(t, "1", rows[0].Value)
	require.Equal(t, 0, rows[0].ValueIndex)
	require.Equal(t, "MaxMap", rows[1].Key)
	require.Equal(t, "2", rows[1].Value)
	require.Equal(t, 1, rows[1].ValueIndex)
	require.Equal(t, "Name", rows[2].Key)
	require.Equal(t, "Main", rows[2].Value)
}

func TestServerViewParsesMapInfoRows(t *testing.T) {
	cfg := loadServerViewTestINI(t, `
[MapZone]
Map2=7
Map1=3
Bad=9
MapBad=4
`)

	rows := mapRowsFromINI(db.ServerViewServerTypeZone, cfg)

	require.Equal(t, []db.ServerViewMapZone{
		{ServerType: db.ServerViewServerTypeZone, MapID: 1, ZoneID: 3},
		{ServerType: db.ServerViewServerTypeZone, MapID: 2, ZoneID: 7},
	}, stripMapZoneAuditFields(rows))
}

func TestServerViewParsesGameMasters(t *testing.T) {
	cfg := loadServerViewTestINI(t, `
[GM]
GMCount=3
GMName_0=Alice
GMLevel_0=10
GMName_2=Charlie
`)

	rows := gameMasterRowsFromINI(cfg)

	require.Len(t, rows, 2)
	require.Equal(t, 0, rows[0].GMIndex)
	require.Equal(t, "Alice", rows[0].Name)
	require.NotNil(t, rows[0].Level)
	require.Equal(t, 10, *rows[0].Level)
	require.Equal(t, 2, rows[1].GMIndex)
	require.Equal(t, "Charlie", rows[1].Name)
	require.Nil(t, rows[1].Level)
}

func TestServerViewShopRowsUseOneBasedFileLines(t *testing.T) {
	fileEditor := NewMockFileEditorService(t)
	fileEditor.EXPECT().ReadFile("100.txt").Return([]byte("101\r\n\r\n202\n"), nil)
	service := &serverViewService{fileEditor: fileEditor}

	rows, err := service.shopRowsFromFile(100, "100.txt")

	require.NoError(t, err)
	require.Equal(t, []db.ServerViewShopRow{
		{NPCID: 100, LineNumber: 1, ItemID: 101},
		{NPCID: 100, LineNumber: 3, ItemID: 202},
	}, stripShopAuditFields(rows))
}

func TestServerViewAggregatesRawRowsWithClientNames(t *testing.T) {
	internalDB := newServerViewTestDB(t)
	require.NoError(t, internalDB.BulkReplaceMapClientData([]db.MapClientData{{ID: 1, Name: "Temenos"}}))
	require.NoError(t, internalDB.BulkReplaceMonsterClientData([]db.MonsterClientData{
		{ID: 50, Name: "Wolf"},
		{ID: 51, Name: "Bear"},
	}))
	require.NoError(t, internalDB.BulkReplaceItemClientData(db.ItemClientDataTypeIT0, []db.ItemClientData{
		{ID: 12, Name: "Claw"},
		{ID: 99, Name: "Fang"},
	}))
	require.NoError(t, internalDB.ReplaceServerViewSpawnRowsForMap(1, []db.ServerViewSpawnRow{
		{MapID: 1, RowIndex: 1, NPCID: 50},
		{MapID: 1, RowIndex: 2, NPCID: 50},
	}))
	require.NoError(t, internalDB.ReplaceServerViewDropRowsForNPC(50, []db.ServerViewDropRow{
		{NPCID: 50, RowIndex: 1, ItemID: 0x4000 + 12, DropRate: 5, GroupCode: 8},
	}))
	require.NoError(t, internalDB.ReplaceServerViewShopRowsForNPC(50, []db.ServerViewShopRow{
		{NPCID: 50, LineNumber: 1, ItemID: 12},
	}))
	require.NoError(t, internalDB.ReplaceServerViewShopRowsForNPC(51, []db.ServerViewShopRow{
		{NPCID: 51, LineNumber: 1, ItemID: 99},
	}))
	service := &serverViewService{internalDB: internalDB}

	spawns, err := service.GetZoneSpawns(context.Background(), "teme", "wolf")
	require.NoError(t, err)
	require.Len(t, spawns, 1)
	require.Equal(t, int64(2), spawns[0].Count)
	require.Equal(t, "Temenos (1)", spawns[0].MapDisplay)
	require.Equal(t, "Wolf (50)", spawns[0].NPCDisplay)

	drops, err := service.GetZoneDrops(context.Background(), "claw")
	require.NoError(t, err)
	require.Len(t, drops, 1)
	require.Equal(t, int64(50), drops[0].NPCID)

	dropDetails, err := service.GetZoneDropDetails(context.Background(), 50, "claw")
	require.NoError(t, err)
	require.Len(t, dropDetails, 1)
	require.Equal(t, int64(0x4000+12), dropDetails[0].ItemID)
	require.Equal(t, "Claw (16396)", dropDetails[0].ItemDisplay)

	shops, err := service.GetZoneShops(context.Background(), "wolf")
	require.NoError(t, err)
	require.Len(t, shops, 1)
	require.Equal(t, int64(1), shops[0].ItemCount)

	shops, err = service.GetZoneShops(context.Background(), "claw")
	require.NoError(t, err)
	require.Len(t, shops, 1)
	require.Equal(t, int64(50), shops[0].NPCID)

	shops, err = service.GetZoneShops(context.Background(), "12")
	require.NoError(t, err)
	require.Len(t, shops, 1)
	require.Equal(t, int64(50), shops[0].NPCID)

	shopDetails, err := service.GetZoneShopDetails(context.Background(), 50, "claw")
	require.NoError(t, err)
	require.Len(t, shopDetails, 1)
	require.Equal(t, "Claw (12)", shopDetails[0].ItemDisplay)
}

func TestServerViewOverviewBeforeFirstSyncReturnsEmptyArrays(t *testing.T) {
	internalDB := newServerViewTestDB(t)
	service := &serverViewService{internalDB: internalDB}

	overview, err := service.GetOverview(context.Background())

	require.NoError(t, err)
	require.Len(t, overview.Servers, 4)
	require.Nil(t, overview.Sync.LatestRun)
	require.NotNil(t, overview.Sync.Warnings)
	for _, server := range overview.Servers {
		require.NotNil(t, server.Sections)
	}

	data, err := json.Marshal(overview)
	require.NoError(t, err)
	require.NotContains(t, string(data), `"sections":null`)
	require.Contains(t, string(data), `"latest_run":null`)
	require.Contains(t, string(data), `"warnings":[]`)
}

func TestServerViewStartSyncReturnsRunningWhenLocked(t *testing.T) {
	service := &serverViewService{running: true}

	_, err := service.StartSync(context.Background(), nil)

	require.ErrorIs(t, err, ErrServerViewSyncRunning)
}

func TestServerViewSyncDropsLeavesExistingRowsWhenFileFails(t *testing.T) {
	internalDB := newServerViewTestDB(t)
	require.NoError(t, internalDB.ReplaceServerViewDropRowsForNPC(100, []db.ServerViewDropRow{
		{NPCID: 100, RowIndex: 1, ItemID: 500, DropRate: 10, GroupCode: 2},
	}))
	fileEditor := NewMockFileEditorService(t)
	dir := filepath.Join("zone", filepath.FromSlash(serverViewNPCDataDir))
	fileEditor.EXPECT().ReadDir(dir).Return([]fs.DirEntry{serverViewTestDirEntry{name: "100.itm"}}, nil)
	fileEditor.EXPECT().ReadDropFileData(filepath.Join(dir, "100.itm")).Return(dropfile.DropFile(nil), errors.New("parse failed"))
	service := &serverViewService{internalDB: internalDB, fileEditor: fileEditor}
	warnings := []string{}

	service.syncDrops(serverViewPaths{
		ByType: map[string]string{db.ServerViewServerTypeZone: "zone"},
	}, func(source string, err error) {
		warnings = append(warnings, source+": "+err.Error())
	})

	rows, err := internalDB.GetServerViewDropRows()
	require.NoError(t, err)
	require.Equal(t, []db.ServerViewDropRow{
		{NPCID: 100, RowIndex: 1, ItemID: 500, DropRate: 10, GroupCode: 2},
	}, stripDropAuditFields(rows))
	require.Len(t, warnings, 1)
	require.Contains(t, warnings[0], "parse failed")
}

func loadServerViewTestINI(t *testing.T, content string) *ini.File {
	t.Helper()

	cfg, err := ini.LoadSources(ini.LoadOptions{AllowShadows: true}, []byte(content))
	require.NoError(t, err)
	return cfg
}

func newServerViewTestDB(t *testing.T) db.InternalDB {
	t.Helper()

	log := logger.NewZerologLogger(zerolog.New(io.Discard), "test", zerolog.Disabled)
	internalDB := db.NewSQLiteDB(filepath.Join(t.TempDir(), "test.db"), log)
	require.NoError(t, internalDB.Connect())
	require.NoError(t, internalDB.MigrateUp())
	t.Cleanup(func() {
		require.NoError(t, internalDB.Close())
	})

	return internalDB
}

func stripMapZoneAuditFields(rows []db.ServerViewMapZone) []db.ServerViewMapZone {
	stripped := make([]db.ServerViewMapZone, 0, len(rows))
	for _, row := range rows {
		stripped = append(stripped, db.ServerViewMapZone{
			ServerType: row.ServerType,
			MapID:      row.MapID,
			ZoneID:     row.ZoneID,
		})
	}

	return stripped
}

func stripShopAuditFields(rows []db.ServerViewShopRow) []db.ServerViewShopRow {
	stripped := make([]db.ServerViewShopRow, 0, len(rows))
	for _, row := range rows {
		stripped = append(stripped, db.ServerViewShopRow{
			NPCID:      row.NPCID,
			LineNumber: row.LineNumber,
			ItemID:     row.ItemID,
		})
	}

	return stripped
}

func stripDropAuditFields(rows []db.ServerViewDropRow) []db.ServerViewDropRow {
	stripped := make([]db.ServerViewDropRow, 0, len(rows))
	for _, row := range rows {
		stripped = append(stripped, db.ServerViewDropRow{
			NPCID:     row.NPCID,
			RowIndex:  row.RowIndex,
			ItemID:    row.ItemID,
			DropRate:  row.DropRate,
			GroupCode: row.GroupCode,
		})
	}

	return stripped
}

type serverViewTestDirEntry struct {
	name string
}

func (entry serverViewTestDirEntry) Name() string {
	return entry.name
}

func (entry serverViewTestDirEntry) IsDir() bool {
	return false
}

func (entry serverViewTestDirEntry) Type() fs.FileMode {
	return 0
}

func (entry serverViewTestDirEntry) Info() (fs.FileInfo, error) {
	return nil, nil
}
