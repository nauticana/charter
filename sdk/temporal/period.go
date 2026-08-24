// Package temporal evaluates Charter dates and validity periods.
package temporal

import (
	"time"

	"github.com/nauticana/charter/sdk/model"
)

// Period evaluates one effective-dated validity; unparsable dates fail closed.
type Period struct {
	Validity model.Validity
}

func NewPeriod(v model.Validity) Period { return Period{Validity: v} }

func (p Period) date(d model.Date) (time.Time, error) {
	return time.Parse("2006-01-02", string(d))
}

// Contains reports whether the calendar date of t falls inside the period.
func (p Period) Contains(t time.Time) bool {
	from, err := p.date(p.Validity.From)
	if err != nil {
		return false
	}
	day := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
	if day.Before(from) {
		return false
	}
	if p.Validity.To == "" {
		return true
	}
	to, err := p.date(p.Validity.To)
	return err == nil && !day.After(to)
}

func (p Period) Ordered() bool {
	if p.Validity.To == "" {
		return true
	}
	from, err1 := p.date(p.Validity.From)
	to, err2 := p.date(p.Validity.To)
	return err1 == nil && err2 == nil && !to.Before(from)
}
