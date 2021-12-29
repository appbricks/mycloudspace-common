package monitors

import (
	"sync"
	"sync/atomic"
	"time"
)

type Counter struct {
	name string

	cumalative bool

	incBy,
	value,
	cumalativeValue int64

	counterLock sync.RWMutex
}

type counterSnapshot struct {
	Name      string `json:"name"`
	Timestamp int64  `json:"timestamp"`
	Value     int64  `json:"value"`
}

// Returns a counter. If 'cumalative=true' then setting the counter
// value is assumed to be a cumalative value and will be determined
// using the last accumlated value.
func NewCounter(name string, cumalative bool) *Counter {
	return &Counter{
		name: name,

		cumalative: cumalative,

		incBy: 1,
		value: 0,
		cumalativeValue: 0,
	}
}

func (c *Counter) collect() *counterSnapshot {
	c.counterLock.Lock()
	defer c.counterLock.Unlock()

	cs := &counterSnapshot{
		Name: c.name,
		Timestamp: time.Now().UnixNano() / int64(time.Millisecond),
		Value: c.value,
	}
	c.cumalativeValue += c.value
	c.value = 0
	return cs
}

func (c *Counter) Name() string {
	return c.name
}

func (c *Counter) Get() int64 {
	c.counterLock.RLock()
	defer c.counterLock.RUnlock()

	return atomic.AddInt64(&c.value, c.cumalativeValue)
}

func (c *Counter) SetInc(incValue int64) {
	c.counterLock.Lock()
	defer c.counterLock.Unlock()

	c.incBy = incValue
}

func (c *Counter) Set(value int64) {
	c.counterLock.RLock()
	defer c.counterLock.RUnlock()

	if c.cumalative {
		atomic.StoreInt64(&c.value, value - c.cumalativeValue)
	} else {
		atomic.StoreInt64(&c.value, value)
	}
}

func (c *Counter) Inc() {
	c.counterLock.RLock()
	defer c.counterLock.RUnlock()

	atomic.AddInt64(&c.value, c.incBy)
}

func (c *Counter) Add(value int64) {
	c.counterLock.RLock()
	defer c.counterLock.RUnlock()

	atomic.AddInt64(&c.value, value)
}
