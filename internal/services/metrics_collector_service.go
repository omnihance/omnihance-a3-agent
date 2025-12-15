package services

import (
	"context"
	"fmt"
	"sync"

	"github.com/omnihance/omnihance-a3-agent/internal/config"
	"github.com/omnihance/omnihance-a3-agent/internal/db"
	"github.com/omnihance/omnihance-a3-agent/internal/logger"
	"github.com/omnihance/omnihance-a3-agent/internal/services/collectors"
	"github.com/robfig/cron/v3"
)

type MetricsCollectorService interface {
	Start() error
	Stop() error
}

type metricsCollectorService struct {
	cfg        *config.EnvVars
	logger     logger.Logger
	collectors []collectors.Collector
	internalDB db.InternalDB
	cron       *cron.Cron
	ctx        context.Context
	cancel     context.CancelFunc
}

func NewMetricsCollectorService(
	cfg *config.EnvVars,
	logger logger.Logger,
	internalDB db.InternalDB,
) MetricsCollectorService {
	return &metricsCollectorService{
		cfg:        cfg,
		logger:     logger,
		internalDB: internalDB,
		collectors: []collectors.Collector{
			collectors.NewCpuCollector(),
			collectors.NewMemoryCollector(),
		},
	}
}

func (m *metricsCollectorService) Start() error {
	m.ctx, m.cancel = context.WithCancel(context.Background())

	m.cron = cron.New(cron.WithSeconds())

	collectionSchedule := fmt.Sprintf("@every %ds", m.cfg.MetricsCollectionIntervalSeconds)
	_, err := m.cron.AddFunc(collectionSchedule, m.collectMetrics)
	if err != nil {
		m.cancel()
		return fmt.Errorf("failed to schedule metrics collection: %w", err)
	}

	cleanupSchedule := fmt.Sprintf("@every %ds", m.cfg.MetricsCleanupIntervalSeconds)
	_, err = m.cron.AddFunc(cleanupSchedule, m.cleanupMetrics)
	if err != nil {
		m.cancel()
		return fmt.Errorf("failed to schedule metrics cleanup: %w", err)
	}

	m.cron.Start()

	m.logger.Info(
		"metrics collector service started",
		logger.Field{Key: "collection_interval_seconds", Value: m.cfg.MetricsCollectionIntervalSeconds},
		logger.Field{Key: "cleanup_interval_seconds", Value: m.cfg.MetricsCleanupIntervalSeconds},
		logger.Field{Key: "retention_days", Value: m.cfg.MetricsRetentionDays},
	)

	m.collectMetrics()

	return nil
}

func (m *metricsCollectorService) Stop() error {
	if m.cron != nil {
		ctx := m.cron.Stop()
		<-ctx.Done()
	}

	if m.cancel != nil {
		m.cancel()
	}

	m.logger.Info("metrics collector service stopped")

	return nil
}

func (m *metricsCollectorService) collectMetrics() {
	var wg sync.WaitGroup
	var mu sync.Mutex
	allMetrics := make([]collectors.MetricData, 0)

	for _, collector := range m.collectors {
		wg.Add(1)
		go func(c collectors.Collector) {
			defer wg.Done()

			metrics, err := c.Collect()
			if err != nil {
				m.logger.Error(
					"failed to collect metrics",
					logger.Field{Key: "error", Value: err},
				)
				return
			}

			mu.Lock()
			allMetrics = append(allMetrics, metrics...)
			mu.Unlock()
		}(collector)
	}

	wg.Wait()

	unit := collectors.UnitPercent
	metricType := db.MetricTypeGauge

	for _, metricData := range allMetrics {
		labels := make(map[string]string)
		for _, label := range metricData.Metric.Labels {
			labels[label.Name] = label.Value
		}

		description := fmt.Sprintf("%s metric", metricData.Metric.Name)

		err := m.internalDB.InsertMetric(
			metricData.Metric.Name,
			metricType,
			labels,
			metricData.Metric.Value,
			&metricData.Timestamp,
			&unit,
			&description,
		)

		if err != nil {
			m.logger.Error(
				"failed to insert metric",
				logger.Field{Key: "metric_name", Value: metricData.Metric.Name},
				logger.Field{Key: "error", Value: err},
			)
		}
	}

	m.logger.Debug(
		"collected metrics",
		logger.Field{Key: "count", Value: len(allMetrics)},
	)
}

func (m *metricsCollectorService) cleanupMetrics() {
	err := m.internalDB.DeleteOldMetrics(m.cfg.MetricsRetentionDays)
	if err != nil {
		m.logger.Error(
			"failed to cleanup old metrics",
			logger.Field{Key: "retention_days", Value: m.cfg.MetricsRetentionDays},
			logger.Field{Key: "error", Value: err},
		)
		return
	}

	m.logger.Debug(
		"cleaned up old metrics",
		logger.Field{Key: "retention_days", Value: m.cfg.MetricsRetentionDays},
	)
}
