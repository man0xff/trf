package filter

import (
	"regexp"
	"strings"
	"time"
)

type Extractor struct {
	regexp *regexp.Regexp
	format string
}

func convertTimeFormat(f string) string {
	subs := []struct{ orig, new string }{
		{"%a", "Mon"},
		{"%A", "Monday"},
		{"%d", "02"},
		{"%_d", "_2"},
		{"%b", "Jan"},
		{"%B", "January"},
		{"%m", "01"},
		{"%y", "06"},
		{"%Y", "2006"},
		{"%H", "15"},
		{"%I", "03"},
		{"%p", "PM"},
		{"%M", "04"},
		{"%S", "05"},
		{"%f", ".000000"},
		{"%j", "002"},
	}
	for _, s := range subs {
		f = strings.ReplaceAll(f, s.orig, s.new)
	}
	return f
}

func NewExtractor(expr, format string) (Extractor, error) {
	var err error
	e := Extractor{}
	e.format = convertTimeFormat(format)

	if expr != "" {
		if e.regexp, err = regexp.Compile(expr); err != nil {
			return e, err
		}
	}
	return e, nil
}

func (e *Extractor) extract(s string) (time.Time, bool) {
	var (
		err error
		t   time.Time
	)

	if e.regexp != nil {
		m := e.regexp.FindStringSubmatch(s)
		switch len(m) {
		case 0:
			return t, false
		case 1:
			s = m[0]
		default:
			s = m[1]
		}
	}
	t, err = time.ParseInLocation(e.format, s, time.Now().Location())
	if err != nil {
		return t, false
	}
	if t.Year() == 0 {
		now := time.Now()
		t = time.Date(now.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(),
			t.Second(), t.Nanosecond(), t.Location())
	}
	return t, true
}
