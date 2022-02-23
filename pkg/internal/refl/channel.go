package refl

import "reflect"

// Channel wraps a channel for its usage with refl.Function objects
type Channel struct {
	reflect.Value
}

// ChannelType wraps a channel type
type ChannelType struct {
	inner reflect.Type
}

func makeChannel(inType reflect.Type, bufLen int) reflect.Value {
	chanType := reflect.ChanOf(reflect.BothDir, inType)
	return reflect.MakeChan(chanType, bufLen)
}

// NilChannel returns a pointer to a nil Channel
func NilChannel() *Channel {
	var nv *interface{}
	vo := reflect.ValueOf(nv)
	ch := Channel{vo}
	return &ch
}

// Fork returns two new channels that will forward the contents of the receiver channel.
// It spawns a new goroutine and, when the receiver channel is closed, both returned channels
// are also closed
func (ch *Channel) Fork() (Channel, Channel) {
	chanType := reflect.ChanOf(reflect.BothDir, ch.Type().Elem())
	out1 := reflect.MakeChan(chanType, ch.Len())
	out2 := reflect.MakeChan(chanType, ch.Len())
	go func() {
		for in, ok := ch.Recv(); ok; in, ok = ch.Recv() {
			out1.Send(in)
			out2.Send(in)
		}
		out1.Close()
		out2.Close()
	}()
	return Channel{out1}, Channel{out2}
}

func (ch ChannelType) CanSend() bool {
	return ch.inner.ChanDir()&reflect.SendDir != 0
}

func (ch *ChannelType) CanReceive() bool {
	return ch.inner.ChanDir()&reflect.RecvDir != 0
}

func (ch *ChannelType) String() string {
	return ch.inner.String()
}

func (ch *ChannelType) Instantiate(bufLen int) Channel {
	return Channel{makeChannel(ch.inner.Elem(), bufLen)}
}
