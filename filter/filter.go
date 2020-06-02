package filter

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type Config struct {
	Concurrency  int
	TimeRange    TimeRange
	CacheFile    string
	Extractors   []*Extractor
	Debug        bool
	NoCacheRead  bool
	NoCacheWrite bool
	Lines        int
}

type Filter struct {
	config Config
	cache  *cache
}

type Input interface {
	Read(*string) bool
}

func New(config *Config) *Filter {
	f := &Filter{}
	f.config = *config
	if f.config.Concurrency <= 0 {
		f.config.Concurrency = 1
	}
	f.cache = newCache(f, f.config.CacheFile)
	return f
}

func (f *Filter) Close() {
	f.cache.close()
}

func (f *Filter) error(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "error: "+format+"\n", args...)
}

func (f *Filter) warning(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "warning: "+format+"\n", args...)
}

func (f *Filter) debug(format string, args ...interface{}) {
	if f.config.Debug {
		fmt.Fprintf(os.Stderr, "debug: "+format+"\n", args...)
	}
}

func (f *Filter) getMTime(file string) (time.Time, bool) {
	stat, err := os.Stat(file)
	if err != nil {
		f.error("stat file failed (file:'%s', reason:'%s')", file, err)
		return time.Time{}, false
	}
	return stat.ModTime(), true
}

func (f *Filter) extractTime(lines []string, file string, loc string) time.Time {
	for i, line := range lines {
		for j, ex := range f.config.Extractors {
			if t, ok := ex.extract(line); ok {
				f.debug("  extractor %d hit on string %d -> %s", j, i, line)
				return t
			}
			f.debug("  extractor %d miss on string %d -> %s", j, i, line)
		}
	}
	f.warning("time information extracting failed "+
		"(file:'%s', loc:'%s', lines:%d)", file, loc, len(lines))
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
	tr.From = f.extractTime(head, file, "head")
	tr.To = f.extractTime(tail, file, "tail")
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

		mTime, _ := f.getMTime(absPath)
		if tr, ok = f.cache.read(absPath, &mTime); ok {
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
				f.cache.write(absPath, &tr, &mtime)
				if !f.intersects(&tr) {
					return
				}
			}
			fmt.Fprintln(out, path)
		}(path, absPath, mTime)
	}
	return nil
}
