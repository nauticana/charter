package temporal

import (
	"time"

	"github.com/nauticana/charter/sdk/model"
)

// EffectiveAt reports whether an optional validity contains t; an absent validity is open-ended.
func EffectiveAt(v *model.Validity, t time.Time) bool {
	return v == nil || NewPeriod(*v).Contains(t)
}

// Window is a date-time interval with an optional end, such as an approval's issue and expiry instants.
type Window struct {
	From time.Time
	To   *time.Time
}

func (w Window) Contains(t time.Time) bool {
	if t.Before(w.From) {
		return false
	}
	return w.To == nil || !t.After(*w.To)
}
