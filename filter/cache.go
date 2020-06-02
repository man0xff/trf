package filter

import (
	"math/rand"
	"os"
	"sync"
	"time"

	"github.com/recoilme/pudge"
)

type cache struct {
	f       *Filter
	db      *pudge.Db
	wg      sync.WaitGroup
	done    chan struct{}
	noRead  bool
	noWrite bool
}

type cacheData struct {
	From  time.Time
	To    time.Time
	MTime time.Time
}

func newCache(f *Filter, file string) *cache {
	c := &cache{
		f:    f,
		done: make(chan struct{}),
	}

	if f.config.NoCacheRead && f.config.NoCacheWrite {
		return c
	}
	db, err := pudge.Open(file, nil)
	if err != nil {
		f.error("database opening failed (file:'%s', reason:'%s')", file, err)
		return c
	}
	c.db = db
	c.noRead = f.config.NoCacheRead
	c.noWrite = f.config.NoCacheWrite
	c.wg.Add(1)
	go c.clean()
	return c
}

func (c *cache) read(file string, mTime *time.Time) (TimeRange, bool) {
	var tr TimeRange

	if c.noRead {
		return tr, false
	}

	data := cacheData{}
	if err := c.db.Get(file, &data); err != nil {
		if err != pudge.ErrKeyNotFound {
			c.f.error("restoring from database failed (key:'%s', reason:'%s')",
				file, err)
		}
		return tr, false
	}
	if data.MTime != *mTime {
		return tr, false
	}
	tr.From = data.From
	tr.To = data.To
	return tr, true
}

func (c *cache) write(file string, tr *TimeRange, mTime *time.Time) {
	if c.noWrite {
		return
	}

	err := c.db.Set(file, &cacheData{From: tr.From, To: tr.To, MTime: *mTime})
	if err != nil {
		c.f.error("storing to database failed (key:'%s', reason:'%s')", file, err)
	}
}

func (c *cache) clean() {
	defer c.wg.Done()

	const limit = 10
	var (
		data   cacheData
		offset int
	)

	rand.Seed(time.Now().UTC().UnixNano())
	for {
		select {
		case <-c.done:
			return
		default:
		}

		if n, err := c.db.Count(); err != nil {
			c.f.warning("obtaining number of records in database failed "+
				"(reason:'%s')", err)
			return
		} else {
			offset = rand.Int() % n
		}

		files, err := c.db.Keys(nil, limit, offset, true)
		if err != nil {
			c.f.warning("obtaining keys from database failed (reason:'%s')", err)
			return
		} else if len(files) == 0 {
			break
		}
		offset += limit

		for _, file := range files {
			if err := c.db.Get(file, &data); err != nil {
				if err != pudge.ErrKeyNotFound {
					c.f.warning("reading data from database failed "+
						"(key:'%s', reason:'%s')", file, err)
				}
				continue
			}
			stat, err := os.Stat(string(file))
			if err == os.ErrNotExist || stat.ModTime() != data.MTime {
				c.db.Delete(file)
			}
		}
	}
}

func (c *cache) close() {
	close(c.done)
	c.wg.Wait()
	if c.db != nil {
		c.db.Close()
	}
}
