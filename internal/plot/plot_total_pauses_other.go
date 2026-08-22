package plot

import (
	"runtime/metrics"
	"time"
)

var _ = register(description{
	metrics: []string{
		"/sched/pauses/total/other:seconds",
	},
	getvalues: func() getvalues {
		var delta cumulativeHistogramDelta

		return func(_ time.Time, samples []metrics.Sample) any {
			hist := samples[idx_sched_pauses_total_other_seconds].Value.Float64Histogram()
			return delta.next(hist)
		}
	},
	layout: func(samples []metrics.Sample) Heatmap {
		hist := samples[idx_sched_pauses_total_other_seconds].Value.Float64Histogram()
		histfactor := downsampleFactor(hist, maxBuckets)
		buckets := downsampleBuckets(hist, histfactor)
		tickVals, tickText := durationHistogramTicks(buckets)

		return Heatmap{
			Name:       "total-pauses-other",
			Tags:       []tag{tagScheduler},
			Title:      "Stop-the-world Pause Latencies (Other)",
			Type:       "heatmap",
			UpdateFreq: 5,
			Colorscale: PinkShades,
			Buckets:    floatseq(len(buckets)),
			CustomData: buckets,
			Hover: HeapmapHover{
				YName: "pause duration",
				YUnit: "duration",
				ZName: "pauses per interval",
			},
			Layout: HeatmapLayout{
				YAxis: HeatmapYaxis{
					Title:    "pause duration",
					TickMode: "array",
					TickVals: tickVals,
					TickText: tickText,
				},
			},
			InfoText: `This heatmap shows the distribution of individual <b>non-GC-related</b> stop-the-world <i>pause latencies</i>.
This is the time from deciding to stop the world until the world is started again.
Some of this time is spent getting all threads to stop (measured directly in <i>/sched/pauses/stopping/other:seconds</i>).
Each column contains pauses observed since the preceding metrics sample.
Uses <b>/sched/pauses/total/other:seconds</b>.`,
		}
	},
})
