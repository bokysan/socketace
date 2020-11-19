package streams

import (
	"fmt"
	"github.com/bokysan/socketace/v2/internal/util/buffers"
	"io"
)

// NamedWriter implements the io.WriteCloser interface as well as fmt.Stringer. It allows the caller to setup
// a name for the stream which will be returned when outputing the stream with `%v`.
// It also makes sure that `Close()` can be called safely multiple times. Calling `Close()` on a closed object
// will simply succeed without an error.
type NamedWriter struct {
	WriteCloserClosed
	name   string
	closed bool
}

// NewNamedStream will, unsurprisingly, create a new NamedStream with a given name
func NewNamedWriter(wrapped io.WriteCloser, name string) *NamedWriter {
	return &NamedWriter{
		WriteCloserClosed: NewSafeWriter(wrapped),
		name:              name,
	}
}

func (ns *NamedWriter) ReadFrom(r io.Reader) (n int64, err error) {
	if o, ok := ns.WriteCloserClosed.(io.ReaderFrom); ok {
		return o.ReadFrom(r)
	} else {
		return io.CopyBuffer(ns.WriteCloserClosed, r, make([]byte, buffers.BufferSize))
	}
}

func (ns *NamedWriter) String() string {
	result := ns.name

	var s io.WriteCloser
	s = ns.WriteCloserClosed
	for true {
		if t, ok := s.(UnwrappedWriteCloser); ok {
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

func (ns *NamedWriter) Unwrap() io.WriteCloser {
	return ns.WriteCloserClosed
}
