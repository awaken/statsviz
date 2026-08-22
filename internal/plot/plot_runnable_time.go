package plot

import (
	"runtime/metrics"
	"time"
)

var _ = register(description{
	metrics: []string{
		"/sched/latencies:seconds",
	},
	getvalues: func() getvalues {
		var delta cumulativeHistogramDelta

		return func(_ time.Time, samples []metrics.Sample) any {
			hist := samples[idx_sched_latencies_seconds].Value.Float64Histogram()
			return delta.next(hist)
		}
	},
	layout: func(samples []metrics.Sample) Heatmap {
		hist := samples[idx_sched_latencies_seconds].Value.Float64Histogram()
		histfactor := downsampleFactor(hist, maxBuckets)
		buckets := downsampleBuckets(hist, histfactor)
		tickVals, tickText := durationHistogramTicks(buckets)

		return Heatmap{
			Name:       "runnable-time",
			Tags:       []tag{tagScheduler},
			Title:      "Time Goroutines Spend in 'Runnable' state",
			Type:       "heatmap",
			UpdateFreq: 5,
			Colorscale: GreenShades,
			Buckets:    floatseq(len(buckets)),
			CustomData: buckets,
			Hover: HeapmapHover{
				YName: "duration",
				YUnit: "duration",
				ZName: "scheduling events per interval",
			},
			Layout: HeatmapLayout{
				YAxis: HeatmapYaxis{
					Title:    "duration",
					TickMode: "array",
					TickVals: tickVals,
					TickText: tickText,
				},
			},
			InfoText: `This heatmap shows the distribution of the time goroutines have spent in the scheduler in a runnable state before actually running. Each column contains scheduling events observed since the preceding metrics sample. It uses <b>/sched/latencies:seconds</b>.`,
		}
	},
})
