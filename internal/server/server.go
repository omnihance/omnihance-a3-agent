package server

import (
	"embed"
	"fmt"
	"net/http"
	"time"

	"github.com/omnihance/omnihance-a3-agent/internal/config"
	"github.com/omnihance/omnihance-a3-agent/internal/db"
	"github.com/omnihance/omnihance-a3-agent/internal/logger"
	"github.com/omnihance/omnihance-a3-agent/internal/services"
)

type Server struct {
	cfg                  *config.EnvVars
	log                  logger.Logger
	frontendFiles        embed.FS
	docsFiles            embed.FS
	version              string
	internalDB           db.InternalDB
	fileEditor           services.FileEditorService
	processService       services.ProcessService
	serverManagerService services.ServerManagerService
	versionChecker       services.VersionCheckerService
	backupService        services.BackupService
	serverViewService    services.ServerViewService
	uploadManager        *fileUploadManager
}

func NewServer(
	cfg *config.EnvVars,
	log logger.Logger,
	frontendFiles embed.FS,
	docsFiles embed.FS,
	version string,
	internalDB db.InternalDB,
	fileEditor services.FileEditorService,
	processService services.ProcessService,
	serverManagerService services.ServerManagerService,
	versionChecker services.VersionCheckerService,
	backupService services.BackupService,
	serverViewService services.ServerViewService,
) *http.Server {
	newServer := &Server{
		cfg:                  cfg,
		log:                  log,
		frontendFiles:        frontendFiles,
		docsFiles:            docsFiles,
		version:              version,
		internalDB:           internalDB,
		fileEditor:           fileEditor,
		processService:       processService,
		serverManagerService: serverManagerService,
		versionChecker:       versionChecker,
		backupService:        backupService,
		serverViewService:    serverViewService,
	}

	newServer.uploadManager = newFileUploadManager(cfg.RevisionsDirectory, fileEditor, log, cfg.MaxFileUploadSizeBytes())
	if err := newServer.uploadManager.Start(); err != nil && log != nil {
		log.Error("Could not start file upload manager", logger.Field{Key: "error", Value: err})
	}

	server := &http.Server{
		Addr:              fmt.Sprintf(":%s", newServer.cfg.Port),
		Handler:           newServer.RegisterRoutes(),
		IdleTimeout:       8 * time.Minute,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		ReadHeaderTimeout: 9 * time.Minute,
	}
	server.RegisterOnShutdown(newServer.uploadManager.Stop)

	return server
}
