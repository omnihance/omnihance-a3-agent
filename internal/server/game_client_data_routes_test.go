package server

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/omnihance/omnihance-a3-agent/internal/config"
	"github.com/omnihance/omnihance-a3-agent/internal/constants"
	"github.com/omnihance/omnihance-a3-agent/internal/db"
	"github.com/omnihance/omnihance-a3-agent/internal/logger"
	"github.com/omnihance/omnihance-a3-agent/internal/services"
	"github.com/omnihance/omnihance-a3-agent/internal/utils"
	"github.com/project-agonyl/agonyl-utils-go/itemfile"
	agonylUtils "github.com/project-agonyl/agonyl-utils-go/utils"
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
				Items:    db.ItemClientDataCounts{},
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
				Items:    db.ItemClientDataCounts{},
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

func TestHandleUploadIT1File(t *testing.T) {
	internalDB := newTestInternalDB(t)
	_, err := internalDB.CreateUser("admin@example.com", "password", constants.RoleAdmin, nil)
	require.NoError(t, err)
	log := logger.NewZerologLogger(zerolog.Nop(), "test", zerolog.Disabled)
	server := &Server{
		cfg:        &config.EnvVars{MaxFileUploadSizeMb: 1},
		internalDB: internalDB,
		fileEditor: services.NewFileEditorService(log),
	}

	data := itemfile.IT1File{{Type: 4, Row: 0, NPCPrice: 200}}
	copy(data[0].Name[:], "Potion")

	var itemFile bytes.Buffer
	require.NoError(t, itemfile.WriteIT1(&itemFile, data))
	encoded := append([]byte(nil), itemFile.Bytes()...)
	agonylUtils.EncodeULL(encoded, len(encoded))

	req := newGameClientDataUploadRequest(t, "/api/game-client-data/upload-it1-file", "IT1.ull", encoded)
	req = req.WithContext(gameClientDataUploadContext(req.Context()))
	rr := httptest.NewRecorder()

	server.handleUploadIT1File(rr, req)

	require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())

	var response map[string]interface{}
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &response))
	require.Equal(t, "IT1 item file uploaded successfully", response["message"])
	require.Equal(t, float64(1), response["count"])

	items, err := internalDB.GetAllItemClientData("")
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.Equal(t, int64(4096), items[0].ID)
	require.Equal(t, "Potion", items[0].Name)
	require.Equal(t, string(db.ItemClientDataTypeIT1), items[0].ItemType)
}

func TestHandleUploadIT1FileParseError(t *testing.T) {
	internalDB := newTestInternalDB(t)
	log := logger.NewZerologLogger(zerolog.Nop(), "test", zerolog.Disabled)
	server := &Server{
		cfg:        &config.EnvVars{MaxFileUploadSizeMb: 1},
		internalDB: internalDB,
		fileEditor: services.NewFileEditorService(log),
	}

	req := newGameClientDataUploadRequest(t, "/api/game-client-data/upload-it1-file", "IT1.ull", []byte{1, 2, 3})
	req = req.WithContext(gameClientDataUploadContext(req.Context()))
	rr := httptest.NewRecorder()

	server.handleUploadIT1File(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
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

func newGameClientDataUploadRequest(t *testing.T, target string, fileName string, data []byte) *http.Request {
	t.Helper()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", fileName)
	require.NoError(t, err)
	_, err = part.Write(data)
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	req := httptest.NewRequest(http.MethodPost, target, &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	return req
}

func gameClientDataUploadContext(ctx context.Context) context.Context {
	ctx = utils.SetUserIdInContext(ctx, 1)
	return utils.SetUserRolesInContext(ctx, []string{constants.RoleAdmin})
}
