package db

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/doug-martin/goqu/v9"
	"github.com/omnihance/omnihance-a3-agent/internal/logger"
)

type MetricType string

const (
	MetricTypeCounter   MetricType = "counter"
	MetricTypeGauge     MetricType = "gauge"
	MetricTypeHistogram MetricType = "histogram"
	MetricTypeSummary   MetricType = "summary"
)

type MetricName struct {
	ID          int64     `db:"id" json:"id"`
	Name        string    `db:"name" json:"name"`
	Type        string    `db:"type" json:"type"`
	Unit        *string   `db:"unit" json:"unit"`
	Description *string   `db:"description" json:"description"`
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
}

type Label struct {
	ID    int64  `db:"id" json:"id"`
	Key   string `db:"key" json:"key"`
	Value string `db:"value" json:"value"`
}

type MetricSeries struct {
	ID          int64     `db:"id" json:"id"`
	MetricID    int64     `db:"metric_id" json:"metric_id"`
	LabelHash   string    `db:"label_hash" json:"label_hash"`
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
	LastUpdated time.Time `db:"last_updated" json:"last_updated"`
}

type SeriesLabel struct {
	SeriesID int64 `db:"series_id" json:"series_id"`
	LabelID  int64 `db:"label_id" json:"label_id"`
}

type MetricSample struct {
	SeriesID  int64   `db:"series_id" json:"series_id"`
	Timestamp int64   `db:"timestamp" json:"timestamp"`
	Value     float64 `db:"value" json:"value"`
}

type SeriesWithLabels struct {
	SeriesID    int64     `db:"series_id" json:"series_id"`
	MetricName  string    `db:"metric_name" json:"metric_name"`
	MetricType  string    `db:"metric_type" json:"metric_type"`
	MetricUnit  *string   `db:"metric_unit" json:"metric_unit"`
	Labels      string    `db:"labels" json:"labels"`
	LabelHash   string    `db:"label_hash" json:"label_hash"`
	LastUpdated time.Time `db:"last_updated" json:"last_updated"`
}

type LatestSample struct {
	SeriesID   int64   `db:"series_id" json:"series_id"`
	MetricName string  `db:"metric_name" json:"metric_name"`
	Labels     string  `db:"labels" json:"labels"`
	Timestamp  int64   `db:"timestamp" json:"timestamp"`
	Value      float64 `db:"value" json:"value"`
	MetricUnit *string `db:"metric_unit" json:"metric_unit"`
}

type MetricSampleWithLabels struct {
	SeriesID   int64   `db:"series_id" json:"series_id"`
	MetricName string  `db:"metric_name" json:"metric_name"`
	Labels     string  `db:"labels" json:"labels"`
	Timestamp  int64   `db:"timestamp" json:"timestamp"`
	Value      float64 `db:"value" json:"value"`
	MetricUnit *string `db:"metric_unit" json:"metric_unit"`
}

func (s *sqliteInternalDB) InsertMetric(
	metricName string,
	metricType MetricType,
	labels map[string]string,
	value float64,
	timestamp *int64,
	unit *string,
	description *string,
) error {
	metricID, err := s.getOrCreateMetricName(metricName, metricType, unit, description)
	if err != nil {
		return fmt.Errorf("failed to get or create metric name: %w", err)
	}

	labelIDs := make([]int64, 0, len(labels))
	for key, val := range labels {
		labelID, err := s.getOrCreateLabel(key, val)
		if err != nil {
			return fmt.Errorf("failed to get or create label %s=%s: %w", key, val, err)
		}

		labelIDs = append(labelIDs, labelID)
	}

	labelHash := generateLabelHash(labels)
	seriesID, err := s.getOrCreateSeries(metricID, labelHash, labelIDs)
	if err != nil {
		return fmt.Errorf("failed to get or create series: %w", err)
	}

	if err := s.InsertMetricSample(seriesID, value, timestamp); err != nil {
		return fmt.Errorf("failed to insert metric sample: %w", err)
	}

	return nil
}

func (s *sqliteInternalDB) InsertMetricSample(seriesID int64, value float64, timestamp *int64) error {
	var ts int64
	if timestamp != nil {
		ts = *timestamp
	} else {
		ts = time.Now().Unix()
	}

	_, err := s.goqu.Insert("metric_samples").
		Prepared(true).
		Rows(goqu.Record{
			"series_id": seriesID,
			"timestamp": ts,
			"value":     value,
		}).
		Executor().
		Exec()
	if err != nil {
		s.logger.Error(
			"failed to insert metric sample",
			logger.Field{Key: "series_id", Value: seriesID},
			logger.Field{Key: "value", Value: value},
			logger.Field{Key: "timestamp", Value: ts},
			logger.Field{Key: "error", Value: err},
		)
		return fmt.Errorf("failed to insert metric sample: %w", err)
	}

	return nil
}

