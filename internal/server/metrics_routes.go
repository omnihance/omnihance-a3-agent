package server

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/omnihance/omnihance-a3-agent/internal/constants"
	"github.com/omnihance/omnihance-a3-agent/internal/db"
	"github.com/omnihance/omnihance-a3-agent/internal/logger"
	"github.com/omnihance/omnihance-a3-agent/internal/mw"
	"github.com/omnihance/omnihance-a3-agent/internal/permissions"
	"github.com/omnihance/omnihance-a3-agent/internal/services/collectors"
	"github.com/omnihance/omnihance-a3-agent/internal/services/echarts"
	"github.com/omnihance/omnihance-a3-agent/internal/utils"
)

type MetricCard struct {
	Name         string  `json:"name"`
	MetricName   string  `json:"metric_name"`
	Description  string  `json:"description"`
	Value        float64 `json:"value"`
	DisplayValue string  `json:"display_value"`
}

func (s *Server) InitializeMetricsRoutes(r *chi.Mux) {
	r.Route("/api/metrics", func(r chi.Router) {
		r.Use(mw.CheckCookie(s.internalDB, s.cfg.CookieSecret))
		r.Get("/summary", s.getMetricsSummaryHandler)
		r.Get("/charts", s.getMetricsChartsHandler)
	})
}

func (s *Server) getMetricsSummaryHandler(w http.ResponseWriter, r *http.Request) {
	if !s.requireUserPermission(w, r, permissions.ActionViewMetrics) {
		return
	}

	if !s.cfg.MetricsEnabled {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusServiceUnavailable, map[string]interface{}{
			"errorCode": constants.ErrorCodeBadRequest,
			"context":   "metrics",
			"errors":    []string{"Metrics collection is disabled"},
		})
		return
	}

	samples, err := s.internalDB.GetLatestSamples()
	if err != nil {
		s.log.Error("Failed to get latest metric samples", logger.Field{Key: "error", Value: err})
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusInternalServerError, map[string]interface{}{
			"errorCode": constants.ErrorCodeInternalServerError,
			"context":   "db",
			"errors":    []string{"Failed to retrieve metrics"},
		})
		return
	}

	var cpuUsagePerCore float64
	var cpuSampleCount int
	var ramUsage float64
	var ramFound bool

	for _, sample := range samples {
		switch sample.MetricName {
		case collectors.CPUUsagePercentageMetricName:
			if cpuSampleCount == 0 {
				cpuUsagePerCore = sample.Value
			}
			cpuSampleCount++
		case collectors.MemoryUsagePercentageMetricName:
			ramUsage = sample.Value
			ramFound = true
		}
	}

	processCount, err := s.processService.GetProcessCount()
	if err != nil {
		s.log.Warn("Failed to get process count", logger.Field{Key: "error", Value: err})
	}

	cards := make([]MetricCard, 0, 3)

	if cpuSampleCount > 0 {
		cpuUsageTotal := cpuUsagePerCore * float64(cpuSampleCount)
		cards = append(cards, MetricCard{
			Name:         "CPU",
			MetricName:   collectors.CPUUsagePercentageMetricName,
			Description:  "Usage Percentage",
			Value:        cpuUsageTotal,
			DisplayValue: fmt.Sprintf("%.2f%%", cpuUsageTotal),
		})
	} else {
		cards = append(cards, MetricCard{
			Name:         "CPU",
			MetricName:   collectors.CPUUsagePercentageMetricName,
			Description:  "Usage Percentage",
			Value:        0,
			DisplayValue: fmt.Sprintf("%.2f%%", 0.0),
		})
	}

	if ramFound {
		cards = append(cards, MetricCard{
			Name:         "RAM",
			MetricName:   collectors.MemoryUsagePercentageMetricName,
			Description:  "Usage Percentage",
			Value:        ramUsage,
			DisplayValue: fmt.Sprintf("%.2f%%", ramUsage),
		})
	} else {
		cards = append(cards, MetricCard{
			Name:         "RAM",
			MetricName:   collectors.MemoryUsagePercentageMetricName,
			Description:  "Usage Percentage",
			Value:        0,
			DisplayValue: fmt.Sprintf("%.2f%%", 0.0),
		})
	}

	cards = append(cards, MetricCard{
		Name:         "Processes",
		MetricName:   "process_count",
		Description:  "Running Processes",
		Value:        float64(processCount),
		DisplayValue: fmt.Sprintf("%d", processCount),
	})

	response := map[string]interface{}{
		"cards": cards,
	}
	_ = utils.WriteJSONResponse(w, response)
}

type ChartConfig struct {
	Title      string                 `json:"title"`
	MetricName string                 `json:"metric_name"`
	Options    map[string]interface{} `json:"options"`
	Filters    []TimeRangeFilter      `json:"filters"`
}

type TimeRangeFilter struct {
	Key             string   `json:"key"`
	AvailableValues []string `json:"available_values"`
	DefaultValue    string   `json:"default_value"`
}

type ChartResponse struct {
	Charts []ChartConfig `json:"charts"`
}

const (
	defaultMetricsRange      = "1h"
	metricsChartTargetPoints = int64(60)
)

