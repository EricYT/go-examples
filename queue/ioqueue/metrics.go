package ioqueue

import (
	"fmt"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

func init() {
	prometheus.Register(handlerRequestsTotal)
	prometheus.Register(handlerRequestsBytes)
	prometheus.Register(handlerRequestsDuration)
	prometheus.Register(fairQueuePriorityClassRequestsTotal)
	prometheus.Register(fairQueuePriorityClassQueuedRequestsTotal)
}

var (
	handlerRequestsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "ioqueue",
		Subsystem: "ioqueue",
		Name:      "requests_total",
		Help:      "The counter of handle operations total.",
	}, []string{"method"})

	handlerRequestsBytes = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "ioqueue",
		Subsystem: "ioqueue",
		Name:      "requests_bytes",
		Help:      "The requests total bytes handled by ioqueue.",
	}, []string{"method"})

	handlerRequestsDuration = prometheus.NewSummaryVec(prometheus.SummaryOpts{
		Namespace: "ioqueue",
		Subsystem: "queue",
		Name:      "requests_duration_ms",
		Help:      "The summary of processing time (ms) of successfully handled requests, by io queue.",
	}, []string{"method", "granularity"})

	fairQueuePriorityClassRequestsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "ioqueue",
		Subsystem: "fair_queue",
		Name:      "priority_class_requests_total",
		Help:      "The counter of handle operations total of priority class.",
	}, []string{"class", "shares"})

	fairQueuePriorityClassQueuedRequestsTotal = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "ioqueue",
		Subsystem: "fair_queue",
		Name:      "priority_class_queued_requests_total",
		Help:      "The counter of queued requests total of priority class.",
	}, []string{"class", "shares"})
)

const (
	MethodRead  string = "READ"
	MethodWrite string = "WRITE"
)

func convertMethod(typ RequestType) string {
	switch typ {
	case RequestTypeWrite:
		return MethodWrite
	case RequestTypeRead:
		return MethodRead
	default:
	}
	return "UNKNOW"
}

var (
	sizeSteps256KB string = "256KB"
	sizeSteps1M    string = "1MB"
	sizeSteps2M    string = "2MB"
	sizeSteps3M    string = "3MB"
	sizeSteps4M    string = "4MB"
)

const (
	sizeBytes256KB int = 256 * 1024
	sizeBytes1M    int = 4 * sizeBytes256KB
	sizeBytes2M    int = 2 * sizeBytes1M
	sizeBytes3M    int = 3 * sizeBytes1M
	sizeBytes4M    int = 4 * sizeBytes1M
)

func convertGranularity(size int) string {
	if size <= sizeBytes256KB {
		return sizeSteps256KB
	} else if size <= sizeBytes1M {
		return sizeSteps1M
	} else if size <= sizeBytes2M {
		return sizeSteps2M
	} else if size <= sizeBytes3M {
		return sizeSteps3M
	} else {
		return sizeSteps4M
	}
}

func QueueRequestMetric(typ RequestType, size int) {
	method := convertMethod(typ)
	handlerRequestsTotal.WithLabelValues(method).Inc()
	handlerRequestsBytes.WithLabelValues(method).Add(float64(size))
}

func RequestDurationMetric(typ RequestType, startTime time.Time, size int) {
	method := convertMethod(typ)
	granularity := convertGranularity(size)
	elapse := float64(time.Since(startTime).Nanoseconds()) / float64(time.Millisecond)
	handlerRequestsDuration.WithLabelValues(method, granularity).Observe(elapse)
}

func FairQueuePriorityClassRequestsMetric(class string, shares uint32) {
	fairQueuePriorityClassRequestsTotal.WithLabelValues(class, fmt.Sprintf("%d", shares)).Inc()
}

func FairQueuePriorityClassQueuedRequestsMetric(class string, shares uint32, size int) {
	fairQueuePriorityClassQueuedRequestsTotal.WithLabelValues(class, fmt.Sprintf("%d", shares)).Set(float64(size))
}