func (s *sqliteInternalDB) GetSeriesWithLabels() ([]SeriesWithLabels, error) {
	var results []SeriesWithLabels

	err := s.goqu.Select(
		goqu.I("ms.id").As("series_id"),
		goqu.I("mn.name").As("metric_name"),
		goqu.I("mn.type").As("metric_type"),
		goqu.I("mn.unit").As("metric_unit"),
		goqu.L("COALESCE(GROUP_CONCAT(l.key || '=\"' || l.value || '\"', ', '), '')").As("labels"),
		goqu.I("ms.label_hash"),
		goqu.I("ms.last_updated"),
	).
		Prepared(true).
		From(goqu.T("metric_series").As("ms")).
		InnerJoin(goqu.T("metric_names").As("mn"), goqu.On(goqu.I("ms.metric_id").Eq(goqu.I("mn.id")))).
		LeftJoin(goqu.T("series_labels").As("sl"), goqu.On(goqu.I("ms.id").Eq(goqu.I("sl.series_id")))).
		LeftJoin(goqu.T("labels").As("l"), goqu.On(goqu.I("sl.label_id").Eq(goqu.I("l.id")))).
		GroupBy(
			goqu.I("ms.id"),
			goqu.I("mn.name"),
			goqu.I("mn.type"),
			goqu.I("mn.unit"),
			goqu.I("ms.label_hash"),
			goqu.I("ms.last_updated"),
		).
		Order(goqu.I("ms.id").Asc()).
		ScanStructs(&results)

	if err != nil {
		s.logger.Error(
			"failed to query series with labels",
			logger.Field{Key: "error", Value: err},
		)
		return nil, fmt.Errorf("failed to query series with labels: %w", err)
	}

	for i := range results {
		if results[i].MetricUnit != nil && *results[i].MetricUnit == "" {
			results[i].MetricUnit = nil
		}
	}

	return results, nil
}

func (s *sqliteInternalDB) GetLatestSamples() ([]LatestSample, error) {
	latestSubquery := s.goqu.Select(
		goqu.I("series_id"),
		goqu.MAX("timestamp").As("max_ts"),
	).
		From("metric_samples").
		GroupBy(goqu.I("series_id"))

	var results []LatestSample

	err := s.goqu.Select(
		goqu.I("ms.series_id"),
		goqu.I("mn.name").As("metric_name"),
		goqu.L("COALESCE(GROUP_CONCAT(l.key || '=\"' || l.value || '\"', ', '), '')").As("labels"),
		goqu.I("ms.timestamp"),
		goqu.I("ms.value"),
		goqu.I("mn.unit").As("metric_unit"),
	).
		Prepared(true).
		With("latest", latestSubquery).
		From(goqu.T("metric_samples").As("ms")).
		InnerJoin(goqu.T("latest"), goqu.On(
			goqu.I("ms.series_id").Eq(goqu.I("latest.series_id")),
			goqu.I("ms.timestamp").Eq(goqu.I("latest.max_ts")),
		)).
		InnerJoin(goqu.T("metric_series").As("ser"), goqu.On(goqu.I("ms.series_id").Eq(goqu.I("ser.id")))).
		InnerJoin(goqu.T("metric_names").As("mn"), goqu.On(goqu.I("ser.metric_id").Eq(goqu.I("mn.id")))).
		LeftJoin(goqu.T("series_labels").As("sl"), goqu.On(goqu.I("ser.id").Eq(goqu.I("sl.series_id")))).
		LeftJoin(goqu.T("labels").As("l"), goqu.On(goqu.I("sl.label_id").Eq(goqu.I("l.id")))).
		GroupBy(
			goqu.I("ms.series_id"),
			goqu.I("mn.name"),
			goqu.I("ms.timestamp"),
			goqu.I("ms.value"),
			goqu.I("mn.unit"),
		).
		Order(goqu.I("ms.series_id").Asc()).
		ScanStructs(&results)

	if err != nil {
		s.logger.Error(
			"failed to query latest samples",
			logger.Field{Key: "error", Value: err},
		)
		return nil, fmt.Errorf("failed to query latest samples: %w", err)
	}

	for i := range results {
		if results[i].MetricUnit != nil && *results[i].MetricUnit == "" {
			results[i].MetricUnit = nil
		}
	}

	return results, nil
}

