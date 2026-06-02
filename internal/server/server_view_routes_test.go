package server

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/omnihance/omnihance-a3-agent/internal/db"
	"github.com/omnihance/omnihance-a3-agent/internal/services"
	"github.com/stretchr/testify/require"
)

func TestServerViewOverviewRequiresViewGameData(t *testing.T) {
	server := &Server{serverViewService: &serverViewServiceStub{}}
	req := settingsRequest(http.MethodGet, "/api/server-view", nil, "guest")
	rr := httptest.NewRecorder()

	server.handleServerViewOverview(rr, req)

	require.Equal(t, http.StatusForbidden, rr.Code)
}

func TestServerViewOverviewBeforeFirstSyncReturnsParseableShape(t *testing.T) {
	server := &Server{serverViewService: &serverViewServiceStub{
		overview: &services.ServerViewOverview{
			Servers: []services.ServerViewServerInfo{
				{ServerType: db.ServerViewServerTypeMain, Sections: []services.ServerViewInfoSection{}},
				{ServerType: db.ServerViewServerTypeAccount, Sections: []services.ServerViewInfoSection{}},
				{ServerType: db.ServerViewServerTypeZone, Sections: []services.ServerViewInfoSection{}},
				{ServerType: db.ServerViewServerTypeBattle, Sections: []services.ServerViewInfoSection{}},
			},
			Sync: services.ServerViewSyncStatus{
				Warnings: []db.ServerViewSyncWarning{},
			},
		},
	}}
	req := settingsRequest(http.MethodGet, "/api/server-view", nil, "viewer")
	rr := httptest.NewRecorder()

	server.handleServerViewOverview(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	var response map[string]interface{}
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &response))
	servers, ok := response["servers"].([]interface{})
	require.True(t, ok)
	require.Len(t, servers, 4)
	for _, serverValue := range servers {
		serverObject, ok := serverValue.(map[string]interface{})
		require.True(t, ok)
		require.IsType(t, []interface{}{}, serverObject["sections"])
	}
	syncObject, ok := response["sync"].(map[string]interface{})
	require.True(t, ok)
	require.Nil(t, syncObject["latest_run"])
	require.IsType(t, []interface{}{}, syncObject["warnings"])
}

func TestServerViewInternalErrorDoesNotLeakDetails(t *testing.T) {
	server := &Server{serverViewService: &serverViewServiceStub{
		overviewErr: errors.New("database password leaked"),
	}}
	req := settingsRequest(http.MethodGet, "/api/server-view", nil, "viewer")
	rr := httptest.NewRecorder()

	server.handleServerViewOverview(rr, req)

	require.Equal(t, http.StatusInternalServerError, rr.Code)
	require.Contains(t, rr.Body.String(), "Internal server error")
	require.NotContains(t, rr.Body.String(), "database password leaked")
}

func TestServerViewStartSyncRequiresManageServer(t *testing.T) {
	server := &Server{serverViewService: &serverViewServiceStub{}}
	req := settingsRequest(http.MethodPost, "/api/server-view/sync", nil, "viewer")
	rr := httptest.NewRecorder()

	server.handleServerViewStartSync(rr, req)

	require.Equal(t, http.StatusForbidden, rr.Code)
}

func TestServerViewStartSyncReturnsConflictWhenRunning(t *testing.T) {
	server := &Server{serverViewService: &serverViewServiceStub{startSyncErr: services.ErrServerViewSyncRunning}}
	req := settingsRequest(http.MethodPost, "/api/server-view/sync", nil, "admin")
	rr := httptest.NewRecorder()

	server.handleServerViewStartSync(rr, req)

	require.Equal(t, http.StatusConflict, rr.Code)
	require.Contains(t, rr.Body.String(), "server view sync is already running")
}

