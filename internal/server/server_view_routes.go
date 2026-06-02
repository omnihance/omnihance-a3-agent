package server

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/omnihance/omnihance-a3-agent/internal/constants"
	"github.com/omnihance/omnihance-a3-agent/internal/logger"
	"github.com/omnihance/omnihance-a3-agent/internal/mw"
	"github.com/omnihance/omnihance-a3-agent/internal/permissions"
	"github.com/omnihance/omnihance-a3-agent/internal/services"
	"github.com/omnihance/omnihance-a3-agent/internal/utils"
)

const serverViewErrorContext = "server-view"

func (s *Server) InitializeServerViewRoutes(r *chi.Mux) {
	r.Route("/api/server-view", func(r chi.Router) {
		r.Use(mw.CheckCookie(s.internalDB, s.cfg.CookieSecret))
		r.Get("/", s.handleServerViewOverview)
		r.Get("/sync/status", s.handleServerViewSyncStatus)
		r.Post("/sync", s.handleServerViewStartSync)
		r.Get("/main/maps", s.handleServerViewMainMaps)
		r.Get("/zone/maps", s.handleServerViewZoneMaps)
		r.Get("/zone/spawns", s.handleServerViewZoneSpawns)
		r.Get("/zone/drops", s.handleServerViewZoneDrops)
		r.Get("/zone/drops/{npc_id}", s.handleServerViewZoneDropDetails)
		r.Get("/zone/shops", s.handleServerViewZoneShops)
		r.Get("/zone/shops/{npc_id}", s.handleServerViewZoneShopDetails)
	})
}

func (s *Server) handleServerViewOverview(w http.ResponseWriter, r *http.Request) {
	if !s.requireUserPermission(w, r, permissions.ActionViewGameData) {
		return
	}

	overview, err := s.serverViewService.GetOverview(r.Context())
	if err != nil {
		s.writeServerViewServiceError(w, err)
		return
	}

	_ = utils.WriteJSONResponse(w, overview)
}

func (s *Server) handleServerViewSyncStatus(w http.ResponseWriter, r *http.Request) {
	if !s.requireUserPermission(w, r, permissions.ActionViewGameData) {
		return
	}

	status, err := s.serverViewService.GetSyncStatus(r.Context())
	if err != nil {
		s.writeServerViewServiceError(w, err)
		return
	}

	_ = utils.WriteJSONResponse(w, status)
}

func (s *Server) handleServerViewStartSync(w http.ResponseWriter, r *http.Request) {
	if !s.requireUserPermission(w, r, permissions.ActionManageServer) {
		return
	}

	run, err := s.serverViewService.StartSync(r.Context(), serverViewUserID(r))
	if err != nil {
		s.writeServerViewServiceError(w, err)
		return
	}

	_ = utils.WriteJSONResponse(w, run)
}

func (s *Server) handleServerViewMainMaps(w http.ResponseWriter, r *http.Request) {
	if !s.requireUserPermission(w, r, permissions.ActionViewGameData) {
		return
	}

	rows, err := s.serverViewService.GetMainMaps(r.Context(), r.URL.Query().Get("q"))
	if err != nil {
		s.writeServerViewServiceError(w, err)
		return
	}

	_ = utils.WriteJSONResponse(w, map[string]interface{}{"maps": rows})
}

func (s *Server) handleServerViewZoneMaps(w http.ResponseWriter, r *http.Request) {
	if !s.requireUserPermission(w, r, permissions.ActionViewGameData) {
		return
	}

	rows, err := s.serverViewService.GetZoneMaps(r.Context(), r.URL.Query().Get("q"))
	if err != nil {
		s.writeServerViewServiceError(w, err)
		return
	}

	_ = utils.WriteJSONResponse(w, map[string]interface{}{"maps": rows})
}

