// Package safetest holds race-safe values for tests to share with the
// goroutines of the code under test. It imports nothing from the app, so it
// can be used by tests of any package.
package safetest

import (
	"log"
	"slices"
	"strings"
	"sync"
)

// LogCapture is an io.Writer that collects what is logged from any goroutine.
type LogCapture struct {
	mu      sync.Mutex
	builder strings.Builder
}

func (inst *LogCapture) Write(p []byte) (int, error) {
	inst.mu.Lock()
	defer inst.mu.Unlock()
	return inst.builder.Write(p)
}

// String returns everything logged so far.
func (inst *LogCapture) String() string {
	inst.mu.Lock()
	defer inst.mu.Unlock()
	return inst.builder.String()
}

// Logger returns a logger without a prefix or flags writing to the capture.
func (inst *LogCapture) Logger() *log.Logger {
	return log.New(inst, "", 0)
}

// Recorder collects values recorded from any goroutine, such as the arguments
// a callback was called with.
type Recorder[T any] struct {
	mu     sync.Mutex
	values []T
}

func (inst *Recorder[T]) Record(v T) {
	inst.mu.Lock()
	defer inst.mu.Unlock()
	inst.values = append(inst.values, v)
}

// Calls returns a copy of what was recorded, in order.
func (inst *Recorder[T]) Calls() []T {
	inst.mu.Lock()
	defer inst.mu.Unlock()
	return slices.Clone(inst.values)
}

// Count returns how many values were recorded.
func (inst *Recorder[T]) Count() int {
	inst.mu.Lock()
	defer inst.mu.Unlock()
	return len(inst.values)
}

// Value is a single value that can be set and read from any goroutine.
type Value[T any] struct {
	mu sync.Mutex
	v  T
}

func (inst *Value[T]) Set(v T) {
	inst.mu.Lock()
	defer inst.mu.Unlock()
	inst.v = v
}

func (inst *Value[T]) Get() T {
	inst.mu.Lock()
	defer inst.mu.Unlock()
	return inst.v
}
