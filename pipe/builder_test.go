package pipe_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mariomac/pipes/pipe"
)

type StartError struct{}

func (StartError) Error() string { return "" }

type MidError struct{}

func (MidError) Error() string { return "" }

type FinalError struct{}

func (FinalError) Error() string { return "" }

func TestError_StartNode(t *testing.T) {
	b := pipe.NewBuilder(&smfPipe{})
	pipe.AddStartProvider(b, start, func() (pipe.StartFunc[int], error) {
		return nil, StartError{}
	})
	pipe.AddMiddle(b, mid, func(in <-chan int, out chan<- int) {})
	pipe.AddFinal(b, final, func(in <-chan int) {})

	_, err := b.Build()
	require.Error(t, err)
	assert.ErrorIs(t, err, StartError{})
}

func TestError_MiddleNode(t *testing.T) {
	b := pipe.NewBuilder(&smfPipe{})
	pipe.AddStart(b, start, func(out chan<- int) {})
	pipe.AddMiddleProvider(b, mid, func() (pipe.MiddleFunc[int, int], error) {
		return nil, MidError{}
	})
	pipe.AddFinal(b, final, func(in <-chan int) {})

	_, err := b.Build()
	require.Error(t, err)
	assert.ErrorIs(t, err, MidError{})
}

func TestError_FinalNode(t *testing.T) {
	b := pipe.NewBuilder(&smfPipe{})
	pipe.AddStart(b, start, func(out chan<- int) {})
	pipe.AddMiddle(b, mid, func(in <-chan int, out chan<- int) {})
	pipe.AddFinalProvider(b, final, func() (pipe.FinalFunc[int], error) {
		return nil, FinalError{}
	})

	_, err := b.Build()
	require.Error(t, err)
	assert.ErrorIs(t, err, FinalError{})
}