func (s *Server) handleServerViewZoneSpawns(w http.ResponseWriter, r *http.Request) {
	if !s.requireUserPermission(w, r, permissions.ActionViewGameData) {
		return
	}

	rows, err := s.serverViewService.GetZoneSpawns(r.Context(), r.URL.Query().Get("map_q"), r.URL.Query().Get("npc_q"))
	if err != nil {
		s.writeServerViewServiceError(w, err)
		return
	}

	_ = utils.WriteJSONResponse(w, map[string]interface{}{"spawns": rows})
}

func (s *Server) handleServerViewZoneDrops(w http.ResponseWriter, r *http.Request) {
	if !s.requireUserPermission(w, r, permissions.ActionViewGameData) {
		return
	}

	rows, err := s.serverViewService.GetZoneDrops(r.Context(), r.URL.Query().Get("q"))
	if err != nil {
		s.writeServerViewServiceError(w, err)
		return
	}

	_ = utils.WriteJSONResponse(w, map[string]interface{}{"drops": rows})
}

func (s *Server) handleServerViewZoneDropDetails(w http.ResponseWriter, r *http.Request) {
	if !s.requireUserPermission(w, r, permissions.ActionViewGameData) {
		return
	}

	npcID, ok := serverViewNPCIDParam(w, r)
	if !ok {
		return
	}

	rows, err := s.serverViewService.GetZoneDropDetails(r.Context(), npcID, r.URL.Query().Get("q"))
	if err != nil {
		s.writeServerViewServiceError(w, err)
		return
	}

	_ = utils.WriteJSONResponse(w, map[string]interface{}{"drops": rows})
}

func (s *Server) handleServerViewZoneShops(w http.ResponseWriter, r *http.Request) {
	if !s.requireUserPermission(w, r, permissions.ActionViewGameData) {
		return
	}

	rows, err := s.serverViewService.GetZoneShops(r.Context(), r.URL.Query().Get("q"))
	if err != nil {
		s.writeServerViewServiceError(w, err)
		return
	}

	_ = utils.WriteJSONResponse(w, map[string]interface{}{"shops": rows})
}

func (s *Server) handleServerViewZoneShopDetails(w http.ResponseWriter, r *http.Request) {
	if !s.requireUserPermission(w, r, permissions.ActionViewGameData) {
		return
	}

	npcID, ok := serverViewNPCIDParam(w, r)
	if !ok {
		return
	}

	rows, err := s.serverViewService.GetZoneShopDetails(r.Context(), npcID, r.URL.Query().Get("q"))
	if err != nil {
		s.writeServerViewServiceError(w, err)
		return
	}

	_ = utils.WriteJSONResponse(w, map[string]interface{}{"items": rows})
}

func (s *Server) writeServerViewServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, services.ErrServerViewSyncRunning):
		writeServerViewError(w, http.StatusConflict, constants.ErrorCodeBadRequest, err.Error())
	default:
		if s.log != nil {
			s.log.Error("server view request failed", logger.Field{Key: "error", Value: err})
		}

		writeServerViewError(w, http.StatusInternalServerError, constants.ErrorCodeInternalServerError, "Internal server error")
	}
}

func writeServerViewError(w http.ResponseWriter, status int, errorCode string, message string) {
	_ = utils.WriteJSONResponseWithStatus(w, status, map[string]interface{}{
		"errorCode": errorCode,
		"context":   serverViewErrorContext,
		"errors":    []string{message},
	})
}

func serverViewNPCIDParam(w http.ResponseWriter, r *http.Request) (int64, bool) {
	npcID, err := strconv.ParseInt(chi.URLParam(r, "npc_id"), 10, 64)
	if err != nil {
		writeServerViewError(w, http.StatusBadRequest, constants.ErrorCodeBadRequest, "Invalid NPC ID")
		return 0, false
	}

	return npcID, true
}

func serverViewUserID(r *http.Request) *int64 {
	userID, ok := utils.GetUserIdFromContext(r.Context())
	if !ok {
		return nil
	}

	return &userID
}
