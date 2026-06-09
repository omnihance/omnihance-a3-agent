package db

import (
	"io"
	"path/filepath"
	"testing"
	"time"

	"github.com/doug-martin/goqu/v9"
	"github.com/omnihance/omnihance-a3-agent/internal/constants"
	"github.com/omnihance/omnihance-a3-agent/internal/logger"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
)

func TestFileDownloadLinksReuseExpiryAndChangedFingerprint(t *testing.T) {
	internalDB := newFileDownloadTestDB(t)
	user, err := internalDB.CreateUser("download-admin@example.com", "password", constants.RoleAdmin, nil)
	require.NoError(t, err)

	now := time.Now().UTC()
	payload := testFileDownloadPayload(user.ID, "file-hash", now.Add(time.Hour))
	link, err := internalDB.CreateFileDownloadLink(payload)
	require.NoError(t, err)
	require.Equal(t, int64(0), link.DownloadCount)

	reusable, err := internalDB.GetReusableFileDownloadLink(payload, now)
	require.NoError(t, err)
	require.NotNil(t, reusable)
	require.Equal(t, link.ID, reusable.ID)

	reusable, err = internalDB.GetReusableFileDownloadLink(payload, now.Add(2*time.Hour))
	require.NoError(t, err)
	require.Nil(t, reusable)

	changedPayload := payload
	changedPayload.FileHash = "changed-hash"
	reusable, err = internalDB.GetReusableFileDownloadLink(changedPayload, now)
	require.NoError(t, err)
	require.Nil(t, reusable)
}

func TestRecordFileDownloadIncrementsCountAndStoresEvent(t *testing.T) {
	internalDB := newFileDownloadTestDB(t)
	sqliteDB := internalDB.(*sqliteInternalDB)
	user, err := internalDB.CreateUser("download-event-admin@example.com", "password", constants.RoleAdmin, nil)
	require.NoError(t, err)

	link, err := internalDB.CreateFileDownloadLink(testFileDownloadPayload(user.ID, "file-hash", time.Now().UTC().Add(time.Hour)))
	require.NoError(t, err)

	userAgent := "test-agent"
	ipAddress := "127.0.0.1"
	require.NoError(t, internalDB.RecordFileDownload(link, user.ID, &userAgent, &ipAddress))

	updatedLink, err := internalDB.GetFileDownloadLinkByPublicID(link.PublicID)
	require.NoError(t, err)
	require.Equal(t, int64(1), updatedLink.DownloadCount)
	require.NotNil(t, updatedLink.LastDownloadedAt)

	var eventCount int64
	_, err = sqliteDB.goqu.From("file_download_events").
		Where(goqu.Ex{
			"link_id":       link.ID,
			"user_id":       user.ID,
			"user_agent":    userAgent,
			"ip_address":    ipAddress,
			"source_type":   FileDownloadSourceFileBrowser,
			"original_path": link.OriginalPath,
		}).
		Select(goqu.COUNT("*")).
		ScanVal(&eventCount)
	require.NoError(t, err)
	require.Equal(t, int64(1), eventCount)
}

func newFileDownloadTestDB(t *testing.T) InternalDB {
	t.Helper()

	log := logger.NewZerologLogger(zerolog.New(io.Discard), "test", zerolog.Disabled)
	internalDB := NewSQLiteDB(filepath.Join(t.TempDir(), "test.db"), log)
	require.NoError(t, internalDB.Connect())
	require.NoError(t, internalDB.MigrateUp())
	t.Cleanup(func() {
		require.NoError(t, internalDB.Close())
	})

	return internalDB
}

func testFileDownloadPayload(userID int64, fileHash string, expiresAt time.Time) FileDownloadLinkPayload {
	return FileDownloadLinkPayload{
		PublicID:       "public-" + fileHash + "-" + expiresAt.Format("150405"),
		UserID:         userID,
		FileID:         "file-id",
		SourceType:     FileDownloadSourceFileBrowser,
		OriginalPath:   filepath.Clean("C:/a3/server/test.dat"),
		FileName:       "test.dat",
		FileSize:       4,
		FileHash:       fileHash,
		FileModifiedAt: 10,
		ExpiresAt:      expiresAt,
	}
}