var metricsRangeOptions = []string{"1h", "6h", "1d", "7d"}

func (s *Server) getMetricsChartsHandler(w http.ResponseWriter, r *http.Request) {
	if !s.requireUserPermission(w, r, permissions.ActionViewMetrics) {
		return
	}

	if !s.cfg.MetricsEnabled {
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusServiceUnavailable, map[string]interface{}{
			"errorCode": constants.ErrorCodeBadRequest,
			"context":   "metrics",
			"errors":    []string{"Metrics collection is disabled"},
		})
		return
	}

	timeWindow, err := parseMetricsTimeWindow(r, metricsRangeOptions, defaultMetricsRange)
	if err != nil {
		s.log.Error("Failed to parse metrics time window", logger.Field{Key: "error", Value: err})
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusBadRequest, map[string]interface{}{
			"errorCode": constants.ErrorCodeBadRequest,
			"context":   "time_window",
			"errors":    []string{err.Error()},
		})
		return
	}

	stepSeconds := calculateMetricsStepSeconds(
		timeWindow.DurationSeconds,
		int64(s.cfg.MetricsCollectionIntervalSeconds),
		metricsChartTargetPoints,
	)

	charts := make([]ChartConfig, 0, 2)

	cpuSamples, err := s.internalDB.GetMetricSamplesByTimeWindow(
		collectors.CPUUsagePercentageMetricName,
		timeWindow.StartTime,
		timeWindow.EndTime,
		stepSeconds,
	)
	if err != nil {
		s.log.Error("Failed to get CPU metric samples", logger.Field{Key: "error", Value: err})
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusInternalServerError, map[string]interface{}{
			"errorCode": constants.ErrorCodeInternalServerError,
			"context":   "db",
			"errors":    []string{"Failed to retrieve CPU metrics"},
		})
		return
	}

	cpuOptions, err := s.generateLineChartOptions(cpuSamples, true)
	if err != nil {
		s.log.Error("Failed to generate CPU chart options", logger.Field{Key: "error", Value: err})
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusInternalServerError, map[string]interface{}{
			"errorCode": constants.ErrorCodeInternalServerError,
			"context":   "chart_generation",
			"errors":    []string{"Failed to generate CPU chart"},
		})
		return
	}

	charts = append(charts, ChartConfig{
		Title:      "CPU Usage",
		MetricName: collectors.CPUUsagePercentageMetricName,
		Options:    cpuOptions,
		Filters: []TimeRangeFilter{
			{
				Key:             fmt.Sprintf("%s_range", collectors.CPUUsagePercentageMetricName),
				AvailableValues: metricsRangeOptions,
				DefaultValue:    defaultMetricsRange,
			},
		},
	})

	ramSamples, err := s.internalDB.GetMetricSamplesByTimeWindow(
		collectors.MemoryUsagePercentageMetricName,
		timeWindow.StartTime,
		timeWindow.EndTime,
		stepSeconds,
	)
	if err != nil {
		s.log.Error("Failed to get RAM metric samples", logger.Field{Key: "error", Value: err})
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusInternalServerError, map[string]interface{}{
			"errorCode": constants.ErrorCodeInternalServerError,
			"context":   "db",
			"errors":    []string{"Failed to retrieve RAM metrics"},
		})
		return
	}

	ramOptions, err := s.generateLineChartOptions(ramSamples, false)
	if err != nil {
		s.log.Error("Failed to generate RAM chart options", logger.Field{Key: "error", Value: err})
		_ = utils.WriteJSONResponseWithStatus(w, http.StatusInternalServerError, map[string]interface{}{
			"errorCode": constants.ErrorCodeInternalServerError,
			"context":   "chart_generation",
			"errors":    []string{"Failed to generate RAM chart"},
		})
		return
	}

	charts = append(charts, ChartConfig{
		Title:      "RAM Usage",
		MetricName: collectors.MemoryUsagePercentageMetricName,
		Options:    ramOptions,
		Filters: []TimeRangeFilter{
			{
				Key:             fmt.Sprintf("%s_range", collectors.MemoryUsagePercentageMetricName),
				AvailableValues: metricsRangeOptions,
				DefaultValue:    defaultMetricsRange,
			},
		},
	})

	response := ChartResponse{
		Charts: charts,
	}

	_ = utils.WriteJSONResponse(w, response)
}

