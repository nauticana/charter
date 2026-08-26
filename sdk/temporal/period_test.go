package temporal

import (
	"testing"
	"time"

	"github.com/nauticana/charter/sdk/model"
)

func TestPeriodFailsClosedOnUnparsableDates(t *testing.T) {
	at := time.Date(2026, 6, 18, 12, 0, 0, 0, time.UTC)
	if NewPeriod(model.Validity{From: "yesterday"}).Contains(at) {
		t.Error("unparsable from accepted")
	}
	if !NewPeriod(model.Validity{From: "2026-06-18"}).Contains(at) || NewPeriod(model.Validity{From: "2026-06-19"}).Contains(at) {
		t.Error("from boundary wrong")
	}
	if NewPeriod(model.Validity{From: "2026-01-01", To: "2025-12-31"}).Ordered() {
		t.Error("reversed validity reported ordered")
	}
	if !EffectiveAt(nil, at) || EffectiveAt(&model.Validity{From: "2026-01-01", To: "2026-06-17"}, at) {
		t.Error("EffectiveAt wrong")
	}
}

func TestWindow(t *testing.T) {
	from := time.Date(2026, 6, 18, 16, 0, 0, 0, time.UTC)
	to := from.Add(24 * time.Hour)
	w := Window{From: from, To: &to}
	if !w.Contains(from) || !w.Contains(to) || w.Contains(from.Add(-time.Second)) || w.Contains(to.Add(time.Second)) {
		t.Error("bounded window wrong")
	}
	if !(Window{From: from}).Contains(to.AddDate(10, 0, 0)) {
		t.Error("open window wrong")
	}
}
