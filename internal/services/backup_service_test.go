package services

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/omnihance/omnihance-a3-agent/internal/config"
	"github.com/omnihance/omnihance-a3-agent/internal/db"
	"github.com/omnihance/omnihance-a3-agent/internal/logger"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
)

func TestPrepareDirectoryDownloadCreatesTaggedJobAndReusesCache(t *testing.T) {
	service, internalDB := newBackupServiceForTest(t)
	sourceDir := writeDirectoryDownloadSource(t, "server", "maps/temoz.map", []byte("map data"))

	result, err := service.PrepareDirectoryDownload(context.Background(), sourceDir, nil)
	require.NoError(t, err)
	require.Equal(t, DirectoryDownloadStatusStarted, result.Status)
	require.Contains(t, result.Message, "do not refresh")

	ready := waitDirectoryDownloadReady(t, service, result.RunID)
	require.Equal(t, DirectoryDownloadStatusReady, ready.Status)
	require.NotEmpty(t, ready.ArchivePath)

	job, err := internalDB.GetBackupJob(result.JobID)
	require.NoError(t, err)
	require.NotNil(t, job.Tag)
	require.Equal(t, db.BackupJobTagDirectoryDownload, *job.Tag)
	require.Nil(t, job.CronExpression)
	require.Equal(t, filepath.Clean(sourceDir), *job.SourcePath)

	fingerprint, err := service.buildDirectoryDownloadFingerprint(context.Background(), filepath.Clean(sourceDir))
	require.NoError(t, err)
	archive, err := internalDB.GetDirectoryDownloadArchive(filepath.Clean(sourceDir), fingerprint)
	require.NoError(t, err)
	require.Equal(t, ready.ArchivePath, archive.ArchivePath)

	reused, err := service.PrepareDirectoryDownload(context.Background(), sourceDir, nil)
	require.NoError(t, err)
	require.Equal(t, DirectoryDownloadStatusReady, reused.Status)
	require.True(t, reused.ArchiveReused)
	require.Equal(t, ready.ArchivePath, reused.ArchivePath)
}

func TestPrepareDirectoryDownloadResumesSameRunningJobAndRejectsDifferentDirectory(t *testing.T) {
	service, internalDB := newBackupServiceForTest(t)
	sourceDir := writeDirectoryDownloadSource(t, "server", "maps/temoz.map", []byte("map data"))
	otherDir := writeDirectoryDownloadSource(t, "other", "maps/quanato.map", []byte("map data"))

	tag := db.BackupJobTagDirectoryDownload
	sourcePath := filepath.Clean(sourceDir)
	job, err := internalDB.CreateBackupJob(db.BackupJobPayload{
		JobType:              db.BackupJobTypeFile,
		Tag:                  &tag,
		Name:                 "Directory download: server",
		Status:               db.BackupJobStatusActive,
		DestinationDirectory: service.directoryDownloadsDirectory(),
		SourcePath:           &sourcePath,
	}, nil)
	require.NoError(t, err)

	run, err := internalDB.CreateBackupRun(job.ID, db.BackupRunTriggerDirectoryDownload, db.BackupJobStatusActive, nil)
	require.NoError(t, err)

	result, err := service.PrepareDirectoryDownload(context.Background(), sourceDir, nil)
	require.NoError(t, err)
	require.Equal(t, DirectoryDownloadStatusInProgress, result.Status)
	require.Equal(t, run.ID, result.RunID)

	_, err = service.PrepareDirectoryDownload(context.Background(), otherDir, nil)
	require.ErrorIs(t, err, ErrDirectoryDownloadConflict)
}

