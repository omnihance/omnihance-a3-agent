package db

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestServerViewReplaceRowsReplacesOnlyMatchingSource(t *testing.T) {
	internalDB := newItemClientDataTestDB(t)

	require.NoError(t, internalDB.ReplaceServerViewMapZones(ServerViewServerTypeMain, []ServerViewMapZone{
		{ServerType: ServerViewServerTypeMain, MapID: 1, ZoneID: 10},
		{ServerType: ServerViewServerTypeMain, MapID: 2, ZoneID: 20},
	}))
	require.NoError(t, internalDB.ReplaceServerViewMapZones(ServerViewServerTypeZone, []ServerViewMapZone{
		{ServerType: ServerViewServerTypeZone, MapID: 1, ZoneID: 30},
	}))
	require.NoError(t, internalDB.ReplaceServerViewMapZones(ServerViewServerTypeMain, []ServerViewMapZone{
		{ServerType: ServerViewServerTypeMain, MapID: 3, ZoneID: 40},
	}))

	mainRows, err := internalDB.GetServerViewMapZones(ServerViewServerTypeMain)
	require.NoError(t, err)
	require.Equal(t, []ServerViewMapZone{
		{ServerType: ServerViewServerTypeMain, MapID: 3, ZoneID: 40},
	}, stripServerViewMapZoneAuditFields(mainRows))

	zoneRows, err := internalDB.GetServerViewMapZones(ServerViewServerTypeZone)
	require.NoError(t, err)
	require.Equal(t, []ServerViewMapZone{
		{ServerType: ServerViewServerTypeZone, MapID: 1, ZoneID: 30},
	}, stripServerViewMapZoneAuditFields(zoneRows))
}

func TestServerViewSyncRunStatusAndWarnings(t *testing.T) {
	internalDB := newItemClientDataTestDB(t)

	run, err := internalDB.CreateServerViewSyncRun(nil)
	require.NoError(t, err)
	require.Equal(t, ServerViewSyncStatusRunning, run.Status)

	require.NoError(t, internalDB.AddServerViewSyncWarning(run.ID, "MapInfo.ini", "invalid map row"))
	require.NoError(t, internalDB.FinishServerViewSyncRun(run.ID, ServerViewSyncStatusSucceeded, 1, nil))

	latest, err := internalDB.GetLatestServerViewSyncRun()
	require.NoError(t, err)
	require.NotNil(t, latest)
	require.Equal(t, run.ID, latest.ID)
	require.Equal(t, ServerViewSyncStatusSucceeded, latest.Status)
	require.NotNil(t, latest.FinishedAt)
	require.Equal(t, 1, latest.WarningCount)

	warnings, err := internalDB.GetServerViewSyncWarnings(run.ID)
	require.NoError(t, err)
	require.Len(t, warnings, 1)
	require.Equal(t, "MapInfo.ini", warnings[0].Source)
	require.Equal(t, "invalid map row", warnings[0].Message)
}

func TestServerViewMarkOrphanedSyncRunsFailed(t *testing.T) {
	internalDB := newItemClientDataTestDB(t)
	run, err := internalDB.CreateServerViewSyncRun(nil)
	require.NoError(t, err)

	require.NoError(t, internalDB.MarkOrphanedServerViewSyncRunsFailed())

	running, err := internalDB.GetRunningServerViewSyncRun()
	require.NoError(t, err)
	require.Nil(t, running)

	latest, err := internalDB.GetLatestServerViewSyncRun()
	require.NoError(t, err)
	require.NotNil(t, latest)
	require.Equal(t, run.ID, latest.ID)
	require.Equal(t, ServerViewSyncStatusFailed, latest.Status)
	require.NotNil(t, latest.ErrorDetails)
}

func stripServerViewMapZoneAuditFields(rows []ServerViewMapZone) []ServerViewMapZone {
	stripped := make([]ServerViewMapZone, 0, len(rows))
	for _, row := range rows {
		stripped = append(stripped, ServerViewMapZone{
			ServerType: row.ServerType,
			MapID:      row.MapID,
			ZoneID:     row.ZoneID,
		})
	}

	return stripped
}
