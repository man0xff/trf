package main

import (
	"os"

	"gopkg.in/alecthomas/kingpin.v2"

	"trf/filter"
)

const (
	maxJobsN = 1000
)

var (
	defaultCacheFile  = home(".local/trf/cache")
	defaultConfigFile = home(".config/trf/config.yaml")
)

func home(s string) string {
	h, err := os.UserHomeDir()
	kingpin.FatalIfError(err, "")
	return h + "/" + s
}

type app struct {
	extractors extractors
	timeRange  cliTimeRange
	files      *[]string
	debug      *bool
	cache      *string
	lines      *int
	file       *string
	jobs       *int
}

func newApp() *app {
	a := &app{}
	kingpin.HelpFlag.Short('h')
	kingpin.Flag("extractor", "how to extract time information from string").
		Short('e').Required().SetValue(&a.extractors)
	a.file = kingpin.Flag("file", "use file as input ('-' for stdin)").
		Short('f').String()
	a.jobs = kingpin.Flag("jobs", "maximum number of parallel jobs").
		Short('j').Default("1").Int()
	a.lines = kingpin.Flag("lines", "number of lines to analyze from both sides").
		Short('n').Default("3").Int()
	a.debug = kingpin.Flag("debug", "enable debugging messages").Bool()
	a.cache = kingpin.Flag("cache", "cache mode (one from ',r,w,rw')").Default("rw").
		Enum("", "r", "w", "rw")
	kingpin.Arg("time range", "interesting time range").
		Required().SetValue(&a.timeRange)
	a.files = kingpin.Arg("files", "files to go through").Strings()
	return a
}

func (a *app) run() {
	var (
		err   error
		input filter.Input
		file  *os.File
	)

	kingpin.Parse()

	if *a.jobs < 1 || *a.jobs > maxJobsN {
		*a.jobs = maxJobsN
	}

	switch *a.file {
	case "-":
		input = a.newFileInput(os.Stdin)
	case "":
		input = a.newStringsInput(*a.files)
	default:
		if file, err = os.Open(*a.file); err != nil {
			kingpin.FatalIfError(err, "")
		}
		input = a.newFileInput(file)
	}

	var noCacheRead, noCacheWrite bool
	switch *a.cache {
	case "":
		noCacheRead = true
		noCacheWrite = true
	case "r":
		noCacheWrite = true
	case "w":
		noCacheRead = true
	case "rw":
	}

	f := filter.New(&filter.Config{
		TimeRange:    a.timeRange.TimeRange,
		CacheFile:    "/tmp/trf.cache1",
		Extractors:   ([]*filter.Extractor)(a.extractors),
		NoCacheRead:  noCacheRead,
		NoCacheWrite: noCacheWrite,
		Debug:        *a.debug,
		Lines:        *a.lines,
		Concurrency:  *a.jobs,
	})
	f.Do(input, os.Stdout)
	f.Close()
}

func main() {
	app := newApp()
	app.run()
}
