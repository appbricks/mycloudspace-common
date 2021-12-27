package monitors

import (
	"sync"
	"time"
)

type Counter struct {
	name string

	cumalative bool

	incBy,
	value,
	cumalativeValue int64

	counterLock sync.Mutex
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

func (c *Counter) SetInc(incValue int64) {
	c.counterLock.Lock()
	defer c.counterLock.Unlock()
	
	c.incBy = incValue
}

func (c *Counter) Set(value int64) {
	c.counterLock.Lock()
	defer c.counterLock.Unlock()
	
	if c.cumalative {
		c.value = value - c.cumalativeValue
	} else {
		c.value = value
	}
}

func (c *Counter) Inc() {
	c.counterLock.Lock()
	defer c.counterLock.Unlock()

	c.value += c.incBy
}

func (c *Counter) Add(value int64) {
	c.counterLock.Lock()
	defer c.counterLock.Unlock()
	
	c.value += value
}
