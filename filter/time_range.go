package filter

import (
	"time"
)

type TimeRange struct {
	From time.Time
	To   time.Time
}

const timeFormat = "2006-01-02 15:04:05"

func (r *TimeRange) String() string {
	from, to := "", ""
	if !r.From.IsZero() {
		from = r.From.Format(timeFormat)
	}
	if !r.To.IsZero() {
		to = r.To.Format(timeFormat)
	}

	switch {
	case from == "" && to == "":
		return "[-, -]"
	case from == "":
		return "[-, " + to + "]"
	case to == "":
		return "[" + from + ", -]"
	}
	return "[" + from + ", " + to + "]"
}

func (r *TimeRange) Intersects(other *TimeRange) bool {
	switch {
	case r.From.IsZero() && r.To.IsZero(),
		other.From.IsZero() && other.To.IsZero():
		return true
	case r.From.IsZero():
		if other.From.IsZero() {
			return true
		}
		return !other.To.Before(other.From)
	case r.To.IsZero():
		if other.To.IsZero() {
			return true
		}
		return !r.From.After(other.To)
	default:
		if r.From.After(other.To) || r.To.Before(other.From) {
			return false
		}
		return true
	}
}
