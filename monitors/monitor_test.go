package monitors_test

import (
	"encoding/json"
	"math/rand"
	"sync"
	"time"

	"github.com/appbricks/mycloudspace-common/monitors"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/mevansam/goutils/utils"
	"go.uber.org/atomic"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Monitors", func() {

	var (
		err error
	)

	It("collects from an incrementing and decrementing monitor counter", func() {

		s := &testSender{}
		msvc := monitors.NewMonitorService(s, 5)

		monitor := msvc.NewMonitor("testMonitor")
		Expect(monitor).NotTo(BeNil())
		counter := monitors.NewCounter("testCounter", false)
		Expect(counter).NotTo(BeNil())
		monitor.AddCounter(counter)
		
		err = msvc.Start()
		Expect(err).NotTo(HaveOccurred())

		wg := sync.WaitGroup{}
		wg.Add(3)

		cumalativeValue := 0
		incAt := []time.Duration{100, 200, 500}
		for i := 0; i < 3; i++ {
			incInt := incAt[i]

			go func() {
				t := time.Duration(0)
				for t < 15500 {
					timer := time.NewTicker(incInt * time.Millisecond)
					<-timer.C
					counter.Inc()
					cumalativeValue++
					t += incInt
				}
				wg.Done()
			}()
		}
		wg.Wait()

		msvc.Stop()
		Expect(s.numEvents).To(Equal(16))
		Expect(s.cumalativeValue).To(Equal(cumalativeValue))
	})

	It("collects from an incrementing and decrementing monitor counter", func() {

		s := &testSender{}
		msvc := monitors.NewMonitorService(s, 5)

		monitor := msvc.NewMonitor("testMonitor")
		Expect(monitor).NotTo(BeNil())
		counter := monitors.NewCounter("testCounter", true)
		Expect(counter).NotTo(BeNil())
		monitor.AddCounter(counter)
		
		err = msvc.Start()
		Expect(err).NotTo(HaveOccurred())

		wg := sync.WaitGroup{}
		wg.Add(3)

		var cumalativeValue atomic.Int64
		incAt := []time.Duration{100, 200, 500}
		for i := 0; i < 3; i++ {
			incInt := incAt[i]

			go func() {
				t := time.Duration(0)
				for t < 15500 {
					timer := time.NewTicker(incInt * time.Millisecond)
					<-timer.C
					counter.Set(cumalativeValue.Add(rand.Int63n(4)+1))
					t += incInt
				}
				wg.Done()
			}()
		}
		wg.Wait()

		msvc.Stop()
		Expect(s.numEvents).To(Equal(16))
		Expect(s.cumalativeValue).To(Equal(int(cumalativeValue.Load())))
	})
})

type testSender struct {
	events []*cloudevents.Event	

	numEvents,
	cumalativeValue int
}
func (s *testSender) PostMeasurementEvents(events []*cloudevents.Event) error {
	defer GinkgoRecover()

	var (
		err error
	)

	for _, e := range events {
		Expect(e.Context.GetID()).To(MatchRegexp("^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$"))
		Expect(e.Context.GetType()).To(Equal("io.appbricks.mycs.network.metric"))
		Expect(e.Context.GetSubject()).To(Equal("Application Monitor Snapshot"))
		Expect(e.Context.GetDataContentType()).To(Equal("application/json"))

		data := make(map[string]interface{})
		err = json.Unmarshal(e.Data(), &data)
		Expect(err).NotTo(HaveOccurred())
		
		s.numEvents++
		s.cumalativeValue += int((utils.MustGetValueAtPath("monitors/0/counters/0/value", data)).(float64))		
		s.events = append(s.events, e)
	}

	return nil
}
