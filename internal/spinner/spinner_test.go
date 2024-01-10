package spinner

import (
	"bytes"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/goleak"
	testclock "k8s.io/utils/clock/testing"
)

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}

func TestOutput(t *testing.T) {
	assert := assert.New(t)

	buf := bytes.Buffer{}
	counter := atomic.Uint64{}
	paintChan := make(chan struct{})
	callback := func(eventType callbackEventType) {
		counter.Add(1)
		if eventType == callbackEventTypeTick {
			paintChan <- struct{}{}
		}
	}
	clock := testclock.NewFakeClock(time.Now())
	s := &Spinner{
		prefix:       "test ",
		ticker:       clock.NewTicker(time.Minute),
		out:          &buf,
		callback:     callback,
		stopChan:     make(chan string, 1),
		stopDoneChan: make(chan struct{}, 1),
	}
	s.Start()
	for i := 0; i < 10; i++ {
		clock.Step(time.Minute)
		<-paintChan
	}
	s.Stop("done")

	assert.Equal(12, int(counter.Load())) // start + 10 dots + stop
	printed := buf.String()
	assert.Contains(printed, "test ..........") // start message + 10 dots
	assert.Contains(printed, "done")
}

func TestConcurrency(t *testing.T) {
	assert := assert.New(t)

	buf := bytes.Buffer{}
	s := New("test ", time.Millisecond, &buf)
	s.Start()

	wg := sync.WaitGroup{}

	start := func() {
		defer wg.Done()
		s.Start()
	}
	stop := func() {
		defer wg.Done()
		s.Stop("done")
	}
	wg.Add(10)
	go start()
	go start()
	go start()
	go start()
	go start()
	time.Sleep(10 * time.Millisecond)
	go stop()
	go stop()
	go stop()
	go stop()
	go stop()
	wg.Wait()

	assert.Contains(buf.String(), "test")
	assert.Contains(buf.String(), "done")
}
