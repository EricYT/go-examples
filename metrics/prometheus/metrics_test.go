package main

import (
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

func TestGauage(t *testing.T) {
	gauge.With(prometheus.Labels{"label1": "404", "label2": "GET"}).Set(float64(1234))
	time.Sleep(time.Minute * 30)
}

func BenchmarkPrometheusIncr(b *testing.B) {
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		counter.With(prometheus.Labels{"label1": "404", "label2": "GET"}).Inc()
	}
}

func BenchmarkPrometheusIncrParallel(b *testing.B) {
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			counter.With(prometheus.Labels{"label1": "404", "label2": "GET"}).Inc()
		}
	})
}

func BenchmarkPrometheusGauge(b *testing.B) {
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		gauge.With(prometheus.Labels{"label1": "404", "label2": "GET"}).Set(float64(i))
	}
}

func BenchmarkPrometheusGaugeParallel(b *testing.B) {
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			gauge.With(prometheus.Labels{"label1": "404", "label2": "GET"}).Set(1)
		}
	})
}

func BenchmarkPrometheusMeasure(b *testing.B) {
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		measure.With(prometheus.Labels{"label1": "404", "label2": "GET"}).Observe(float64(time.Now().UnixNano()))
	}
}

func BenchmarkPrometheusMeasureParallel(b *testing.B) {
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			measure.With(prometheus.Labels{"label1": "404", "label2": "GET"}).Observe(float64(time.Now().UnixNano()))
		}
	})
}