func TestServerViewMapResponseShape(t *testing.T) {
	server := &Server{serverViewService: &serverViewServiceStub{
		mainMaps: []services.ServerViewMapRow{
			{MapID: 1, MapDisplay: "Temenos (1)", ZoneID: 2},
		},
	}}
	req := settingsRequest(http.MethodGet, "/api/server-view/main/maps?q=teme", nil, "viewer")
	rr := httptest.NewRecorder()

	server.handleServerViewMainMaps(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	require.JSONEq(t, `{"maps":[{"map_id":1,"map_name":null,"map_display":"Temenos (1)","zone_id":2}]}`, rr.Body.String())
}

func TestServerViewDropDetailsRejectsInvalidNPCID(t *testing.T) {
	server := &Server{serverViewService: &serverViewServiceStub{}}
	req := settingsRequest(http.MethodGet, "/api/server-view/zone/drops/bad", nil, "viewer")
	routeContext := chi.NewRouteContext()
	routeContext.URLParams.Add("npc_id", "bad")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, routeContext))
	rr := httptest.NewRecorder()

	server.handleServerViewZoneDropDetails(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
}

type serverViewServiceStub struct {
	startSyncErr error
	mainMaps     []services.ServerViewMapRow
	overview     *services.ServerViewOverview
	overviewErr  error
}

func (stub *serverViewServiceStub) Start() error {
	return nil
}

func (stub *serverViewServiceStub) GetOverview(ctx context.Context) (*services.ServerViewOverview, error) {
	_ = ctx
	if stub.overviewErr != nil {
		return nil, stub.overviewErr
	}
	if stub.overview != nil {
		return stub.overview, nil
	}

	return &services.ServerViewOverview{}, nil
}

func (stub *serverViewServiceStub) GetSyncStatus(ctx context.Context) (*services.ServerViewSyncStatus, error) {
	_ = ctx
	return &services.ServerViewSyncStatus{}, nil
}

func (stub *serverViewServiceStub) StartSync(ctx context.Context, userID *int64) (*db.ServerViewSyncRun, error) {
	_ = ctx
	_ = userID
	if stub.startSyncErr != nil {
		return nil, stub.startSyncErr
	}

	return &db.ServerViewSyncRun{ID: 1, Status: db.ServerViewSyncStatusRunning}, nil
}

func (stub *serverViewServiceStub) GetMainMaps(ctx context.Context, query string) ([]services.ServerViewMapRow, error) {
	_ = ctx
	_ = query
	return stub.mainMaps, nil
}

func (stub *serverViewServiceStub) GetZoneMaps(ctx context.Context, query string) ([]services.ServerViewMapRow, error) {
	_ = ctx
	_ = query
	return []services.ServerViewMapRow{}, nil
}

func (stub *serverViewServiceStub) GetZoneSpawns(ctx context.Context, mapQuery string, npcQuery string) ([]services.ServerViewSpawnSummaryRow, error) {
	_ = ctx
	_ = mapQuery
	_ = npcQuery
	return []services.ServerViewSpawnSummaryRow{}, nil
}

func (stub *serverViewServiceStub) GetZoneDrops(ctx context.Context, query string) ([]services.ServerViewDropSummaryRow, error) {
	_ = ctx
	_ = query
	return []services.ServerViewDropSummaryRow{}, nil
}

func (stub *serverViewServiceStub) GetZoneDropDetails(ctx context.Context, npcID int64, query string) ([]services.ServerViewDropDetailRow, error) {
	_ = ctx
	_ = npcID
	_ = query
	return []services.ServerViewDropDetailRow{}, nil
}

func (stub *serverViewServiceStub) GetZoneShops(ctx context.Context, query string) ([]services.ServerViewShopSummaryRow, error) {
	_ = ctx
	_ = query
	return []services.ServerViewShopSummaryRow{}, nil
}

func (stub *serverViewServiceStub) GetZoneShopDetails(ctx context.Context, npcID int64, query string) ([]services.ServerViewShopDetailRow, error) {
	_ = ctx
	_ = npcID
	_ = query
	return []services.ServerViewShopDetailRow{}, nil
}
