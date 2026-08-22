package plot

import (
	"runtime/metrics"
	"time"
)

var _ = register(description{
	metrics: []string{
		"/gc/heap/allocs:bytes",
		"/gc/heap/frees:bytes",
	},
	getvalues: func() getvalues {
		return func(_ time.Time, samples []metrics.Sample) any {
			allocBytes := samples[idx_gc_heap_allocs_bytes].Value.Uint64()
			freedBytes := samples[idx_gc_heap_frees_bytes].Value.Uint64()

			return []uint64{allocBytes - freedBytes}
		}
	},
	layout: Scatter{
		Name:   "live-bytes",
		Tags:   []tag{tagGC},
		Title:  "Heap Object Bytes",
		Type:   "bar",
		Events: "lastgc",
		Layout: ScatterLayout{
			Yaxis: ScatterYAxis{
				Title: "bytes",
			},
		},
		Subplots: []Subplot{
			{
				Name:    "heap object bytes",
				Unitfmt: "%{y:.4s}B",
				Color:   RGBString(135, 182, 218),
			},
		},
		InfoText: `<i>Heap object bytes</i> is <b>/gc/heap/allocs:bytes</b> - <b>/gc/heap/frees:bytes</b>. The result includes live objects and dead, unswept objects whose storage has not yet been freed by the GC.`,
	},
})
