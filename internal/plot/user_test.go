package plot

import (
	"bytes"
	"encoding/json"
	"math"
	"slices"
	"testing"
)

func TestUserScatterPlotNonFiniteValuesAreGaps(t *testing.T) {
	value := math.NaN()
	list, err := NewList([]UserPlot{{
		Scatter: &ScatterUserPlot{
			Plot:  Scatter{Name: "user"},
			Funcs: []func() float64{func() float64 { return value }},
		},
	}})
	if err != nil {
		t.Fatal(err)
	}
	list.Config()

	writeUserSeries := func() string {
		t.Helper()
		var buf bytes.Buffer
		if _, err := list.WriteTo(&buf); err != nil {
			t.Fatal(err)
		}
		var event struct {
			Data struct {
				Series map[string]json.RawMessage `json:"series"`
			} `json:"data"`
		}
		if err := json.Unmarshal(buf.Bytes(), &event); err != nil {
			t.Fatal(err)
		}
		return string(event.Data.Series["user"])
	}

	for _, nonFinite := range []float64{math.NaN(), math.Inf(1), math.Inf(-1)} {
		value = nonFinite
		if got := writeUserSeries(); got != "[null]" {
			t.Fatalf("non-finite user series = %s; want [null]", got)
		}
		value = 42
		if got := writeUserSeries(); got != "[42]" {
			t.Fatalf("recovered user series = %s; want [42]", got)
		}
	}
}

func TestUserScatterPlotDefaultsToMiscTag(t *testing.T) {
	layout := UserPlot{
		Scatter: &ScatterUserPlot{Plot: Scatter{Name: "user"}},
	}.Layout().(Scatter)

	if want := []string{tagMisc}; !slices.Equal(layout.Tags, want) {
		t.Fatalf("Layout().Tags = %v, want %v", layout.Tags, want)
	}
}

func Test_hasDuplicatePlotNames(t *testing.T) {
	tests := []struct {
		name  string
		plots []UserPlot
		want  string
	}{
		{
			"nil",
			nil,
			"",
		},
		{
			"empty",
			[]UserPlot{},
			"",
		},
		{
			"single scatter",
			[]UserPlot{
				{Scatter: &ScatterUserPlot{Plot: Scatter{Name: "a"}}},
			},
			"",
		},
		{
			"single heatmap",
			[]UserPlot{
				{Heatmap: &HeatmapUserPlot{Plot: Heatmap{Name: "a"}}},
			},
			"",
		},
		{
			"two scatter",
			[]UserPlot{
				{Scatter: &ScatterUserPlot{Plot: Scatter{Name: "a"}}},
				{Heatmap: &HeatmapUserPlot{Plot: Heatmap{Name: "b"}}},
				{Scatter: &ScatterUserPlot{Plot: Scatter{Name: "a"}}},
			},
			"a",
		},
		{
			"two heatmap",
			[]UserPlot{
				{Heatmap: &HeatmapUserPlot{Plot: Heatmap{Name: "a"}}},
				{Scatter: &ScatterUserPlot{Plot: Scatter{Name: "b"}}},
				{Heatmap: &HeatmapUserPlot{Plot: Heatmap{Name: "a"}}},
			},
			"a",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := hasDuplicatePlotNames(tt.plots); got != tt.want {
				t.Errorf("hasDuplicatePlotNames() = %q, want %q", got, tt.want)
			}
		})
	}
}
