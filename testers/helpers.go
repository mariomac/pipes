package testers

import (
	"sync"
	"testing"
	"time"
)

// AsyncWaiter merges a waitgroup and a channel to provide syntactic sugar in asnynchronous
// tests, making them failing if the waitGroup does not reach the 0 status after a given timeout.
type AsyncWaiter sync.WaitGroup

func AsyncWait(groupLen int) *AsyncWaiter {
	a := &AsyncWaiter{}
	(*sync.WaitGroup)(a).Add(groupLen)
	return a
}

func (a *AsyncWaiter) Done() {
	(*sync.WaitGroup)(a).Done()
}

func (a *AsyncWaiter) Wait(t *testing.T, timeout time.Duration) {
	t.Helper()
	done := make(chan struct{})
	go func() {
		(*sync.WaitGroup)(a).Wait()
		close(done)
	}()
	select {
	case <-done:
		return
	case <-time.After(timeout):
		t.Fatal("timeout waiting for test to be completed")
	}
}

// ReadChannel tries to read a message from a channel and returns it. If there isn't any
// message after the given timeout, it fails the provided test
func ReadChannel[T any](t *testing.T, inCh <-chan T, timeout time.Duration) T {
	t.Helper()
	var item T
	select {
	case item = <-inCh:
		return item
	case <-time.After(timeout):
		t.Fatalf("timeout (%s) while waiting for event in input channel", timeout)
	}
	return item
}