func (s *sqliteInternalDB) getOrCreateMetricName(name string, metricType MetricType, unit *string, description *string) (int64, error) {
	var metric MetricName
	found, err := s.goqu.From("metric_names").
		Prepared(true).
		Where(goqu.Ex{"name": name}).
		ScanStruct(&metric)
	if err != nil {
		s.logger.Error(
			"failed to get metric name",
			logger.Field{Key: "name", Value: name},
			logger.Field{Key: "error", Value: err},
		)
		return 0, fmt.Errorf("failed to get metric name %s: %w", name, err)
	}

	if found {
		return metric.ID, nil
	}

	record := goqu.Record{
		"name":       name,
		"type":       string(metricType),
		"created_at": goqu.L("CURRENT_TIMESTAMP"),
	}

	if unit != nil {
		record["unit"] = *unit
	}
	if description != nil {
		record["description"] = *description
	}

	result, err := s.goqu.Insert("metric_names").
		Prepared(true).
		Rows(record).
		Executor().
		Exec()
	if err != nil {
		s.logger.Error(
			"failed to create metric name",
			logger.Field{Key: "name", Value: name},
			logger.Field{Key: "error", Value: err},
		)
		return 0, fmt.Errorf("failed to create metric name %s: %w", name, err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("failed to get metric name ID: %w", err)
	}

	return id, nil
}

func (s *sqliteInternalDB) getOrCreateLabel(key string, value string) (int64, error) {
	var label Label
	found, err := s.goqu.From("labels").
		Prepared(true).
		Where(goqu.Ex{"key": key, "value": value}).
		ScanStruct(&label)
	if err != nil {
		s.logger.Error(
			"failed to get label",
			logger.Field{Key: "key", Value: key},
			logger.Field{Key: "value", Value: value},
			logger.Field{Key: "error", Value: err},
		)
		return 0, fmt.Errorf("failed to get label %s=%s: %w", key, value, err)
	}

	if found {
		return label.ID, nil
	}

	result, err := s.goqu.Insert("labels").
		Prepared(true).
		Rows(goqu.Record{
			"key":   key,
			"value": value,
		}).
		Executor().
		Exec()
	if err != nil {
		s.logger.Error(
			"failed to create label",
			logger.Field{Key: "key", Value: key},
			logger.Field{Key: "value", Value: value},
			logger.Field{Key: "error", Value: err},
		)
		return 0, fmt.Errorf("failed to create label %s=%s: %w", key, value, err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("failed to get label ID: %w", err)
	}

	return id, nil
}

func (s *sqliteInternalDB) getOrCreateSeries(metricID int64, labelHash string, labelIDs []int64) (int64, error) {
	var series MetricSeries
	found, err := s.goqu.From("metric_series").
		Prepared(true).
		Where(goqu.Ex{"metric_id": metricID, "label_hash": labelHash}).
		ScanStruct(&series)
	if err != nil {
		s.logger.Error(
			"failed to get metric series",
			logger.Field{Key: "metric_id", Value: metricID},
			logger.Field{Key: "label_hash", Value: labelHash},
			logger.Field{Key: "error", Value: err},
		)
		return 0, fmt.Errorf("failed to get metric series: %w", err)
	}

	if found {
		return series.ID, nil
	}

	result, err := s.goqu.Insert("metric_series").
		Prepared(true).
		Rows(goqu.Record{
			"metric_id":    metricID,
			"label_hash":   labelHash,
			"created_at":   goqu.L("CURRENT_TIMESTAMP"),
			"last_updated": goqu.L("CURRENT_TIMESTAMP"),
		}).
		Executor().
		Exec()
	if err != nil {
		s.logger.Error(
			"failed to create metric series",
			logger.Field{Key: "metric_id", Value: metricID},
			logger.Field{Key: "label_hash", Value: labelHash},
			logger.Field{Key: "error", Value: err},
		)
		return 0, fmt.Errorf("failed to create metric series: %w", err)
	}

	seriesID, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("failed to get series ID: %w", err)
	}

	for _, labelID := range labelIDs {
		_, err := s.goqu.Insert("series_labels").
			Prepared(true).
			Rows(goqu.Record{
				"series_id": seriesID,
				"label_id":  labelID,
			}).
			OnConflict(goqu.DoNothing()).
			Executor().
			Exec()
		if err != nil {
			s.logger.Error(
				"failed to link label to series",
				logger.Field{Key: "series_id", Value: seriesID},
				logger.Field{Key: "label_id", Value: labelID},
				logger.Field{Key: "error", Value: err},
			)
			return 0, fmt.Errorf("failed to link label to series: %w", err)
		}
	}

	return seriesID, nil
}

func generateLabelHash(labels map[string]string) string {
	if len(labels) == 0 {
		return ""
	}

	keys := make([]string, 0, len(labels))
	for k := range labels {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	pairs := make([]string, 0, len(keys))
	for _, k := range keys {
		pairs = append(pairs, fmt.Sprintf("%s=%s", k, labels[k]))
	}

	return strings.Join(pairs, ",")
}

func (s *sqliteInternalDB) DeleteOldMetrics(retentionDays int) error {
	cutoffTimestamp := time.Now().Unix() - int64(retentionDays*24*60*60)

	result, err := s.goqu.Delete("metric_samples").
		Prepared(true).
		Where(goqu.C("timestamp").Lt(cutoffTimestamp)).
		Executor().
		Exec()
	if err != nil {
		s.logger.Error(
			"failed to delete old metrics",
			logger.Field{Key: "retention_days", Value: retentionDays},
			logger.Field{Key: "cutoff_timestamp", Value: cutoffTimestamp},
			logger.Field{Key: "error", Value: err},
		)
		return fmt.Errorf("failed to delete old metrics: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		s.logger.Error(
			"failed to get rows affected",
			logger.Field{Key: "error", Value: err},
		)
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected > 0 {
		s.logger.Info(
			"deleted old metrics",
			logger.Field{Key: "retention_days", Value: retentionDays},
			logger.Field{Key: "cutoff_timestamp", Value: cutoffTimestamp},
			logger.Field{Key: "rows_deleted", Value: rowsAffected},
		)
	}

	return nil
}

func (s *sqliteInternalDB) GetMetricSamplesByTimeRange(metricName string, startTime, endTime int64) ([]MetricSampleWithLabels, error) {
	var results []MetricSampleWithLabels

	err := s.goqu.Select(
		goqu.I("ms.series_id"),
		goqu.I("mn.name").As("metric_name"),
		goqu.L("COALESCE(GROUP_CONCAT(l.key || '=\"' || l.value || '\"', ', '), '')").As("labels"),
		goqu.I("ms.timestamp"),
		goqu.I("ms.value"),
		goqu.I("mn.unit").As("metric_unit"),
	).
		Prepared(true).
		From(goqu.T("metric_samples").As("ms")).
		InnerJoin(goqu.T("metric_series").As("ser"), goqu.On(goqu.I("ms.series_id").Eq(goqu.I("ser.id")))).
		InnerJoin(goqu.T("metric_names").As("mn"), goqu.On(goqu.I("ser.metric_id").Eq(goqu.I("mn.id")))).
		LeftJoin(goqu.T("series_labels").As("sl"), goqu.On(goqu.I("ser.id").Eq(goqu.I("sl.series_id")))).
		LeftJoin(goqu.T("labels").As("l"), goqu.On(goqu.I("sl.label_id").Eq(goqu.I("l.id")))).
		Where(
			goqu.I("mn.name").Eq(metricName),
			goqu.I("ms.timestamp").Gte(startTime),
			goqu.I("ms.timestamp").Lte(endTime),
		).
		GroupBy(
			goqu.I("ms.series_id"),
			goqu.I("mn.name"),
			goqu.I("ms.timestamp"),
			goqu.I("ms.value"),
			goqu.I("mn.unit"),
		).
		Order(goqu.I("ms.timestamp").Asc()).
		ScanStructs(&results)

	if err != nil {
		s.logger.Error(
			"failed to query metric samples by time range",
			logger.Field{Key: "metric_name", Value: metricName},
			logger.Field{Key: "start_time", Value: startTime},
			logger.Field{Key: "end_time", Value: endTime},
			logger.Field{Key: "error", Value: err},
		)
		return nil, fmt.Errorf("failed to query metric samples by time range: %w", err)
	}

	for i := range results {
		if results[i].MetricUnit != nil && *results[i].MetricUnit == "" {
			results[i].MetricUnit = nil
		}
	}

	return results, nil
}
