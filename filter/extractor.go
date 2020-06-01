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
		{"%Y", "2006"},
		{"%m", "01"},
		{"%d", "02"},
		{"%H", "15"},
		{"%M", "04"},
		{"%S", "05"},
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
	var err error
	t := time.Time{}

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
	return t, true
}
