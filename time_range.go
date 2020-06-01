package main

import (
	"errors"
	"strings"
	"time"

	"trf/filter"
)

const (
	fDate  = "2006-01-02"
	fTime  = "15:04:05"
	fParse = fDate + "T" + fTime
)

type point struct {
	inf    bool
	val    time.Time
	offset time.Duration
}

type cliTimeRange struct {
	filter.TimeRange
}

var (
	errIncorrectRange = errors.New("incorrect time range")
)

func newPoint(s string) (point, error) {
	var (
		err   error
		p     point
		now   = time.Now()
		runes = []rune(s)
		loc   = now.Location()
	)

	switch {
	case s == "":
		p.val = now
		return p, nil

	case s == "-":
		p.inf = true
		return p, nil

	case s[0] == '+' || s[0] == '-':
		p.offset, err = time.ParseDuration(s[1:])
		if err != nil {
			return p, err
		}
		if s[0] == '-' {
			p.offset = -p.offset
		}
		return p, nil
	}

	if len(runes) == len(fParse) {
		runes[len(fDate)] = 'T'
		p.val, err = time.ParseInLocation(fParse, string(runes), loc)
		if err == nil {
			return p, nil
		}
	}
	if p.val, err = time.ParseInLocation(fDate, s, loc); err == nil {
		return p, nil
	}
	v, err := time.ParseInLocation(fTime, s, loc)
	if err != nil {
		return p, err
	}
	p.val = withTime(&now, v.Hour(), v.Minute(), v.Second())
	return p, nil
}

func (p *point) isConcrete() bool {
	return !p.inf && !p.val.IsZero()
}

func (p *point) resolve(ref *time.Time) time.Time {
	switch {
	case p.inf:
		return time.Time{}
	case !p.val.IsZero():
		return p.val
	}
	return ref.Add(p.offset)
}

func (r *cliTimeRange) Set(s string) error {
	var (
		err error
		now = time.Now()
	)

	if s == "today" {
		r.From, r.To = withTime(&now, 0, 0, 0), withTime(&now, 23, 59, 59)
		return nil

	} else if s == "yesterday" {
		y := now.Add(-24 * time.Hour)
		r.From, r.To = withTime(&y, 0, 0, 0), withTime(&y, 23, 59, 59)
		return nil
	}

	parts := strings.Split(s, ",")
	if len(parts) > 2 {
		return errIncorrectRange
	}
	if len(parts) == 1 {
		parts = append(parts, "")
	}

	from, err := newPoint(parts[0])
	if err != nil {
		return errIncorrectRange
	}
	to, err := newPoint(parts[1])
	if err != nil {
		return errIncorrectRange
	}

	ref := now
	switch {
	case from.isConcrete():
		ref = from.val
	case to.isConcrete():
		ref = to.val
	}
	r.From = from.resolve(&ref)
	r.To = to.resolve(&ref)

	if !r.From.IsZero() && !r.To.IsZero() && r.From.After(r.To) {
		return errIncorrectRange
	}
	return nil
}

func withTime(t *time.Time, hour, min, sec int) time.Time {
	nano := 0
	if sec == 59 {
		nano = 999999999
	}
	return time.Date(t.Year(), t.Month(), t.Day(),
		hour, min, sec, nano, t.Location())
}
