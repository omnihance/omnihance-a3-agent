package db

import (
	"io"
	"path/filepath"
	"strings"
	"testing"

	"github.com/omnihance/omnihance-a3-agent/internal/logger"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
)

func TestGetMetricSamplesByTimeWindowAggregatesSamplesBeforeLabels(t *testing.T) {
	log := logger.NewZerologLogger(zerolog.New(io.Discard), "test", zerolog.Disabled)
	database := NewSQLiteDB(filepath.Join(t.TempDir(), "test.db"), log)
	require.NoError(t, database.Connect())
	t.Cleanup(func() {
		require.NoError(t, database.Close())
	})
	require.NoError(t, database.MigrateUp())

	unit := "%"
	description := "CPU usage"
	labels := map[string]string{
		"host": "game-1",
		"role": "agent",
	}
	firstTimestamp := int64(10)
	secondTimestamp := int64(20)

	require.NoError(t, database.InsertMetric("cpu_usage", MetricTypeGauge, labels, 10, &firstTimestamp, &unit, &description))
	require.NoError(t, database.InsertMetric("cpu_usage", MetricTypeGauge, labels, 30, &secondTimestamp, &unit, &description))

	samples, err := database.GetMetricSamplesByTimeWindow("cpu_usage", 0, 60, 60)
	require.NoError(t, err)
	require.Len(t, samples, 1)
	require.Equal(t, int64(0), samples[0].Timestamp)
	require.Equal(t, 20.0, samples[0].Value)
	require.Equal(t, 1, strings.Count(samples[0].Labels, `host="game-1"`))
	require.Equal(t, 1, strings.Count(samples[0].Labels, `role="agent"`))
}
