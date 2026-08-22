package statsviz

import (
	"errors"
	"fmt"

	"github.com/arl/statsviz/internal/plot"
)

// TimeSeriesType describes the type of a time series plot.
type TimeSeriesType string

const (
	// Scatter is a time series plot made of lines.
	Scatter TimeSeriesType = "scatter"

	// Bar is a time series plot made of bars.
	Bar TimeSeriesType = "bar"
)

// BarMode determines how bars at the same location are displayed on a bar plot.
type BarMode string

const (
	// Stack indicates that bars are stacked on top of one another.
	Stack BarMode = "stack"

	// Group indicates that bars are plotted next to one another, centered
	// around the shared location.
	Group BarMode = "group"

	// Relative indicates that bars are stacked on top of one another, with
	// negative values below the axis and positive values above.
	Relative BarMode = "relative"

	// Overlay indicates that bars are plotted over one another.
	Overlay BarMode = "overlay"
)

var (
	// ErrNoTimeSeries is returned when a user plot has no time series.
	ErrNoTimeSeries = errors.New("user plot must have at least one time series")

	// ErrEmptyPlotName is returned when a user plot has an empty name.
	ErrEmptyPlotName = errors.New("user plot name can't be empty")

	// ErrNilGetValue is returned when a time series has no value function.
	ErrNilGetValue = errors.New("time series GetValue function can't be nil")

	// ErrInvalidTimeSeriesPlot is returned for a zero TimeSeriesPlot.
	ErrInvalidTimeSeriesPlot = errors.New("time series plot is not initialized")
)

// ErrReservedPlotName is returned when a reserved plot name is used for a user plot.
type ErrReservedPlotName string

func (e ErrReservedPlotName) Error() string {
	return fmt.Sprintf("%q is a reserved plot name", string(e))
}

// HoverOnType describes the type of hover effect on a time series plot.
type HoverOnType string

const (
	// HoverOnPoints specifies that hover effects highlight individual
	// points.
	HoverOnPoints HoverOnType = "points"

	// HoverOnFills specifies that hover effects highlight filled regions.
	HoverOnFills HoverOnType = "fills"

	// HoverOnPointsAndFills specifies that hover effects highlight both
	// points and filled regions.
	HoverOnPointsAndFills HoverOnType = "points+fills"
)

// A TimeSeries describes a single time series of a plot.
type TimeSeries struct {
	// Name is the name identifying this time series in the user interface.
	Name string

	// Unitfmt is the Plotly hover-template fragment used to format this time
	// series in the user interface. Numeric placeholders accept d3-format
	// specifiers; for example, "%{y:.4s}B" formats y with SI-prefix precision.
	Unitfmt string

	// HoverOn configures whether the hover effect highlights individual points,
	// filled regions, or both. It defaults to [HoverOnFills].
	HoverOn HoverOnType

	// Type is the time series type, either [Scatter] or [Bar]. It defaults to the
	// containing plot's Type.
	Type TimeSeriesType

	// GetValue specifies the function called to get the value of this time
	// series. A non-finite result is sent as null and rendered as a gap; later
	// finite results continue the series normally.
	GetValue func() float64
}

// TimeSeriesPlotConfig describes the configuration of a time series plot.
type TimeSeriesPlotConfig struct {
	// Name is the plot name, it must be unique.
	Name string

	// Title is the plot title, shown above the plot.
	Title string

	// Type is either [Scatter] or [Bar]. It defaults to [Scatter].
	Type TimeSeriesType

	// BarMode is either [Stack], [Group], [Relative] or [Overlay].
	// It defaults to [Group].
	BarMode BarMode

	// InfoText is the HTML-aware text shown when the user clicks on the plot
	// Info icon.
	InfoText string

	// YAxisTitle is the title of Y axis.
	YAxisTitle string

	// YAxisTickSuffix is the suffix added to tick values.
	YAxisTickSuffix string

	// Series contains the time series shown on this plot, there must be at
	// least one.
	Series []TimeSeries
}

// Build validates the configuration and builds a time series plot for it.
func (p TimeSeriesPlotConfig) Build() (TimeSeriesPlot, error) {
	var zero TimeSeriesPlot
	if p.Name == "" {
		return zero, ErrEmptyPlotName
	}
	if plot.IsReservedPlotName(p.Name) {
		return zero, ErrReservedPlotName(p.Name)
	}
	if len(p.Series) == 0 {
		return zero, ErrNoTimeSeries
	}
	switch p.Type {
	case "":
		p.Type = Scatter
	case Scatter, Bar:
		// ok
	default:
		return zero, fmt.Errorf("invalid plot type %q", p.Type)
	}
	switch p.BarMode {
	case "":
		p.BarMode = Group
	case Stack, Group, Relative, Overlay:
		// ok
	default:
		return zero, fmt.Errorf("invalid bar mode %q", p.BarMode)
	}

	var (
		subplots []plot.Subplot
		funcs    []func() float64
	)
	for _, ts := range p.Series {
		switch ts.Type {
		case "", Scatter, Bar:
			// ok
		default:
			return zero, fmt.Errorf("time series %q has invalid type %q", ts.Name, ts.Type)
		}
		switch ts.HoverOn {
		case "":
			ts.HoverOn = HoverOnFills
		case HoverOnPoints, HoverOnFills, HoverOnPointsAndFills:
			// ok
		default:
			return zero, fmt.Errorf("time series %q has invalid HoverOn value %q", ts.Name, ts.HoverOn)
		}
		if ts.GetValue == nil {
			return zero, fmt.Errorf("time series %q: %w", ts.Name, ErrNilGetValue)
		}

		subplots = append(subplots, plot.Subplot{
			Name:    ts.Name,
			Unitfmt: ts.Unitfmt,
			HoverOn: string(ts.HoverOn),
			Type:    string(ts.Type),
		})
		funcs = append(funcs, ts.GetValue)
	}

	return TimeSeriesPlot{
		timeseries: &plot.ScatterUserPlot{
			Plot: plot.Scatter{
				Name:     p.Name,
				Title:    p.Title,
				Type:     string(p.Type),
				InfoText: p.InfoText,
				Layout: plot.ScatterLayout{
					BarMode: string(p.BarMode),
					Yaxis: plot.ScatterYAxis{
						Title:      p.YAxisTitle,
						TickSuffix: p.YAxisTickSuffix,
					},
				},
				Subplots: subplots,
			},
			Funcs: funcs,
		},
	}, nil
}

// TimeSeriesPlot is an opaque type representing a timeseries plot.
// A plot can be created with [TimeSeriesPlotConfig.Build].
type TimeSeriesPlot struct {
	timeseries *plot.ScatterUserPlot
}
