package plot

import (
	"math"
	"runtime/metrics"
	"sort"
)

// maxBuckets is the maximum number of buckets we'll plots in heatmaps.
// Histograms with more buckets than that are going to be downsampled.
const maxBuckets = 100

// cumulativeHistogramDelta converts cumulative runtime histogram counts into
// nonnegative counts observed since the previous sample.
type cumulativeHistogramDelta struct {
	factor      int
	length      int
	initialized bool
	previous    [maxBuckets]uint64
	values      [maxBuckets]uint64
}

func (d *cumulativeHistogramDelta) next(hist *metrics.Float64Histogram) []uint64 {
	factor := downsampleFactor(hist, maxBuckets)
	values := downsampleCounts(hist, factor, d.values[:])
	if !d.initialized || d.factor != factor || d.length != len(values) {
		d.factor = factor
		d.length = len(values)
		d.initialized = true
		copy(d.previous[:], values)
		clear(values)
		return values
	}

	for i, current := range values {
		previous := d.previous[i]
		d.previous[i] = current
		if current >= previous {
			values[i] = current - previous
		}
		// A lower cumulative count means the runtime counter reset or wrapped.
		// Treat the current value as the count observed since that reset.
	}
	return values
}

// downsampleFactor computes the downsampling factor to use with
// downsampleCounts and downsampleBuckets. nfinal is the maximum number of
// buckets in the resulting histogram.
func downsampleFactor(h *metrics.Float64Histogram, nfinal int) int {
	norg := len(h.Counts)
	mod := norg % nfinal
	if mod == 0 {
		return norg / nfinal
	}
	return 1 + norg/nfinal
}

// downsampleBuckets downsamples the number of buckets in the provided
// histogram, using the given dividing factor, and returns a slice of bucket
// widths.
//
// Given that metrics.Float64Histogram contains the boundaries of histogram
// buckets, the first bucket is not even considered since we're only interested
// in upper bounds. Also, since we can't draw an infinitely large bucket, if h
// last bucket holds +Inf, the width of the last returned bucket will
// extrapolated from the previous 2 buckets.
func downsampleBuckets(h *metrics.Float64Histogram, factor int) []float64 {
	var ret []float64
	vals := h.Buckets[1:]

	for i := range vals {
		if (i+1)%factor == 0 {
			ret = append(ret, vals[i])
		}
	}
	if len(vals)%factor != 0 {
		// If the number of bucket is not divisible by the factor, let's make a
		// last downsampled bucket, even if it doesn't 'contain' the same number
		// of original buckets.
		ret = append(ret, vals[len(vals)-1])
	}

	if len(ret) > 2 && math.IsInf(ret[len(ret)-1], 1) {
		// Plotly doesn't support a +Inf bound for the last bucket. So we make it
		// so that the last bucket has the same 'width' than the penultimate one.
		ret[len(ret)-1] = ret[len(ret)-2] - ret[len(ret)-3] + ret[len(ret)-2]
	}

	return ret
}

// downsampleCounts downsamples the counts in the provided histogram, using the
// given factor. Every 'factor' buckets are merged into one, larger, bucket. If
// the number of buckets is not divisible by 'factor', then an additional last
// bucket will contain the sum of the counts in all relainbing buckets.
//
// Note: slice should be a slice of maxBuckets elements, so that it can be
// reused across calls.
func downsampleCounts(h *metrics.Float64Histogram, factor int, slice []uint64) []uint64 {
	vals := h.Counts

	if factor == 1 {
		copy(slice, vals)
		slice = slice[:len(vals)]
		return slice
	}

	slice = slice[:0]

	var sum uint64
	for i := range vals {
		if i%factor == 0 && i > 1 {
			slice = append(slice, sum)
			sum = vals[i]
		} else {
			sum += vals[i]
		}
	}

	// Whatever sum remains, it goes to the last bucket.
	return append(slice, sum)
}

func histogramTicks(buckets, labels []float64) ([]float64, []float64) {
	if len(buckets) == 0 {
		return nil, nil
	}

	values := make([]float64, 0, len(labels))
	texts := make([]float64, 0, len(labels))
	for _, label := range labels {
		if label > buckets[len(buckets)-1] {
			continue
		}

		idx := sort.SearchFloat64s(buckets, label)
		if idx > 0 && math.Abs(buckets[idx-1]-label) < math.Abs(buckets[idx]-label) {
			idx--
		}
		if len(values) > 0 && values[len(values)-1] == float64(idx) {
			continue
		}

		values = append(values, float64(idx))
		texts = append(texts, label)
	}
	return values, texts
}

func durationHistogramTicks(buckets []float64) ([]float64, []float64) {
	labels := []float64{1e-7, 1e-6, 1e-5, 1e-4, 1e-3, 5e-3, 1e-2, 5e-2, 1e-1, 5e-1, 1, 5, 10}
	return histogramTicks(buckets, labels)
}

func floatseq(n int) []float64 {
	seq := make([]float64, n)
	for i := range n {
		seq[i] = float64(i)
	}
	return seq
}
