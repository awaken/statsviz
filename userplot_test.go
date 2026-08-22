package statsviz

import (
	"errors"
	"testing"
)

func TestTimeSeriesPlotConfigErrors(t *testing.T) {
	t.Run("empty name", func(t *testing.T) {
		tsb := TimeSeriesPlotConfig{}
		if _, err := tsb.Build(); !errors.Is(err, ErrEmptyPlotName) {
			t.Errorf("Build() returned err = %v, want %v", err, ErrEmptyPlotName)
		}
	})
	t.Run("reserved name", func(t *testing.T) {
		tsb := TimeSeriesPlotConfig{Name: "timestamp"}
		var target ErrReservedPlotName
		if _, err := tsb.Build(); !errors.As(err, &target) {
			t.Errorf("Build() returned err = %v, want %v", err, target)
		}
	})
	t.Run("no time series", func(t *testing.T) {
		tsb := TimeSeriesPlotConfig{Name: "some name"}
		if _, err := tsb.Build(); !errors.Is(err, ErrNoTimeSeries) {
			t.Errorf("Build() returned err = %v, want %v", err, ErrNoTimeSeries)
		}
	})
	t.Run("nil value function", func(t *testing.T) {
		tsb := TimeSeriesPlotConfig{
			Name:   "some name",
			Series: []TimeSeries{{Name: "some series"}},
		}
		if _, err := tsb.Build(); !errors.Is(err, ErrNilGetValue) {
			t.Errorf("Build() returned err = %v, want %v", err, ErrNilGetValue)
		}
	})
	t.Run("zero time series plot", func(t *testing.T) {
		srv, err := NewServer(TimeseriesPlot(TimeSeriesPlot{}))
		if !errors.Is(err, ErrInvalidTimeSeriesPlot) {
			t.Errorf("NewServer() returned err = %v, want %v", err, ErrInvalidTimeSeriesPlot)
		}
		if srv != nil {
			srv.Close()
		}
	})
	t.Run("invalid plot type", func(t *testing.T) {
		cfg := TimeSeriesPlotConfig{
			Name:   "some name",
			Type:   TimeSeriesType("invalid"),
			Series: []TimeSeries{{Name: "some series", GetValue: func() float64 { return 0 }}},
		}
		if _, err := cfg.Build(); err == nil {
			t.Error("Build() accepted an invalid plot type")
		}
	})
	t.Run("invalid bar mode", func(t *testing.T) {
		cfg := TimeSeriesPlotConfig{
			Name:    "some name",
			BarMode: BarMode("invalid"),
			Series:  []TimeSeries{{Name: "some series", GetValue: func() float64 { return 0 }}},
		}
		if _, err := cfg.Build(); err == nil {
			t.Error("Build() accepted an invalid bar mode")
		}
	})
	t.Run("invalid series type", func(t *testing.T) {
		cfg := TimeSeriesPlotConfig{
			Name: "some name",
			Series: []TimeSeries{{
				Name:     "some series",
				Type:     TimeSeriesType("invalid"),
				GetValue: func() float64 { return 0 },
			}},
		}
		if _, err := cfg.Build(); err == nil {
			t.Error("Build() accepted an invalid series type")
		}
	})
}

func TestTimeSeriesPlotConfigDefaults(t *testing.T) {
	userPlot, err := (TimeSeriesPlotConfig{
		Name:   "some name",
		Series: []TimeSeries{{Name: "some series", GetValue: func() float64 { return 0 }}},
	}).Build()
	if err != nil {
		t.Fatal(err)
	}

	layout := userPlot.timeseries.Plot
	if layout.Type != string(Scatter) {
		t.Errorf("plot type = %q, want %q", layout.Type, Scatter)
	}
	if layout.Layout.BarMode != string(Group) {
		t.Errorf("bar mode = %q, want %q", layout.Layout.BarMode, Group)
	}
}
