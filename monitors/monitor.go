package monitors

import (
	"context"
	"sync"
	"time"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/cloudevents/sdk-go/v2/event"
	"github.com/google/uuid"

	"github.com/mevansam/goutils/logger"
	"github.com/mevansam/goutils/utils"
)

const networkMetricEventType = `io.appbricks.mycs.network.metric` 
const collectionInterval = 1000 // 1 second in ms

type Sender interface {
	PostMeasurementEvents(events []*cloudevents.Event) error
}

type monitor struct {
	name     string
	counters []*Counter

	lock *sync.Mutex
}

type monitorService struct {
	ctx    context.Context
	cancel context.CancelFunc

	sender Sender
	sendWG sync.WaitGroup

	sendInterval, 
	sendCountdown int

	monitors []*monitor
	lock     sync.Mutex

	eventPayloads []*eventPayload

	authExecTimer *utils.ExecTimer
}

type eventPayload struct {
	Monitors []*monitorSnapshot `json:"monitors"`
}
type monitorSnapshot struct {
	Name     string            `json:"name"`
	Counters []*counterSnapshot `json:"counters"`
}

// Creates a new monitor services with a 'sender' that
// will post monitor events to an upstream service
// every 'sendInterval' seconds.
func NewMonitorService(sender Sender, sendInterval int) *monitorService {

	ctx, cancel := context.WithCancel(context.Background())

	return &monitorService{
		ctx:    ctx,
		cancel: cancel,

		sender:        sender,
		sendInterval:  sendInterval-1,
		sendCountdown: sendInterval-1,

		monitors: []*monitor{},

		// payload for each snapshot collected
		eventPayloads: make([]*eventPayload, 0, sendInterval),
	}
}

func (ms *monitorService) NewMonitor(name string) *monitor {
	ms.lock.Lock()
	defer ms.lock.Unlock()

	monitor := &monitor{
		name:     name,
		counters: []*Counter{},

		lock: &ms.lock,
	}
	ms.monitors = append(ms.monitors, monitor)

	return monitor
}

func (ms *monitorService) Start() error {
	ms.authExecTimer = utils.NewExecTimer(ms.ctx, ms.collect, false)
	return ms.authExecTimer.Start(collectionInterval)
}

func (ms *monitorService) collect() (time.Duration, error) {
	ms.lock.Lock()
	defer ms.lock.Unlock()

	ms.collectEvents()
	if ms.sendCountdown == 0 {
		ms.postEvents()
		ms.sendCountdown = ms.sendInterval
	} else {
		ms.sendCountdown--
	}

	// metrics collected every second
	return collectionInterval, nil
}

func (ms *monitorService) collectEvents() {

	eventPayload := eventPayload{}
	ms.eventPayloads = append(ms.eventPayloads, &eventPayload)
	for _, m := range ms.monitors {
		monitorSnapshot := monitorSnapshot{
			Name: m.name,
		}
		eventPayload.Monitors = append(eventPayload.Monitors, &monitorSnapshot)

		for _, c := range m.counters {
			counterSnapshot := c.collect()
			monitorSnapshot.Counters = append(monitorSnapshot.Counters, counterSnapshot)
		}
	}
}

func (ms *monitorService) postEvents() {
	numEvents := len(ms.eventPayloads)
	logger.DebugMessage("monitorService.collect(): Posting %d cloud events", numEvents)

	// make a copy of all the payloads that will 
	// be pushed to the cloud asynchronously
	eventPayloads := make([]*eventPayload, numEvents)
	copy(eventPayloads, ms.eventPayloads)
	ms.eventPayloads = ms.eventPayloads[:0]

	ms.sendWG.Add(1)
	go func() {
		defer ms.sendWG.Done()

		var (
			err error
		)
		
		events := make([]*event.Event, 0, numEvents)
		for _, data := range eventPayloads {
			eventUUID := uuid.NewString()
			
			event := cloudevents.NewEvent()
			event.SetID(eventUUID)
			event.SetType(networkMetricEventType)
			event.SetSubject("MyCS Application Monitors")
			event.SetDataContentType("application/json")
			event.SetTime(time.Now())
			if err = event.SetData(cloudevents.ApplicationJSON, data); err != nil {
				logger.ErrorMessage(
					"monitorService.collect(): Unable to add monitor payload to cloud event instance with id \"%s\": %s", 
					eventUUID, err.Error(),
				)
			}
			events = append(events, &event)
		}
		if err = ms.sender.PostMeasurementEvents(events); err != nil {
			logger.ErrorMessage("monitorService.collect(): Unable to post measurement events: %s", err.Error())
		}
	}()	
}

func (ms *monitorService) Stop() {

	if ms.authExecTimer != nil {
		if err := ms.authExecTimer.Stop(); err != nil {
			logger.DebugMessage(
				"monitorService.Stop(): Auth execution timer stopped with err: %s", 
				err.Error())	
		}
	}
	ms.sendWG.Wait()

	// ensure all data that is waiting to 
	// be collected or posted are processed
	ms.lock.Lock()
	defer ms.lock.Unlock()

	ms.collectEvents()
	ms.postEvents()
	ms.sendWG.Wait()
}

func (m *monitor) AddCounter(counter *Counter) {
	m.lock.Lock()
	defer m.lock.Unlock()

	m.counters = append(m.counters, counter)
}