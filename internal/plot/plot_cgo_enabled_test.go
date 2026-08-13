//go:build cgo

package plot

import (
	"slices"
	"testing"
)

func TestCgoPlotEnabled(t *testing.T) {
	const metric = "/cgo/go-to-c-calls:calls"
	registry := reg()
	if !slices.Contains(registry.metrics, metric) {
		t.Fatalf("enabled cgo metric %q is not registered for sampling", metric)
	}
	for _, description := range registry.descriptions {
		if nameFromLayout(description.layout) == "cgo" {
			return
		}
	}
	t.Fatal("cgo plot is not registered when cgo is enabled")
}
