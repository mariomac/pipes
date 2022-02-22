package refl

import "reflect"

// Channel wraps a channel for its usage with refl.Function objects
type Channel reflect.Value

func (ch *Channel) IsNil() bool {
	return (*reflect.Value)(ch).IsNil()
}

func makeChannel(inType reflect.Type, bufLen int) reflect.Value {
	chanType := reflect.ChanOf(reflect.BothDir, inType)
	return reflect.MakeChan(chanType, bufLen)
}

// Nil returns a pointer to a nil Channel
func Nil() *Channel {
	var nv *interface{}
	vo := reflect.ValueOf(nv)
	ch := Channel(vo)
	return &ch
}

// some functions for syntactic sugar
func typeOf(fn *Function) reflect.Type {
	return (*reflect.Value)(fn).Type()
}

func valueOf(fn *Function) *reflect.Value {
	return (*reflect.Value)(fn)
}
