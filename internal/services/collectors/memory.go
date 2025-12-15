package collectors

import (
	"runtime"
	"time"
)

const MemoryUsagePercentageMetricName = "memory_usage_percentage"

type memoryCollector struct{}

func NewMemoryCollector() Collector {
	return &memoryCollector{}
}

func (c *memoryCollector) Collect() ([]MetricData, error) {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	totalMemory := memStats.Sys
	usedMemory := memStats.Alloc

	var usagePercentage float64
	if totalMemory > 0 {
		usagePercentage = (float64(usedMemory) / float64(totalMemory)) * 100.0
	} else {
		usagePercentage = 0.0
	}

	if usagePercentage > 100.0 {
		usagePercentage = 100.0
	}
	if usagePercentage < 0.0 {
		usagePercentage = 0.0
	}

	timestamp := time.Now().Unix()

	metric := MetricData{
		Timestamp: timestamp,
		Metric: MetricValue{
			Name:   MemoryUsagePercentageMetricName,
			Labels: []*LabelData{},
			Value:  usagePercentage,
		},
	}

	return []MetricData{metric}, nil
}
