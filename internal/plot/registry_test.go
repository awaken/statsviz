package plot

import (
	"bytes"
	"encoding/json"
	"runtime/metrics"
	"testing"
	"time"
)

func registryTestNoPanic(t *testing.T, f func()) {
	t.Helper()
	defer func() {
		if v := recover(); v != nil {
			t.Errorf("missing metric caused panic: %v", v)
		}
	}()
	f()
}

func TestRegistryMissingMetrics(t *testing.T) {
	const present, missing = "/sched/goroutines:goroutines", "/fixture/missing:count"
	t.Run("index", func(t *testing.T) {
		r := &registry{allMetrics: map[string]bool{present: true}}
		registryTestNoPanic(t, func() {
			if got := r.mustidx(missing); got != -1 {
				t.Errorf("missing index = %d; want -1", got)
			}
		})
		if len(r.metrics) != 0 {
			t.Fatalf("missing metric added to sample list: %v", r.metrics)
		}
		if first, repeated := r.mustidx(present), r.mustidx(present); first != 0 || repeated != first || len(r.metrics) != 1 {
			t.Fatalf("available index is not stable: first=%d repeated=%d metrics=%v", first, repeated, r.metrics)
		}
	})
	for _, heatmap := range []bool{false, true} {
		name := "scatter"
		if heatmap {
			name = "heatmap"
		}
		t.Run(name, func(t *testing.T) {
			r := &registry{allMetrics: map[string]bool{present: true}}
			built := false
			var layout any = Scatter{Name: "unavailable"}
			if heatmap {
				layout = func([]metrics.Sample) Heatmap { built = true; return Heatmap{Name: "unavailable"} }
			}
			registryTestNoPanic(t, func() {
				r.register(description{metrics: []string{present, missing}, layout: layout})
			})
			if built || len(r.descriptions) != 0 || len(r.metrics) != 0 {
				t.Fatalf("unavailable plot was partly registered: built=%t descriptions=%d metrics=%v", built, len(r.descriptions), r.metrics)
			}
		})
	}
}

func TestRegistryPreparesLayoutMetrics(t *testing.T) {
	const metric = "/sched/goroutines:goroutines"
	r := &registry{allMetrics: map[string]bool{metric: true}}
	built := 0
	r.register(description{
		metrics: []string{metric},
		layout: func(samples []metrics.Sample) Heatmap {
			built++
			if len(samples) != 1 || samples[0].Name != metric || samples[0].Value.Kind() != metrics.KindUint64 {
				t.Errorf("layout received unprepared samples: %v", samples)
			}
			return Heatmap{Name: "available"}
		},
		getvalues: func() getvalues {
			return func(_ time.Time, samples []metrics.Sample) any { return []uint64{samples[0].Value.Uint64()} }
		},
	})
	if built != 1 || len(r.descriptions) != 1 {
		t.Fatalf("available plot: built=%d registered=%d", built, len(r.descriptions))
	}
	list := &List{reg: r, samples: r.newSamples()}
	if len(list.Config().Series) != 1 {
		t.Fatal("available plot was omitted")
	}
	var buf bytes.Buffer
	if _, err := list.WriteTo(&buf); err != nil || !json.Valid(buf.Bytes()) || !bytes.Contains(buf.Bytes(), []byte(`"available"`)) {
		t.Fatalf("available plot did not sample: %s, %v", &buf, err)
	}
}
