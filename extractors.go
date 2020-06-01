package main

import (
	"strings"

	"trf/filter"
)

type extractors []*filter.Extractor

func (e *extractors) Set(s string) error {
	var (
		err error
		ex  filter.Extractor
	)

	parts := strings.SplitN(s, "@", 2)
	if len(parts) == 1 {
		ex, err = filter.NewExtractor("", parts[0])
	} else {
		ex, err = filter.NewExtractor(parts[0], parts[1])
	}
	if err != nil {
		return err
	}
	*e = append(*e, &ex)
	return nil
}

func (e *extractors) IsCumulative() bool {
	return true
}

func (e *extractors) String() string {
	return ""
}
