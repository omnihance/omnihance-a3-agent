package server

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/omnihance/omnihance-a3-agent/internal/constants"
	"github.com/stretchr/testify/require"
)

func TestHandleUpdateUserRolesUpdatesRole(t *testing.T) {
	internalDB := newTestInternalDB(t)
	actor, err := internalDB.CreateUser("actor@example.com", "password", constants.RoleSuperAdmin, nil)
	require.NoError(t, err)
	target, err := internalDB.CreateUser("target@example.com", "password", constants.RoleUser, nil)
	require.NoError(t, err)
	server := &Server{internalDB: internalDB}
	body := bytes.NewBufferString(`{"role":"admin"}`)
	req := settingsRequest(http.MethodPatch, "/api/users/"+strconv.FormatInt(target.ID, 10)+"/roles", body, constants.RoleSuperAdmin)
	req = withURLParam(req, "id", strconv.FormatInt(target.ID, 10))
	rr := httptest.NewRecorder()

	server.handleUpdateUserRoles(rr, req)

	require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())
	updated, err := internalDB.GetUserByID(target.ID)
	require.NoError(t, err)
	require.Equal(t, constants.RoleAdmin, updated.Roles)
	require.NotNil(t, updated.UpdatedBy)
	require.Equal(t, actor.ID, *updated.UpdatedBy)
}

func TestHandleUpdateUserRolesRejectsSuperAdminRole(t *testing.T) {
	internalDB := newTestInternalDB(t)
	_, err := internalDB.CreateUser("actor@example.com", "password", constants.RoleSuperAdmin, nil)
	require.NoError(t, err)
	target, err := internalDB.CreateUser("target@example.com", "password", constants.RoleUser, nil)
	require.NoError(t, err)
	server := &Server{internalDB: internalDB}
	body := bytes.NewBufferString(`{"role":"super_admin"}`)
	req := settingsRequest(http.MethodPatch, "/api/users/"+strconv.FormatInt(target.ID, 10)+"/roles", body, constants.RoleSuperAdmin)
	req = withURLParam(req, "id", strconv.FormatInt(target.ID, 10))
	rr := httptest.NewRecorder()

	server.handleUpdateUserRoles(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code, rr.Body.String())
	updated, err := internalDB.GetUserByID(target.ID)
	require.NoError(t, err)
	require.Equal(t, constants.RoleUser, updated.Roles)
}

func TestHandleUpdateUserRolesRejectsSuperAdminUser(t *testing.T) {
	internalDB := newTestInternalDB(t)
	_, err := internalDB.CreateUser("actor@example.com", "password", constants.RoleSuperAdmin, nil)
	require.NoError(t, err)
	target, err := internalDB.CreateUser("target@example.com", "password", constants.RoleSuperAdmin, nil)
	require.NoError(t, err)
	server := &Server{internalDB: internalDB}
	body := bytes.NewBufferString(`{"role":"admin"}`)
	req := settingsRequest(http.MethodPatch, "/api/users/"+strconv.FormatInt(target.ID, 10)+"/roles", body, constants.RoleSuperAdmin)
	req = withURLParam(req, "id", strconv.FormatInt(target.ID, 10))
	rr := httptest.NewRecorder()

	server.handleUpdateUserRoles(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code, rr.Body.String())
	updated, err := internalDB.GetUserByID(target.ID)
	require.NoError(t, err)
	require.Equal(t, constants.RoleSuperAdmin, updated.Roles)
}
