package main

import (
	"bufio"
	"io"
	"strings"

	"gopkg.in/alecthomas/kingpin.v2"
)

type readerInput struct {
	app    *app
	reader *bufio.Reader
}

func (q *readerInput) Read(path *string) bool {
	var (
		err error
		p   string
	)

	if p, err = q.reader.ReadString('\n'); err == io.EOF {
		return false
	}
	kingpin.FatalIfError(err, "")
	*path = strings.TrimRight(p, "\n")
	return true
}

type stringsInput struct {
	app     *app
	strings []string
	i       int
}

func (q *stringsInput) Read(path *string) bool {
	if q.i < len(q.strings) {
		*path = q.strings[q.i]
		q.i++
		return true
	}
	return false
}

func (a *app) newFileInput(r io.Reader) *readerInput {
	return &readerInput{
		app:    a,
		reader: bufio.NewReader(r),
	}
}

func (a *app) newStringsInput(s []string) *stringsInput {
	return &stringsInput{
		app:     a,
		strings: s,
	}
}
