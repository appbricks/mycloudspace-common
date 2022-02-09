package monitors

import (
	"sync"
	"sync/atomic"
	"time"
)

type Counter struct {
	name    string
	attribs map[string]string

	cumalative,
	ignoreZeroSnapshots bool

	incBy,
	value,
	cumalativeValue int64

	counterLock sync.RWMutex
}

type counterSnapshot struct {
	Name      string `json:"name"`
	Timestamp int64  `json:"timestamp"`
	Value     int64  `json:"value"`

	Attribs map[string]string `json:"attribs,omitempty"`
}

// Returns a counter. If 'cumalative=true' then setting the counter
// value is assumed to be a cumalative value and will be determined
// using the last accumlated value.
func NewCounter(
	name string, 
	cumalative, ignoreZeroSnapshots bool,
) *Counter {

	return &Counter{
		name: name,

		cumalative:          cumalative,
		ignoreZeroSnapshots: ignoreZeroSnapshots,

		incBy: 1,
		value: 0,
		cumalativeValue: 0,
	}
}

func NewCounterWithAttribs(
	name string, 
	cumalative, ignoreZeroSnapshots bool, 
	attribs map[string]string,
) *Counter {

	return &Counter{
		name:    name,
		attribs: attribs,

		cumalative:          cumalative,
		ignoreZeroSnapshots: ignoreZeroSnapshots,

		incBy: 1,
		value: 0,
		cumalativeValue: 0,
	}
}

func (c *Counter) AddAttribute(name, value string) {
	c.attribs[name] = value
}

func (c *Counter) collect() *counterSnapshot {
	c.counterLock.Lock()
	defer c.counterLock.Unlock()

	if !c.ignoreZeroSnapshots || c.value != 0 {
		cs := &counterSnapshot{
			Name: c.name,
			Timestamp: time.Now().UnixNano() / int64(time.Millisecond),
			Value: c.value,
		}
		c.cumalativeValue += c.value
		c.value = 0
		return cs	
	}
	return nil
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
