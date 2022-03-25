package connect

import (
	"testing"
	"time"

	helpers "github.com/mariomac/pipes/pkg/test"
	"github.com/stretchr/testify/assert"
)

const timeout = 2 * time.Second

func TestJoiner(t *testing.T) {
	j := NewJoiner[int](20)
	finished := helpers.AsyncWait(1)

	go func() {
		j.AcquireSender() <- 1
		j.AcquireSender() <- 2
		j.AcquireSender() <- 3
		j.ReleaseSender()
		j.ReleaseSender()
		j.ReleaseSender()
	}()
	set := map[int]struct{}{}

	go func() {
		recv := j.Receiver()
		for i := range recv {
			set[i] = struct{}{}
		}
		finished.Done()
	}()
	finished.Wait(t, timeout)

	assert.Equal(t, map[int]struct{}{1: {}, 2: {}, 3: {}}, set)

	// check that all the channels have been closed
	assert.Panics(t, func() {
		ch := j.AcquireSender()
		close(ch)
	})
}

func TestForker(t *testing.T) {
	joiner1 := NewJoiner[int](20)
	joiner2 := NewJoiner[int](20)
	joiner3 := NewJoiner[int](20)

	f := Fork(&joiner1, &joiner2, &joiner3)
	sender := f.Sender()
	sender <- 1
	sender <- 2
	sender <- 3
	f.Close()

	finished := helpers.AsyncWait(3)
	var arr1, arr2, arr3 []int
	go func() {
		for i := range joiner1.Receiver() {
			arr1 = append(arr1, i)
		}
		finished.Done()
	}()
	go func() {
		for i := range joiner2.Receiver() {
			arr2 = append(arr2, i*2)
		}
		finished.Done()
	}()
	go func() {
		for i := range joiner3.Receiver() {
			arr3 = append(arr3, i+10)
		}
		finished.Done()
	}()

	finished.Wait(t, timeout)

	assert.Equal(t, []int{1, 2, 3}, arr1)
	assert.Equal(t, []int{2, 4, 6}, arr2)
	assert.Equal(t, []int{11, 12, 13}, arr3)

	// check that all the channels have been closed
	assert.Panics(t, func() {
		f.Sender() <- 1
	})
	assert.Panics(t, func() {
		r := joiner1.Receiver()
		close(r)
	})
	assert.Panics(t, func() {
		r := joiner2.Receiver()
		close(r)
	})
	assert.Panics(t, func() {
		r := joiner3.Receiver()
		close(r)
	})
}
