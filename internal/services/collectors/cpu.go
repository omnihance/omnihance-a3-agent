package collectors

import (
	"runtime"
	"strconv"
	"time"
)

const CPUUsagePercentageMetricName = "cpu_usage_percentage"

type cpuCollector struct {
	prevTime         time.Time
	prevNumGoroutine int
	prevNumGC        uint32
}

func NewCpuCollector() Collector {
	return &cpuCollector{
		prevTime:         time.Now(),
		prevNumGoroutine: runtime.NumGoroutine(),
		prevNumGC:        0,
	}
}

func (c *cpuCollector) Collect() ([]MetricData, error) {
	numCPU := runtime.NumCPU()
	results := make([]MetricData, 0, numCPU)

	now := time.Now()
	currentNumGoroutine := runtime.NumGoroutine()

	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	currentNumGC := memStats.NumGC

	elapsed := now.Sub(c.prevTime)
	if elapsed <= 0 {
		elapsed = time.Second
	}

	goroutineDelta := float64(currentNumGoroutine - c.prevNumGoroutine)
	gcDelta := float64(currentNumGC - c.prevNumGC)

	elapsedSeconds := elapsed.Seconds()

	cpuUsage := (goroutineDelta + gcDelta*10.0) / elapsedSeconds
	if cpuUsage < 0 {
		cpuUsage = 0
	}
	if cpuUsage > 100.0 {
		cpuUsage = 100.0
	}

	c.prevTime = now
	c.prevNumGoroutine = currentNumGoroutine
	c.prevNumGC = currentNumGC

	timestamp := time.Now().Unix()

	for i := 1; i <= numCPU; i++ {
		coreLabel := &LabelData{
			Name:  "core",
			Value: strconv.Itoa(i),
		}

		metric := MetricData{
			Timestamp: timestamp,
			Metric: MetricValue{
				Name:   CPUUsagePercentageMetricName,
				Labels: []*LabelData{coreLabel},
				Value:  cpuUsage / float64(numCPU),
			},
		}

		results = append(results, metric)
	}

	return results, nil
}
