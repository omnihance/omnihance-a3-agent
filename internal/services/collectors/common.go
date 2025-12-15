package collectors

const UnitPercent = "percent"

type Collector interface {
	Collect() ([]MetricData, error)
}

type MetricData struct {
	Timestamp int64       `json:"timestamp"`
	Metric    MetricValue `json:"metric"`
}

type MetricValue struct {
	Name   string       `json:"name"`
	Labels []*LabelData `json:"labels"`
	Value  float64      `json:"value"`
}

type LabelData struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}
