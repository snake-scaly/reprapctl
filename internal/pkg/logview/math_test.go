package logview_test

import (
	"github.com/stretchr/testify/assert"
	"reprapctl/internal/pkg/logview"
	"testing"
)

func TestBinarySearch(t *testing.T) {
	tests := []struct {
		name       string
		size       int
		target     float64
		wantIndex  int
		wantMetric float64
	}{
		{
			name:       "Start",
			size:       10,
			target:     1.15,
			wantIndex:  0,
			wantMetric: 1.1,
		},
		{
			name:       "Middle",
			size:       10,
			target:     1.35,
			wantIndex:  2,
			wantMetric: 1.3,
		},
		{
			name:       "End",
			size:       10,
			target:     2.1,
			wantIndex:  10,
			wantMetric: 2.1,
		},
		{
			name:       "BeforeStart",
			size:       10,
			target:     -1,
			wantIndex:  0,
			wantMetric: 1.1,
		},
		{
			name:       "BeyondEnd",
			size:       10,
			target:     5,
			wantIndex:  10,
			wantMetric: 2.1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i, m := logview.BinarySearch(tt.size, tt.target, func(i int) float64 { return float64(i)/10 + 1.1 })
			assert.Equal(t, tt.wantIndex, i)
			assert.Equal(t, tt.wantMetric, m)
		})
	}
}
