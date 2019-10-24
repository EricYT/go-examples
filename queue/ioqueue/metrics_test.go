package ioqueue

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMetrics_convertGranularity(t *testing.T) {
	tcs := []struct {
		size int
		typ  string
	}{
		{0, sizeSteps256KB},
		{200, sizeSteps256KB},
		{sizeBytes256KB, sizeSteps256KB},
		{sizeBytes256KB + 1, sizeSteps1M},
		{sizeBytes1M, sizeSteps1M},
		{sizeBytes1M + 1, sizeSteps2M},
		{sizeBytes2M, sizeSteps2M},
		{sizeBytes2M + 1, sizeSteps3M},
		{sizeBytes3M, sizeSteps3M},
		{sizeBytes3M + 1, sizeSteps4M},
		{sizeBytes4M, sizeSteps4M},
	}
	for _, tc := range tcs {
		assert.Equal(t, tc.typ, convertGranularity(tc.size))
	}
}
