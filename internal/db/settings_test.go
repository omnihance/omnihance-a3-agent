package db

import (
	"errors"
	"testing"

	"github.com/omnihance/omnihance-a3-agent/internal/constants"
	"github.com/stretchr/testify/require"
)

func TestCreateSetting(t *testing.T) {
	internalDB := newItemClientDataTestDB(t)
	user, err := internalDB.CreateUser("admin@example.com", "password", constants.RoleAdmin, nil)
	require.NoError(t, err)

	setting, err := internalDB.CreateSetting(constants.SettingKeyDBHost, "127.0.0.1", &user.ID)
	require.NoError(t, err)
	require.Equal(t, constants.SettingKeyDBHost, setting.Key)
	require.Equal(t, "127.0.0.1", setting.Value)
	require.NotNil(t, setting.CreatedBy)
	require.Equal(t, user.ID, *setting.CreatedBy)
}

func TestCreateSettingRejectsDuplicate(t *testing.T) {
	internalDB := newItemClientDataTestDB(t)

	_, err := internalDB.CreateSetting(constants.SettingKeyDBHost, "127.0.0.1", nil)
	require.NoError(t, err)

	_, err = internalDB.CreateSetting(constants.SettingKeyDBHost, "localhost", nil)
	require.Error(t, err)
	require.True(t, errors.Is(err, ErrSettingAlreadyExists))
}

func TestUpdateSettingChangesValue(t *testing.T) {
	internalDB := newItemClientDataTestDB(t)
	user, err := internalDB.CreateUser("admin@example.com", "password", constants.RoleAdmin, nil)
	require.NoError(t, err)

	_, err = internalDB.CreateSetting(constants.SettingKeyDBUser, "sa", nil)
	require.NoError(t, err)

	setting, err := internalDB.UpdateSetting(constants.SettingKeyDBUser, "a3_user", &user.ID)
	require.NoError(t, err)
	require.Equal(t, constants.SettingKeyDBUser, setting.Key)
	require.Equal(t, "a3_user", setting.Value)
	require.NotNil(t, setting.UpdatedBy)
	require.Equal(t, user.ID, *setting.UpdatedBy)
}

func TestUpdateMissingSettingReturnsNotFound(t *testing.T) {
	internalDB := newItemClientDataTestDB(t)

	_, err := internalDB.UpdateSetting(constants.SettingKeyDBPass, "secret", nil)
	require.Error(t, err)
	require.True(t, errors.Is(err, ErrSettingNotFound))
}

func TestDeleteMissingSettingReturnsNotFound(t *testing.T) {
	internalDB := newItemClientDataTestDB(t)

	err := internalDB.DeleteSetting(constants.SettingKeyDBPass)
	require.Error(t, err)
	require.True(t, errors.Is(err, ErrSettingNotFound))
}
