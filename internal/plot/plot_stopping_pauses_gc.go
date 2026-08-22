package plot

import (
	"runtime/metrics"
	"time"
)

var _ = register(description{
	metrics: []string{
		"/sched/pauses/stopping/gc:seconds",
	},
	getvalues: func() getvalues {
		var delta cumulativeHistogramDelta

		return func(_ time.Time, samples []metrics.Sample) any {
			hist := samples[idx_sched_pauses_stopping_gc_seconds].Value.Float64Histogram()
			return delta.next(hist)
		}
	},
	layout: func(samples []metrics.Sample) Heatmap {
		hist := samples[idx_sched_pauses_stopping_gc_seconds].Value.Float64Histogram()
		histfactor := downsampleFactor(hist, maxBuckets)
		buckets := downsampleBuckets(hist, histfactor)
		tickVals, tickText := durationHistogramTicks(buckets)

		return Heatmap{
			Name:       "stopping-pauses-gc",
			Tags:       []tag{tagScheduler, tagGC},
			Title:      "Stop-the-world Stopping Latencies (GC)",
			Type:       "heatmap",
			UpdateFreq: 5,
			Colorscale: GreenShades,
			Buckets:    floatseq(len(buckets)),
			CustomData: buckets,
			Hover: HeapmapHover{
				YName: "stopping duration",
				YUnit: "duration",
				ZName: "pauses per interval",
			},
			Layout: HeatmapLayout{
				YAxis: HeatmapYaxis{
					Title:    "stopping duration",
					TickMode: "array",
					TickVals: tickVals,
					TickText: tickText,
				},
			},
			InfoText: `This heatmap shows the distribution of individual <b>GC-related</b> stop-the-world <i>stopping latencies</i>.
This is the time it takes from deciding to stop the world until all Ps are stopped.
During this time, some threads may be executing.
Each column contains pauses observed since the preceding metrics sample.
Uses <b>/sched/pauses/stopping/gc:seconds</b>.`,
		}
	},
})
