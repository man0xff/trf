package filter

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/recoilme/pudge"
)

type Config struct {
	Concurrency int
	TimeRange   TimeRange
	CacheFile   string
	Extractors  []*Extractor
	Debug       bool
	NoCache     bool
	Lines       int
}

type Filter struct {
	config Config
	db     *pudge.Db
}

type cacheData struct {
	From  time.Time
	To    time.Time
	MTime time.Time
}

type Input interface {
	Read(*string) bool
}

func New(config *Config) *Filter {
	var err error

	f := &Filter{}
	f.config = *config
	if f.config.Concurrency <= 0 {
		f.config.Concurrency = 1
	}

	if !f.config.NoCache {
		f.db, err = pudge.Open(f.config.CacheFile, nil)
		if err != nil {
			f.error("database opening failed (file:'%s', reason:'%s')",
				f.config.CacheFile, err)
			f.db = nil
		}
	}
	return f
}

func (f *Filter) Close() {
	if f.db != nil {
		f.db.Close()
	}
}

func (f *Filter) error(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "error: "+format+"\n", args...)
}

func (f *Filter) debug(format string, args ...interface{}) {
	if f.config.Debug {
		fmt.Fprintf(os.Stderr, "debug: "+format+"\n", args...)
	}
}

func (f *Filter) getFromCache(file string) (
	mtime time.Time, tr TimeRange, ok bool,
) {
	if f.db == nil {
		return
	}

	stat, err := os.Stat(file)
	if err != nil {
		f.error("stat file failed (file:'%s', reason:'%s')", file, err)
		return
	}
	mtime = stat.ModTime()

	data := cacheData{}
	if err = f.db.Get(file, &data); err != nil {
		if err != pudge.ErrKeyNotFound {
			f.error("restoring from database failed "+
				"(key:'%s', reason:'%s')", file, err)
		}
		return
	}

	if data.MTime != mtime {
		return
	}
	tr.From = data.From
	tr.To = data.To
	ok = true
	return
}

func (f *Filter) putToCache(file string, tr *TimeRange, mtime *time.Time) {
	if f.db == nil {
		return
	}

	data := cacheData{
		From:  tr.From,
		To:    tr.To,
		MTime: *mtime,
	}
	if err := f.db.Set(file, &data); err != nil {
		f.error("storing to database failed (key:'%s', reason:'%s')",
			file, err)
	}
}

func (f *Filter) extractTime(lines []string) time.Time {
	for i, line := range lines {
		for j, ex := range f.config.Extractors {
			if t, ok := ex.extract(line); ok {
				f.debug("  extractor %d hit on string %d -> %s", j, i, line)
				return t
			}
			f.debug("  extractor %d miss on string %d -> %s", j, i, line)
		}
	}
	return time.Time{}
}

func (f *Filter) extractTimeRange(file string) (tr TimeRange, ok bool) {
	var err error

	head, tail, err := readLines(file, f.config.Lines)
	if err != nil {
		f.error("extracting file time range failed "+
			"(file:'%s', reason:'%s')", file, err)
		return
	}
	tr.From = f.extractTime(head)
	tr.To = f.extractTime(tail)
	if tr.From.After(tr.To) {
		f.error("file time range is inversed (file:'%s')", file)
		return
	}
	return tr, true
}

func (f *Filter) intersects(tr *TimeRange) bool {
	if f.config.TimeRange.Intersects(tr) {
		f.debug("  intersects: yes")
		return true
	} else {
		f.debug("  intersects: no")
		return false
	}
}

func (f *Filter) Do(in Input, out io.Writer) error {
	var (
		err           error
		path, absPath string
		mtime         time.Time
		tr            TimeRange
		ok            bool
		wg            sync.WaitGroup
		pool          = make(chan struct{}, f.config.Concurrency)
	)
	defer wg.Wait()

	f.debug("time range: %s", &f.config.TimeRange)

	for in.Read(&path) {
		if absPath, err = filepath.Abs(path); err != nil {
			f.error("skipping path (path:'%s', reason:'%s')", path, err)
			continue
		}
		f.debug("path: '%s'", path)

		if mtime, tr, ok = f.getFromCache(absPath); ok {
			f.debug("  time range: %s (cache)", &tr)
			if f.intersects(&tr) {
				fmt.Println(path)
			}
			continue
		}

		pool <- struct{}{}
		wg.Add(1)
		go func(path, absPath string, mtime time.Time) {
			defer func() {
				<-pool
				wg.Done()
			}()

			if tr, ok := f.extractTimeRange(absPath); ok {
				f.debug("  time range: %s", &tr)
				f.putToCache(absPath, &tr, &mtime)
				if !f.intersects(&tr) {
					return
				}
			}
			fmt.Fprintln(out, path)
		}(path, absPath, mtime)
	}
	return nil
}
