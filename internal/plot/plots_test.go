package plot

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image/color"
	"os"
	"runtime/metrics"
	"slices"
	"strings"
	"sync"
	"testing"
	"text/tabwriter"
)

func TestColorSerialization(t *testing.T) {
	if got, want := RGBString(135, 182, 218), "rgb(135,182,218)"; got != want {
		t.Fatalf("RGBString() = %q; want %q", got, want)
	}

	got, err := json.Marshal(WeightedColor{
		Value: 0.5,
		Color: color.RGBA{R: 20, G: 181, B: 235, A: 255},
	})
	if err != nil {
		t.Fatal(err)
	}
	if want := `[0.5,"rgb(20,181,235)"]`; string(got) != want {
		t.Fatalf("weighted color = %s; want %s", got, want)
	}

	for _, palette := range [][]WeightedColor{BlueShades, PinkShades, GreenShades} {
		for _, weighted := range palette {
			if weighted.Color.A != 255 {
				t.Fatalf("palette color %#v is not opaque", weighted.Color)
			}
		}
	}
}

func TestHeapObjectPlotLabels(t *testing.T) {
	list, err := NewList(nil)
	if err != nil {
		t.Fatal(err)
	}

	want := map[string]struct {
		title   string
		subplot string
	}{
		"live-bytes":   {title: "Heap Object Bytes", subplot: "heap object bytes"},
		"live-objects": {title: "Heap Objects", subplot: "live or unswept objects"},
	}
	for _, series := range list.Config().Series {
		scatter, ok := series.(Scatter)
		if !ok {
			continue
		}
		expect, ok := want[scatter.Name]
		if !ok {
			continue
		}
		if scatter.Title != expect.title || len(scatter.Subplots) != 1 || scatter.Subplots[0].Name != expect.subplot {
			t.Errorf("%s labels = %q, %#v; want %q, %q", scatter.Name, scatter.Title, scatter.Subplots, expect.title, expect.subplot)
		}
		delete(want, scatter.Name)
	}
	for name := range want {
		t.Errorf("plot %q not found", name)
	}
}

func TestUnusedRuntimeMetrics(t *testing.T) {
	// This test just prints the metrics we're not using in any plot. It can't
	// fail, it's informational.
	used := make(map[string]bool)
	for _, d := range reg().descriptions {
		for _, m := range d.metrics {
			used[m] = true
		}
	}

	// Discard godebug metrics and used metrics.
	all := metrics.All()
	all = slices.DeleteFunc(all, func(desc metrics.Description) bool {
		return strings.HasPrefix(desc.Name, "/godebug/")
	})
	all = slices.DeleteFunc(all, func(desc metrics.Description) bool {
		return used[desc.Name]
	})

	if len(all) == 0 {
		t.Log("all metrics are used!")
		return
	}

	t.Log("some runtime metrics are not used by any plot:\n")

	w := tabwriter.NewWriter(os.Stderr, 0, 8, 2, ' ', 0)
	for _, m := range all {
		fmt.Fprintf(w, "\t%s\t%s\t%s\n", m.Name, kindstr(m.Kind), clampstr(m.Description))
	}
	w.Flush()
}

func TestApproximateCleanupQueueLength(t *testing.T) {
	tests := []struct {
		name     string
		queued   uint64
		executed uint64
		want     uint64
	}{
		{name: "empty", queued: 0, executed: 0, want: 0},
		{name: "queued", queued: 5, executed: 3, want: 2},
		{name: "inconsistent snapshot", queued: 3, executed: 5, want: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := approximateCleanupQueueLength(tt.queued, tt.executed); got != tt.want {
				t.Fatalf("approximateCleanupQueueLength(%d, %d) = %d; want %d", tt.queued, tt.executed, got, tt.want)
			}
		})
	}
}

func TestSizeClassesFiniteMinimum(t *testing.T) {
	list, err := NewList(nil)
	if err != nil {
		t.Fatal(err)
	}

	for _, series := range list.Config().Series {
		heatmap, ok := series.(Heatmap)
		if !ok || heatmap.Name != "size-classes" {
			continue
		}
		if heatmap.Hover.YMin == nil || *heatmap.Hover.YMin != 1 {
			t.Fatalf("size-classes minimum = %v; want 1", heatmap.Hover.YMin)
		}
		return
	}

	t.Fatal("size-classes heatmap not found")
}

func TestThreadsYAxisUnit(t *testing.T) {
	if !reg().allMetrics["/sched/threads/total:threads"] {
		t.Skip("thread metric requires Go 1.26")
	}

	list, err := NewList(nil)
	if err != nil {
		t.Fatal(err)
	}

	for _, series := range list.Config().Series {
		scatter, ok := series.(Scatter)
		if !ok || scatter.Name != "threads" {
			continue
		}
		if scatter.Layout.Yaxis.Title != "threads" {
			t.Fatalf("threads y-axis title = %q; want %q", scatter.Layout.Yaxis.Title, "threads")
		}
		return
	}

	t.Fatal("threads plot not found")
}

func TestGCPausePlotTitle(t *testing.T) {
	list, err := NewList(nil)
	if err != nil {
		t.Fatal(err)
	}

	for _, series := range list.Config().Series {
		heatmap, ok := series.(Heatmap)
		if !ok || heatmap.Name != "total-pauses-gc" {
			continue
		}
		if heatmap.Title != "Stop-the-world Pause Latencies (GC)" {
			t.Fatalf("total-pauses-gc title = %q", heatmap.Title)
		}
		return
	}

	t.Fatal("total-pauses-gc heatmap not found")
}

func TestListsWriteConcurrently(t *testing.T) {
	lists := make([]*List, 2)
	for i := range lists {
		var err error
		lists[i], err = NewList(nil)
		if err != nil {
			t.Fatal(err)
		}
		lists[i].Config()
	}
	if &lists[0].samples[0] == &lists[1].samples[0] {
		t.Fatal("independent lists share runtime metric sample storage")
	}

	workers := []*List{lists[0], lists[0], lists[1], lists[1]}
	start := make(chan struct{})
	errs := make(chan error, len(workers))
	var wg sync.WaitGroup
	for _, list := range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			for range 100 {
				var buf bytes.Buffer
				if _, err := list.WriteTo(&buf); err != nil {
					errs <- err
					return
				}
			}
		}()
	}
	close(start)
	wg.Wait()
	close(errs)

	for err := range errs {
		t.Error(err)
	}
}

func kindstr(k metrics.ValueKind) string {
	switch k {
	case metrics.KindUint64:
		return "uint64"
	case metrics.KindFloat64:
		return "float64"
	case metrics.KindFloat64Histogram:
		return "float64 histogram"
	default:
		return "unknown"
	}
}

func clampstr(s string) string {
	const maxlen = 80
	if len(s) > maxlen {
		return s[:maxlen-3] + "..."
	}
	return s
}
