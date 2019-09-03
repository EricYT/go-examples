package main

import (
	"math/rand"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func init() {
	prometheus.Register(counter)
	prometheus.Register(gauge)
	prometheus.Register(measure)
	prometheus.Register(withFunc)
}

func init() {
	http.Handle("/metrics", promhttp.Handler())

	go func() {
		if err := http.ListenAndServe(":9525", nil); err != nil {
			panic(err)
		}
	}()
}

// metrics
var (
	counter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "app1",
		Subsystem: "handler",
		Name:      "put_request_total",
		Help:      "The number of handler get requests.",
	}, []string{"label1", "label2"})

	gauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "app2",
		Subsystem: "handler_request",
		Name:      "put_request_duration",
		Help:      "The number of handler get requests gauge.",
	}, []string{"label1", "label2"})

	measure = prometheus.NewSummaryVec(prometheus.SummaryOpts{
		Namespace: "app2",
		Subsystem: "handler_rate",
		Name:      "handler_reqeust_rate",
		Help:      "The rate of handler",
		MaxAge:    10 * time.Second,
	}, []string{"label1", "label2"})

	withFunc = prometheus.NewGaugeFunc(prometheus.GaugeOpts{
		Namespace: "gauge1",
		Subsystem: "handle_gauge",
		Name:      "xxxdsdss",
		Help:      "no necessary",
	}, func() float64 { return float64(rand.Intn(123)) })
)
