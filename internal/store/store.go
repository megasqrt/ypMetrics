package store

type Storage interface {
    // GetMetric(name string) (float64, error)
    // SetMetric(name string, value float64) error
	UpdateGauge(name string, value float64) 
	UpdateCounter(name string, value int64) int64 
	GetAllMetrics() map[string] interface{}
	GetMetricsByTypeAndName(mName, mType string) ([]byte, error) 
}