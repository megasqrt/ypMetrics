package agent

import "ypMetrics/models"

type Collector interface {
	Poll()
	GetMetrics() []models.Metrics
	PollGopsutil() 
}