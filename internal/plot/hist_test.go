package plot

import (
	"fmt"
	"math"
	"reflect"
	"runtime/metrics"
	"slices"
	"testing"
)

func TestCumulativeHistogramDelta(t *testing.T) {
	hist := metrics.Float64Histogram{
		Buckets: []float64{0, 1, 2, 3},
		Counts:  []uint64{10, 4, 2},
	}
	var delta cumulativeHistogramDelta

	next := func(counts ...uint64) []uint64 {
		t.Helper()
		copy(hist.Counts, counts)
		return slices.Clone(delta.next(&hist))
	}

	tests := []struct {
		name   string
		counts []uint64
		want   []uint64
	}{
		{name: "initial", counts: []uint64{10, 4, 2}, want: []uint64{0, 0, 0}},
		{name: "unchanged", counts: []uint64{10, 4, 2}, want: []uint64{0, 0, 0}},
		{name: "increments", counts: []uint64{11, 4, 5}, want: []uint64{1, 0, 3}},
		{name: "counter reset", counts: []uint64{2, 1, 0}, want: []uint64{2, 1, 0}},
		{name: "after reset", counts: []uint64{3, 1, 1}, want: []uint64{1, 0, 1}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := next(tt.counts...); !slices.Equal(got, tt.want) {
				t.Fatalf("delta = %v; want %v", got, tt.want)
			}
		})
	}
}

func TestCumulativeHistogramDeltaDownsamplesBeforeDifferencing(t *testing.T) {
	hist := metrics.Float64Histogram{
		Buckets: make([]float64, 202),
		Counts:  make([]uint64, 201),
	}
	for i := range hist.Counts {
		hist.Counts[i] = uint64(i)
	}
	var delta cumulativeHistogramDelta
	first := slices.Clone(delta.next(&hist))
	if len(first) > maxBuckets {
		t.Fatalf("initial delta has %d buckets; want at most %d", len(first), maxBuckets)
	}
	if !slices.Equal(first, make([]uint64, len(first))) {
		t.Fatalf("initial delta = %v; want zeroes", first)
	}

	for i := range hist.Counts {
		hist.Counts[i]++
	}
	got := slices.Clone(delta.next(&hist))
	for i, count := range got {
		want := uint64(3)
		if count != want {
			t.Fatalf("delta bucket %d = %d; want %d", i, count, want)
		}
	}
}