func TestPrepareDirectoryDownloadReusesCacheWhenDifferentDirectoryIsRunning(t *testing.T) {
	service, internalDB := newBackupServiceForTest(t)
	sourceDir := writeDirectoryDownloadSource(t, "server", "maps/temoz.map", []byte("map data"))
	otherDir := writeDirectoryDownloadSource(t, "other", "maps/quanato.map", []byte("map data"))
	normalizedPath := filepath.Clean(sourceDir)

	tag := db.BackupJobTagDirectoryDownload
	sourcePath := normalizedPath
	job, err := internalDB.CreateBackupJob(db.BackupJobPayload{
		JobType:              db.BackupJobTypeFile,
		Tag:                  &tag,
		Name:                 "Directory download: server",
		Status:               db.BackupJobStatusActive,
		DestinationDirectory: service.directoryDownloadsDirectory(),
		SourcePath:           &sourcePath,
	}, nil)
	require.NoError(t, err)

	run, err := internalDB.CreateBackupRun(job.ID, db.BackupRunTriggerDirectoryDownload, db.BackupJobStatusActive, nil)
	require.NoError(t, err)

	archivePath := filepath.Join(service.directoryDownloadsDirectory(), "server.zip")
	require.NoError(t, os.MkdirAll(filepath.Dir(archivePath), 0755))
	require.NoError(t, os.WriteFile(archivePath, []byte("zip"), 0600))

	file, err := internalDB.CreateBackupRunFile(run.ID, "server", archivePath, 3)
	require.NoError(t, err)

	otherSourcePath := filepath.Clean(otherDir)
	otherJob, err := internalDB.CreateBackupJob(db.BackupJobPayload{
		JobType:              db.BackupJobTypeFile,
		Tag:                  &tag,
		Name:                 "Directory download: other",
		Status:               db.BackupJobStatusActive,
		DestinationDirectory: service.directoryDownloadsDirectory(),
		SourcePath:           &otherSourcePath,
	}, nil)
	require.NoError(t, err)

	_, err = internalDB.CreateBackupRun(otherJob.ID, db.BackupRunTriggerDirectoryDownload, db.BackupJobStatusActive, nil)
	require.NoError(t, err)

	fingerprint := stableDirectoryDownloadFingerprint(t, service, normalizedPath)

	_, err = internalDB.UpsertDirectoryDownloadArchive(db.DirectoryDownloadArchivePayload{
		NormalizedPath:    normalizedPath,
		SourceFingerprint: fingerprint,
		JobID:             job.ID,
		RunID:             run.ID,
		FileID:            file.ID,
		ArchivePath:       archivePath,
		ArchiveSize:       3,
	})
	require.NoError(t, err)

	archive, err := internalDB.GetDirectoryDownloadArchive(normalizedPath, fingerprint)
	require.NoError(t, err)
	require.NotNil(t, service.directoryDownloadArchiveResult(archive))

	result, err := service.PrepareDirectoryDownload(context.Background(), sourceDir, nil)
	require.NoError(t, err)
	require.Equal(t, DirectoryDownloadStatusReady, result.Status)
	require.True(t, result.ArchiveReused)
	require.Equal(t, archivePath, result.ArchivePath)
}

func newBackupServiceForTest(t *testing.T) (*backupService, db.InternalDB) {
	t.Helper()

	log := logger.NewZerologLogger(zerolog.New(io.Discard), "test", zerolog.Disabled)
	internalDB := db.NewSQLiteDB(filepath.Join(t.TempDir(), "test.db"), log)
	require.NoError(t, internalDB.Connect())
	require.NoError(t, internalDB.MigrateUp())
	t.Cleanup(func() {
		require.NoError(t, internalDB.Close())
	})

	baseDir := t.TempDir()
	service := NewBackupService(
		&config.EnvVars{
			BackupsDirectory:            filepath.Join(baseDir, "backups"),
			DirectoryDownloadsDirectory: filepath.Join(baseDir, "directory-downloads"),
		},
		log,
		internalDB,
		NewFileEditorService(log),
	).(*backupService)

	return service, internalDB
}

func writeDirectoryDownloadSource(t *testing.T, name string, fileName string, content []byte) string {
	t.Helper()

	dir := filepath.Join(t.TempDir(), name)
	path := filepath.Join(dir, fileName)
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0755))
	require.NoError(t, os.WriteFile(path, content, 0600))
	return dir
}

func waitDirectoryDownloadReady(t *testing.T, service *backupService, runID int64) *DirectoryDownloadResult {
	t.Helper()

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		result, err := service.GetDirectoryDownloadStatus(context.Background(), runID, nil)
		require.NoError(t, err)

		switch result.Status {
		case DirectoryDownloadStatusReady:
			return result
		case DirectoryDownloadStatusFailed, DirectoryDownloadStatusCancelled:
			t.Fatalf("directory download ended with %s: %s", result.Status, result.Message)
		}

		time.Sleep(25 * time.Millisecond)
	}

	t.Fatalf("directory download did not finish")
	return nil
}

func stableDirectoryDownloadFingerprint(t *testing.T, service *backupService, sourcePath string) string {
	t.Helper()

	var previous string
	for range 10 {
		current, err := service.buildDirectoryDownloadFingerprint(context.Background(), sourcePath)
		require.NoError(t, err)

		if current == previous {
			return current
		}

		previous = current
		time.Sleep(10 * time.Millisecond)
	}

	return previous
}
