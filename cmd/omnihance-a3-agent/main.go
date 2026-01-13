package main

import (
	"context"
	"embed"
	"os"
	"os/signal"
	"syscall"

	"github.com/omnihance/omnihance-a3-agent/internal/config"
	"github.com/omnihance/omnihance-a3-agent/internal/db"
	"github.com/omnihance/omnihance-a3-agent/internal/logger"
	"github.com/omnihance/omnihance-a3-agent/internal/server"
	"github.com/omnihance/omnihance-a3-agent/internal/services"
)

//go:embed omnihance-a3-agent-ui/dist/*
var frontendFiles embed.FS

//go:embed docs/*
var docsFiles embed.FS

var version = "dev"

func main() {
	cfg := config.New()
	log := logger.NewZerologFileLogger("omnihance-a3-agent", cfg.LogDir, cfg.GetLogLevel())
	defer func() {
		_ = log.Close()
	}()

	internalDB := db.NewSQLiteDB(cfg.DatabaseURL, log)
	if err := internalDB.Connect(); err != nil {
		log.Error("Could not connect to internal database", logger.Field{Key: "error", Value: err})
		os.Exit(1)
	}

	defer func() {
		_ = internalDB.Close()
	}()

	if err := internalDB.MigrateUp(); err != nil {
		log.Error("Could not migrate internal database", logger.Field{Key: "error", Value: err})
		os.Exit(1)
	}

	_ = internalDB.SetDefaultSettings()
	if cfg.MetricsEnabled {
		metricsCollector := services.NewMetricsCollectorService(cfg, log, internalDB)
		if err := metricsCollector.Start(); err != nil {
			log.Error("Could not start metrics collector service", logger.Field{Key: "error", Value: err})
			os.Exit(1)
		}

		defer func() {
			_ = metricsCollector.Stop()
		}()
	}

	log.Info(
		"Starting Omnihance A3 Agent on port "+cfg.Port,
		logger.Field{Key: "port", Value: cfg.Port},
		logger.Field{Key: "log_level", Value: cfg.GetLogLevel().String()},
		logger.Field{Key: "version", Value: version},
	)

	fileEditor := services.NewFileEditorService(log)
	processService := services.NewProcessService(log)
	serverManagerService := services.NewServerManagerService(internalDB, processService, log)
	server := server.NewServer(
		cfg, log,
		frontendFiles,
		docsFiles,
		version,
		internalDB,
		fileEditor,
		processService,
		serverManagerService,
	)
	if err := server.ListenAndServe(); err != nil {
		log.Error("Could not start Omnihance A3 Agent server", logger.Field{Key: "error", Value: err})
		os.Exit(1)
	}

	defer func() {
		_ = server.Shutdown(context.Background())
	}()

	interruptChan := make(chan os.Signal, 1)
	signal.Notify(interruptChan, os.Interrupt, syscall.SIGTERM)
	<-interruptChan

	log.Info("Omnihance A3 Agent shutting down...")
}
