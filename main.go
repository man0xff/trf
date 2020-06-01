package main

import (
	"os"

	"gopkg.in/alecthomas/kingpin.v2"

	"trf/filter"
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
	noCache    *bool
	lines      *int
	file       *string
}

func newApp() *app {
	a := &app{}
	kingpin.Flag("extractor", "how to extract time information from string").
		Short('e').Required().SetValue(&a.extractors)
	a.file = kingpin.Flag("file", "use file as input ('-' for stdin)").
		Short('f').String()
	a.lines = kingpin.Flag("lines", "number of lines to analyze from both sides").
		Short('n').Default("3").Int()
	a.debug = kingpin.Flag("debug", "enable debugging messages").Bool()
	a.noCache = kingpin.Flag("no-cache", "disable caching").Bool()
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

	f := filter.New(&filter.Config{
		TimeRange:  a.timeRange.TimeRange,
		CacheFile:  "/tmp/trf.cache1",
		Extractors: ([]*filter.Extractor)(a.extractors),
		Debug:      *a.debug,
		NoCache:    *a.noCache,
		Lines:      *a.lines,
	})
	f.Do(input, os.Stdout)
	f.Close()
}

func main() {
	app := newApp()
	app.run()
}
