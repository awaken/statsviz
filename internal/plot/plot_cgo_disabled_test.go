//go:build !cgo

package plot

import (
	"slices"
	"testing"
)

func TestCgoPlotDisabled(t *testing.T) {
	const metric = "/cgo/go-to-c-calls:calls"
	registry := reg()
	if slices.Contains(registry.metrics, metric) {
		t.Fatalf("disabled cgo metric %q is registered for sampling", metric)
	}
	for _, description := range registry.descriptions {
		if nameFromLayout(description.layout) == "cgo" {
			t.Fatal("cgo plot is registered when cgo is disabled")
		}
	}
}
