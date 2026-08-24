package authority

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"

	"github.com/nauticana/charter/sdk/model"
)

// LimitEvaluator compares typed limits against measured values; missing measures and unit or currency mismatches fail closed.
type LimitEvaluator struct{}

func (LimitEvaluator) Check(limits []model.Limit, measures map[string]Measure) string {
	var e LimitEvaluator
	for _, l := range limits {
		m, ok := measures[l.LimitKind]
		if !ok {
			return "no measure for limit " + l.LimitKind
		}
		if l.Currency != "" && l.Currency != m.Currency {
			return fmt.Sprintf("limit %s currency %s, measured %s", l.LimitKind, l.Currency, m.Currency)
		}
		if l.Currency != "" {
			if l.CurrencyExponent == nil || *l.CurrencyExponent != m.CurrencyExponent {
				return fmt.Sprintf("limit %s currency exponent differs", l.LimitKind)
			}
			bound, boundOK := e.integer(l.Value)
			value, valueOK := e.integer(m.Value)
			if !boundOK || !valueOK {
				return "limit " + l.LimitKind + " currency value is not integer minor units"
			}
			if !e.compareInt(value, l.Operator, bound) {
				return fmt.Sprintf("limit %s: %d not %s %d minor units", l.LimitKind, value, l.Operator, bound)
			}
			continue
		}
		if l.Unit != "" && l.Unit != m.Unit {
			return fmt.Sprintf("limit %s unit %s, measured %s", l.LimitKind, l.Unit, m.Unit)
		}
		if reason := e.compareScalar(l, m.Value); reason != "" {
			return reason
		}
	}
	return ""
}

func (LimitEvaluator) integer(v any) (int64, bool) {
	switch n := v.(type) {
	case int:
		return int64(n), true
	case int64:
		return n, true
	case json.Number:
		i, err := n.Int64()
		return i, err == nil
	case float64:
		i := int64(n)
		return i, float64(i) == n
	}
	return 0, false
}

func (e LimitEvaluator) compareScalar(l model.Limit, measured any) string {
	if bound, ok := e.number(l.Value); ok {
		value, valueOK := e.number(measured)
		if !valueOK {
			return "measure for limit " + l.LimitKind + " is not numeric"
		}
		if !e.compareNumber(value, l.Operator, bound) {
			return fmt.Sprintf("limit %s: %v not %s %v", l.LimitKind, value, l.Operator, bound)
		}
		return ""
	}
	if l.Operator != "eq" {
		return "limit " + l.LimitKind + " supports non-numeric values only with eq"
	}
	if !reflect.DeepEqual(l.Value, measured) {
		return fmt.Sprintf("limit %s: %v not eq %v", l.LimitKind, measured, l.Value)
	}
	return ""
}

func (LimitEvaluator) number(value any) (float64, bool) {
	switch v := value.(type) {
	case float64:
		return v, true
	case int:
		return float64(v), true
	case int64:
		return float64(v), true
	case json.Number:
		f, err := v.Float64()
		return f, err == nil
	case string:
		f, err := strconv.ParseFloat(v, 64)
		return f, err == nil
	}
	return 0, false
}

func (LimitEvaluator) compareNumber(v float64, op string, bound float64) bool {
	switch op {
	case "lt":
		return v < bound
	case "lte":
		return v <= bound
	case "eq":
		return v == bound
	case "gte":
		return v >= bound
	case "gt":
		return v > bound
	}
	return false
}

func (LimitEvaluator) compareInt(v int64, op string, bound int64) bool {
	switch op {
	case "lt":
		return v < bound
	case "lte":
		return v <= bound
	case "eq":
		return v == bound
	case "gte":
		return v >= bound
	case "gt":
		return v > bound
	}
	return false
}