func Test_downsampleFactor(t *testing.T) {
	tests := []struct {
		nbuckets   int
		maxbuckets int
		want       int
	}{
		{nbuckets: 99, maxbuckets: 100, want: 1},
		{nbuckets: 100, maxbuckets: 100, want: 1},
		{nbuckets: 101, maxbuckets: 100, want: 2},
		{nbuckets: 10, maxbuckets: 5, want: 2},
		{nbuckets: 11, maxbuckets: 5, want: 3},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("n=%d,max=%d", tt.nbuckets, tt.maxbuckets), func(t *testing.T) {
			hist := metrics.Float64Histogram{
				Counts:  make([]uint64, tt.nbuckets),
				Buckets: make([]float64, tt.nbuckets+1),
			}
			if got := downsampleFactor(&hist, tt.maxbuckets); got != tt.want {
				t.Errorf("downsampleFactor() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_downsample(t *testing.T) {
	tests := []struct {
		name        string
		hist        metrics.Float64Histogram
		factor      int
		wantBuckets []float64
		wantCounts  []uint64
	}{
		{
			name: "factor 1",
			hist: metrics.Float64Histogram{
				Buckets: []float64{0, 1, 2, 3, 4, 5, 6},
				Counts:  []uint64{2, 2, 1, 4, 5, 6},
			},
			factor:      1,
			wantBuckets: []float64{1, 2, 3, 4, 5, 6},
			wantCounts:  []uint64{2, 2, 1, 4, 5, 6},
		},
		{
			name: "factor 1 with infinites",
			hist: metrics.Float64Histogram{
				Buckets: []float64{math.Inf(-1), 1, 2, 3, 4, 5, math.Inf(1)},
				Counts:  []uint64{2, 2, 1, 4, 5, 6},
			},
			factor:      1,
			wantBuckets: []float64{1, 2, 3, 4, 5, 6},
			wantCounts:  []uint64{2, 2, 1, 4, 5, 6},
		},
		{
			name: "divisible by factor 3",
			hist: metrics.Float64Histogram{
				Buckets: []float64{0, 1, 2, 3, 4, 5, 6},
				Counts:  []uint64{2, 2, 1, 4, 5, 6},
			},
			factor:      3,
			wantBuckets: []float64{3, 6},
			wantCounts:  []uint64{5, 15},
		},
		{
			name: "divisible by factor 2",
			hist: metrics.Float64Histogram{
				Buckets: []float64{0, 1, 2, 3, 4, 5, 6},
				Counts:  []uint64{2, 2, 1, 4, 5, 6},
			},
			factor:      2,
			wantBuckets: []float64{2, 4, 6},
			wantCounts:  []uint64{4, 5, 11},
		},
		{
			name: "not divisible by factor 2",
			hist: metrics.Float64Histogram{
				Buckets: []float64{0, 1, 2, 3, 4, 5},
				Counts:  []uint64{2, 2, 1, 4, 5},
			},
			factor:      2,
			wantBuckets: []float64{2, 4, 5},
			wantCounts:  []uint64{4, 5, 5},
		},
		{
			name: "not divisible by factor 3, end +Inf",
			hist: metrics.Float64Histogram{
				Buckets: []float64{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, math.Inf(1)},
				Counts:  []uint64{2, 2, 1, 4, 5, 3, 2, 1, 7, 13},
			},
			factor:      3,
			wantBuckets: []float64{3, 6, 9, 12},
			wantCounts:  []uint64{5, 12, 10, 13},
		},
		{
			name: "not divisible by factor 3, start/end +Inf",
			hist: metrics.Float64Histogram{
				Buckets: []float64{math.Inf(-1), 1, 2, 3, 4, 5, 6, 7, 8, 9, math.Inf(1)},
				Counts:  []uint64{2, 2, 1, 4, 5, 3, 2, 1, 7, 13},
			},
			factor:      3,
			wantBuckets: []float64{3, 6, 9, 12},
			wantCounts:  []uint64{5, 12, 10, 13},
		},
		{
			name: "divisible by factor 3, end +Inf",
			hist: metrics.Float64Histogram{
				Buckets: []float64{0, 1, 2, 3, 4, 5, 6, 7, 8, math.Inf(1)},
				Counts:  []uint64{2, 2, 1, 4, 5, 3, 2, 1, 7},
			},
			factor:      3,
			wantBuckets: []float64{3, 6, 9},
			wantCounts:  []uint64{5, 12, 10},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buckets := downsampleBuckets(&tt.hist, tt.factor)
			counts := make([]uint64, maxBuckets)
			counts = downsampleCounts(&tt.hist, tt.factor, counts)

			if !reflect.DeepEqual(buckets, tt.wantBuckets) {
				t.Errorf("downsampleBuckets() = %v, want %v", buckets, tt.wantBuckets)
			}
			if !reflect.DeepEqual(counts, tt.wantCounts) {
				t.Errorf("downsampleCounts() = %v, want %v", counts, tt.wantCounts)
			}
		})
	}
}

func TestHistogramTicks(t *testing.T) {
	buckets := []float64{1, 4, 10}
	labels := []float64{0.5, 3, 5, 11}

	values, texts := histogramTicks(buckets, labels)
	if want := []float64{0, 1}; !reflect.DeepEqual(values, want) {
		t.Fatalf("histogramTicks() values = %v, want %v", values, want)
	}
	if want := []float64{0.5, 3}; !reflect.DeepEqual(texts, want) {
		t.Fatalf("histogramTicks() texts = %v, want %v", texts, want)
	}
}

func TestHistogramTicksEmpty(t *testing.T) {
	values, texts := histogramTicks(nil, []float64{1})
	if values != nil || texts != nil {
		t.Fatalf("histogramTicks() = (%v, %v), want (nil, nil)", values, texts)
	}
}

func TestDurationHistogramTicksMatchRuntimeBuckets(t *testing.T) {
	samples := []metrics.Sample{{Name: "/sched/latencies:seconds"}}
	metrics.Read(samples)
	hist := samples[0].Value.Float64Histogram()
	buckets := downsampleBuckets(hist, downsampleFactor(hist, maxBuckets))
	values, texts := durationHistogramTicks(buckets)

	for i, value := range values {
		idx := int(value)
		if value != float64(idx) || idx < 0 || idx >= len(buckets) {
			t.Fatalf("tick value %v is outside %d runtime buckets", value, len(buckets))
		}
		distance := math.Abs(buckets[idx] - texts[i])
		if idx > 0 && math.Abs(buckets[idx-1]-texts[i]) < distance {
			t.Fatalf("tick %v at bucket %d is closer to preceding bucket", texts[i], idx)
		}
		if idx+1 < len(buckets) && math.Abs(buckets[idx+1]-texts[i]) < distance {
			t.Fatalf("tick %v at bucket %d is closer to following bucket", texts[i], idx)
		}
	}
}
