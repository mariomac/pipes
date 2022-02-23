package node

import (
	"sync"

	"github.com/mariomac/go-pipes/pkg/internal/refl"
)

type Joiner struct {
	mutex        sync.Mutex
	channelType  refl.ChannelType
	totalSenders int
	bufLen       int
	// 0 is the main channel
	main     refl.Channel
	channels []refl.Channel
}

func NewJoiner(ct refl.ChannelType, bufferLength int) Joiner {
	return Joiner{
		channelType: ct,
		bufLen:      bufferLength,
		main:        ct.Instantiate(bufferLength),
	}
}

func (j *Joiner) Receiver() refl.Channel {
	return j.main
}

type Releaser func()

func (j *Joiner) AcquireSender() (refl.Channel, Releaser) {
	j.mutex.Lock()
	defer j.mutex.Unlock()
	if j.totalSenders == 0 {
		j.totalSenders = 1
		return j.main, j.release
	}
	ch := j.channelType.Instantiate(j.bufLen)
	j.totalSenders++
	j.channels = append(j.channels, ch)
	// connect the new channel to the main channel in a new goroutine
	go func() {
		for in, ok := ch.Recv(); ok; in, ok = ch.Recv() {
			j.main.Send(in)
		}
		j.release()
	}()
	return ch, func() { ch.Close() }
}

func (j *Joiner) release() {
	j.mutex.Lock()
	defer j.mutex.Unlock()
	j.totalSenders--
	// if no senders, we close the main channel
	if j.totalSenders == 0 {
		j.main.Close()
		j.channels = nil
	}
}

type Forker struct {
	sendCh         refl.Channel
	releaseChannel Releaser
}

func Fork(joiners ...Joiner) Forker {
	if len(joiners) == 0 {
		panic("can't fork 0 joiners")
	}
	// if there is only one joiner, we directly send, without intermediation
	if len(joiners) == 1 {
		ch, release := joiners[0].AcquireSender()
		return Forker{
			sendCh:         ch,
			releaseChannel: release,
		}
	}
	// assuming all the channels are from the same type (previously verified)
	chType := joiners[0].channelType
	sendCh := chType.Instantiate(joiners[0].bufLen)

	forwarders := make([]refl.Channel, len(joiners))
	releasers := make([]Releaser, len(joiners))
	for i := 0; i < len(joiners); i++ {
		forwarders[i], releasers[i] = joiners[i].AcquireSender()
	}
	go func() {
		for in, ok := sendCh.Recv(); ok; in, ok = sendCh.Recv() {
			for i := 0; i < len(joiners); i++ {
				forwarders[i].Send(in)
			}
		}
		for i := 0; i < len(joiners); i++ {
			releasers[i]()
		}
	}()
	return Forker{
		sendCh:         sendCh,
		releaseChannel: sendCh.Close,
	}
}

func (f *Forker) Sender() refl.Channel {
	return f.sendCh
}

func (f *Forker) Close() {
	f.releaseChannel()
}
