package plot

import (
	"runtime/metrics"
	"time"
)

var _ = register(description{
	metrics: []string{
		"/gc/heap/objects:objects",
	},
	getvalues: func() getvalues {
		return func(_ time.Time, samples []metrics.Sample) any {
			gcHeapObjects := samples[idx_gc_heap_objects_objects].Value.Uint64()

			return []uint64{gcHeapObjects}
		}
	},
	layout: Scatter{
		Name:   "live-objects",
		Tags:   []tag{tagGC},
		Title:  "Heap Objects",
		Type:   "bar",
		Events: "lastgc",
		Layout: ScatterLayout{
			Yaxis: ScatterYAxis{
				Title: "objects",
			},
		},
		Subplots: []Subplot{
			{
				Name:    "live or unswept objects",
				Unitfmt: "%{y:.4s}",
				Color:   RGBString(255, 195, 128),
			},
		},
		InfoText: `<i>Heap objects</i> is <b>/gc/heap/objects:objects</b>. It is the number of live or unswept objects occupying heap memory.`,
	},
})
