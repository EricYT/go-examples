package metrics

import (
	"net/http"
	"testing"
	"time"

	"github.com/armon/go-metrics"
	"github.com/armon/go-metrics/prometheus"
)

func init() {
	cfg := metrics.DefaultConfig("bench")
	sink, _ := prometheus.NewPrometheusSinkFrom(prometheus.PrometheusOpts{Expiration: 0})
	metrics.NewGlobal(cfg, sink)
	go func() {
		if err := http.ListenAndServe(":9526", nil); err != nil {
			panic(err)
		}
	}()
}

func BenchmarkGoMetricsIncr(b *testing.B) {
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		metrics.IncrCounterWithLabels(
			[]string{"bench", "test_counter"},
			1,
			[]metrics.Label{
				metrics.Label{Name: "lable1", Value: "metrics"},
				metrics.Label{Name: "lable2", Value: "metrics"},
			},
		)
	}
}

func BenchmarkGoMetricsIncrParallel(b *testing.B) {
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			metrics.IncrCounterWithLabels(
				[]string{"bench", "test_counter"},
				1,
				[]metrics.Label{
					metrics.Label{Name: "lable1", Value: "metrics"},
					metrics.Label{Name: "lable2", Value: "metrics"},
				},
			)

		}
	})
}

func BenchmarkGoMetricsGauge(b *testing.B) {
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		metrics.SetGaugeWithLabels(
			[]string{"bench", "test_counter"},
			1,
			[]metrics.Label{
				metrics.Label{Name: "lable1", Value: "metrics"},
				metrics.Label{Name: "lable2", Value: "metrics"},
			},
		)
	}
}

func BenchmarkGoMetricsGaugeParallel(b *testing.B) {
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			metrics.SetGaugeWithLabels(
				[]string{"bench", "test_counter"},
				1,
				[]metrics.Label{
					metrics.Label{Name: "lable1", Value: "metrics"},
					metrics.Label{Name: "lable2", Value: "metrics"},
				},
			)

		}
	})
}

func BenchmarkGoMetricsMeasureSince(b *testing.B) {
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		metrics.MeasureSinceWithLabels(
			[]string{"bench", "test_counter"},
			time.Now(),
			[]metrics.Label{
				metrics.Label{Name: "lable1", Value: "metrics"},
				metrics.Label{Name: "lable2", Value: "metrics"},
			},
		)
	}
}

func BenchmarkGoMetricsMeasureSinceParallel(b *testing.B) {
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			metrics.MeasureSinceWithLabels(
				[]string{"bench", "test_counter"},
				time.Now(),
				[]metrics.Label{
					metrics.Label{Name: "lable1", Value: "metrics"},
					metrics.Label{Name: "lable2", Value: "metrics"},
				},
			)

		}
	})
}
