package plot

import (
	"errors"
	"strings"
	"testing"
)

type auditFailedWriter struct{}

func (auditFailedWriter) Write([]byte) (int, error) { return 0, errors.New("owned writer failure") }

func TestAuditPlotListErrors(t *testing.T) {
	for _, name := range []string{"timestamp", "lastgc", "size-classes", "goroutines"} {
		if !IsReservedPlotName(name) {
			t.Errorf("runtime name %q not reserved", name)
		}
	}
	if IsReservedPlotName("owned-user-plot") {
		t.Fatal("user name reserved")
	}
	user := UserPlot{Scatter: &ScatterUserPlot{Plot: Scatter{Name: "owned-user-plot"}}}
	if _, err := NewList([]UserPlot{user, user}); err == nil {
		t.Fatal("duplicate plots accepted")
	}
	list, err := NewList([]UserPlot{user})
	if err != nil {
		t.Fatal(err)
	}
	list.Config()
	if n, err := list.WriteTo(auditFailedWriter{}); n != 0 || err == nil || !strings.Contains(err.Error(), "owned writer failure") {
		t.Fatalf("write failure = %d, %v", n, err)
	}
}
