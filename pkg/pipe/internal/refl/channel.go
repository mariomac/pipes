package refl

import "reflect"

// Channel wraps a channel for its usage with refl.Function objects
type Channel reflect.Value

func makeChannel(inType reflect.Type, bufLen int) reflect.Value {
	chanType := reflect.ChanOf(reflect.BothDir, inType)
	return reflect.MakeChan(chanType, bufLen)
}

// NilChannel returns a pointer to a nil Channel
func NilChannel() *Channel {
	var nv *interface{}
	vo := reflect.ValueOf(nv)
	ch := Channel(vo)
	return &ch
}

func (ch *Channel) IsNil() bool {
	return (*reflect.Value)(ch).IsNil()
}

// Fork returns two new channels that will forward the contents of the receiver channel.
// It spawns a new goroutine and, when the receiver channel is closed, both returned channels
// are also closed
func (ch *Channel) Fork() (Channel, Channel) {
	inChan := (*reflect.Value)(ch)
	chanType := reflect.ChanOf(reflect.BothDir, inChan.Type().Elem())
	out1 := reflect.MakeChan(chanType, inChan.Len())
	out2 := reflect.MakeChan(chanType, inChan.Len())
	go func() {
		for in, ok := inChan.Recv(); ok; in, ok = inChan.Recv() {
			out1.Send(in)
			out2.Send(in)
		}
		out1.Close()
		out2.Close()
	}()
	return Channel(out1), Channel(out2)
}