func (s *Server) generateLineChartOptions(samples []db.MetricSampleWithLabels, aggregateCPU bool) (map[string]interface{}, error) {
	var seriesData []interface{}
	seriesName := "Usage"
	var lineColor string

	if len(samples) == 0 {
		seriesData = []interface{}{}
	} else {
		if aggregateCPU {
			aggregatedData := s.aggregateCPUSamples(samples)
			seriesData = make([]interface{}, 0, len(aggregatedData))
			seriesName = "CPU Usage"
			lineColor = "#5470c6"

			for _, point := range aggregatedData {
				seriesData = append(seriesData, []interface{}{point.Timestamp * 1000, point.Value})
			}
		} else {
			seriesData = make([]interface{}, 0, len(samples))
			seriesName = "RAM Usage"
			lineColor = "#91cc75"

			for _, sample := range samples {
				seriesData = append(seriesData, []interface{}{sample.Timestamp * 1000, sample.Value})
			}
		}
	}

	service := echarts.NewService()

	service.SetTooltip(
		echarts.NewTooltip().
			WithTrigger("axis").
			WithAxisPointer(
				echarts.NewAxisPointer().
					WithType("cross"),
			),
	)

	service.SetLegend(
		echarts.NewLegend().
			WithShow(true).
			WithBottom("2%"),
	)

	service.SetGrid(
		echarts.NewGrid().
			WithLeft("1%").
			WithRight("1%").
			WithTop("20%").
			WithBottom("12%").
			WithContainLabel(true),
	)

	service.AddXAxis(
		echarts.NewAxis().
			WithType("time"),
	)

	service.AddYAxis(
		echarts.NewAxis().
			WithType("value").
			WithName("Usage (%)").
			WithMin(0).
			WithMax(100).
			WithAxisLabel(
				echarts.NewAxisLabel().
					WithFormatter("{value}%"),
			),
	)

	series := echarts.NewSeries().
		WithType("line").
		WithName(seriesName).
		WithData(seriesData).
		WithSmooth(true).
		WithLineStyle(
			echarts.NewLineStyle().
				WithColor(lineColor).
				WithWidth(2),
		)

	service.AddSeries(series)

	return service.ToMap()
}

type AggregatedPoint struct {
	Timestamp int64
	Value     float64
}

func (s *Server) aggregateCPUSamples(samples []db.MetricSampleWithLabels) []AggregatedPoint {
	timestampMap := make(map[int64]float64)
	timestampOrder := make([]int64, 0)

	for _, sample := range samples {
		if _, exists := timestampMap[sample.Timestamp]; !exists {
			timestampOrder = append(timestampOrder, sample.Timestamp)
		}
		timestampMap[sample.Timestamp] += sample.Value
	}

	result := make([]AggregatedPoint, 0, len(timestampOrder))
	for _, ts := range timestampOrder {
		result = append(result, AggregatedPoint{
			Timestamp: ts,
			Value:     timestampMap[ts],
		})
	}

	return result
}

type metricsTimeWindow struct {
	StartTime       int64
	EndTime         int64
	DurationSeconds int64
	Range           string
}

func parseMetricsTimeWindow(r *http.Request, allowedRanges []string, defaultRange string) (metricsTimeWindow, error) {
	rangeValue := r.URL.Query().Get("range")
	fromValue := r.URL.Query().Get("from")
	toValue := r.URL.Query().Get("to")

	if fromValue != "" || toValue != "" {
		if fromValue == "" || toValue == "" {
			return metricsTimeWindow{}, fmt.Errorf("both from and to must be provided for custom time window")
		}

		fromTimestamp, err := strconv.ParseInt(fromValue, 10, 64)
		if err != nil {
			return metricsTimeWindow{}, fmt.Errorf("invalid from timestamp: %s", fromValue)
		}

		toTimestamp, err := strconv.ParseInt(toValue, 10, 64)
		if err != nil {
			return metricsTimeWindow{}, fmt.Errorf("invalid to timestamp: %s", toValue)
		}

		if toTimestamp <= fromTimestamp {
			return metricsTimeWindow{}, fmt.Errorf("to must be greater than from")
		}

		return metricsTimeWindow{
			StartTime:       fromTimestamp,
			EndTime:         toTimestamp,
			DurationSeconds: toTimestamp - fromTimestamp,
			Range:           "",
		}, nil
	}

	if rangeValue == "" {
		rangeValue = defaultRange
	}

	if !isSupportedRange(rangeValue, allowedRanges) {
		return metricsTimeWindow{}, fmt.Errorf("invalid time range: %s", rangeValue)
	}

	durationSeconds, err := utils.ParseTimeRangeToSeconds(rangeValue)
	if err != nil {
		return metricsTimeWindow{}, fmt.Errorf("invalid time range: %s", rangeValue)
	}

	now := time.Now().Unix()

	return metricsTimeWindow{
		StartTime:       now - durationSeconds,
		EndTime:         now,
		DurationSeconds: durationSeconds,
		Range:           rangeValue,
	}, nil
}

func isSupportedRange(rangeValue string, allowedRanges []string) bool {
	for _, allowedRange := range allowedRanges {
		if rangeValue == allowedRange {
			return true
		}
	}

	return false
}

func calculateMetricsStepSeconds(
	durationSeconds int64,
	collectionIntervalSeconds int64,
	targetPoints int64,
) int64 {
	if durationSeconds <= 0 {
		return 1
	}

	if collectionIntervalSeconds <= 0 {
		collectionIntervalSeconds = 1
	}

	if targetPoints <= 0 {
		targetPoints = 1
	}

	windowBasedStep := (durationSeconds + targetPoints - 1) / targetPoints
	if windowBasedStep < 1 {
		windowBasedStep = 1
	}

	if windowBasedStep < collectionIntervalSeconds {
		return collectionIntervalSeconds
	}

	return windowBasedStep
}
