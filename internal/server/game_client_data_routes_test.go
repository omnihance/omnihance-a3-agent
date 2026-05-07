package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/omnihance/omnihance-a3-agent/internal/db"
	"github.com/omnihance/omnihance-a3-agent/internal/logger"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
)

func TestHandleGameClientDataCounts(t *testing.T) {
	tests := []struct {
		name           string
		monsters       []db.MonsterClientData
		maps           []db.MapClientData
		closeDB        bool
		expectedStatus int
		expectedBody   *GameClientDataCountsResponse
	}{
		{
			name:           "empty tables",
			expectedStatus: http.StatusOK,
			expectedBody: &GameClientDataCountsResponse{
				Monsters: 0,
				Maps:     0,
			},
		},
		{
			name:           "populated tables",
			monsters:       []db.MonsterClientData{{ID: 1, Name: "Wolf"}, {ID: 2, Name: "Bear"}},
			maps:           []db.MapClientData{{ID: 1, Name: "Temoz"}, {ID: 2, Name: "Quanato"}, {ID: 3, Name: "Hatrel"}},
			expectedStatus: http.StatusOK,
			expectedBody: &GameClientDataCountsResponse{
				Monsters: 2,
				Maps:     3,
			},
		},
		{
			name:           "database error",
			closeDB:        true,
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			internalDB := newTestInternalDB(t)
			require.NoError(t, internalDB.BulkReplaceMonsterClientData(test.monsters))
			require.NoError(t, internalDB.BulkReplaceMapClientData(test.maps))
			if test.closeDB {
				require.NoError(t, internalDB.Close())
			}

			server := &Server{internalDB: internalDB}
			req := httptest.NewRequest(http.MethodGet, "/api/game-client-data/counts", nil)
			rr := httptest.NewRecorder()

			server.handleGameClientDataCounts(rr, req)

			require.Equal(t, test.expectedStatus, rr.Code)
			if test.expectedBody == nil {
				return
			}

			var body GameClientDataCountsResponse
			require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &body))
			require.Equal(t, *test.expectedBody, body)
		})
	}
}

func newTestInternalDB(t *testing.T) db.InternalDB {
	t.Helper()

	log := logger.NewZerologLogger(zerolog.Nop(), "test", zerolog.Disabled)
	internalDB := db.NewSQLiteDB(filepath.Join(t.TempDir(), "test.db"), log)
	require.NoError(t, internalDB.Connect())
	require.NoError(t, internalDB.MigrateUp())
	t.Cleanup(func() {
		_ = internalDB.Close()
	})

	return internalDB
}
