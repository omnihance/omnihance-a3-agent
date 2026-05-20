package db

import (
	"io"
	"path/filepath"
	"testing"

	"github.com/omnihance/omnihance-a3-agent/internal/logger"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
)

func TestConnectConfiguresSingleSQLiteConnection(t *testing.T) {
	log := logger.NewZerologLogger(zerolog.New(io.Discard), "test", zerolog.Disabled)
	database := NewSQLiteDB(filepath.Join(t.TempDir(), "test.db"), log)
	sqliteDB, ok := database.(*sqliteInternalDB)
	require.True(t, ok)

	require.NoError(t, sqliteDB.Connect())
	t.Cleanup(func() {
		require.NoError(t, sqliteDB.Close())
	})

	require.Equal(t, sqliteMaxOpenConnections, sqliteDB.db.Stats().MaxOpenConnections)
}
