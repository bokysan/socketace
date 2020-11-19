package streams

import (
	"fmt"
	"github.com/bokysan/socketace/v2/internal/util/buffers"
	"io"
)

// NamedStream implements the io.ReadWriteCloser interface as well as fmt.Stringer. It allows the caller to setup
// a name for the stream which will be returned when outputing the stream with `%v`.
// It also makes sure that `Close()` can be called safely multiple times. Calling `Close()` on a closed object
// will simply succeed without an error.
type NamedStream struct {
	ReadWriteCloserClosed
	name string
}

// NewNamedStream will, unsurprisingly, create a new NamedStream with a given name
func NewNamedStream(wrapped io.ReadWriteCloser, name string) *NamedStream {
	return &NamedStream{
		ReadWriteCloserClosed: NewSafeStream(wrapped),
		name:                  name,
	}
}

func (ns *NamedStream) WriteTo(w io.Writer) (n int64, err error) {
	if o, ok := ns.ReadWriteCloserClosed.(io.WriterTo); ok {
		return o.WriteTo(w)
	} else {
		return io.CopyBuffer(w, ns.ReadWriteCloserClosed, make([]byte, buffers.BufferSize))
	}
}

func (ns *NamedStream) ReadFrom(r io.Reader) (n int64, err error) {
	if o, ok := ns.ReadWriteCloserClosed.(io.ReaderFrom); ok {
		return o.ReadFrom(r)
	} else {
		return io.CopyBuffer(ns.ReadWriteCloserClosed, r, make([]byte, buffers.BufferSize))
	}
}

func (ns *NamedStream) String() string {
	result := ns.name

	var s io.ReadWriteCloser
	s = ns.ReadWriteCloserClosed
	for true {
		if t, ok := s.(UnwrappedReadWriteCloser); ok {
			u := t.Unwrap()
			if v, ok := u.(fmt.Stringer); ok {
				result += "->" + v.String()
				break
			}
			s = u
		} else {
			break
		}
	}

	return result
}

func (ns *NamedStream) Unwrap() io.ReadWriteCloser {
	return ns.ReadWriteCloserClosed
}
