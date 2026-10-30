package safetest

import (
	"sync"
	"testing"
)

func TestSafetest__SharedAcrossGoroutines(t *testing.T) {
	var logs LogCapture
	var calls Recorder[int]
	var value Value[int]
	var logger = logs.Logger()

	var wg sync.WaitGroup
	for i := range 20 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			logger.Print("x")
			calls.Record(i)
			value.Set(i)
			_ = logs.String()
			_ = calls.Calls()
			_ = value.Get()
		}()
	}
	wg.Wait()

	if calls.Count() != 20 || len(calls.Calls()) != 20 {
		t.Errorf("recorded %d, want 20", calls.Count())
	}
	if got := logs.String(); len(got) != 40 {
		t.Errorf("logged %q", got)
	}
}
