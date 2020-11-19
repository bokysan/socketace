package streams

import (
	"fmt"
	"github.com/bokysan/socketace/v2/internal/util/buffers"
	"io"
	"net"
)

// NamedStream implements the io.ReadWriteCloser interface as well as fmt.Stringer. It allows the caller to setup
// a name for the stream which will be returned when outputing the stream with `%v`.
// It also makes sure that `Close()` can be called safely multiple times. Calling `Close()` on a closed object
// will simply succeed without an error.
type NamedConnection struct {
	Connection
	name string
}

// NewNamedStream will, unsurprisingly, create a new NamedStream with a given name
func NewNamedConnection(wrapped net.Conn, name string) *NamedConnection {
	return &NamedConnection{
		Connection: NewSafeConnection(wrapped),
		name:       name,
	}
}

func (ns *NamedConnection) WriteTo(w io.Writer) (n int64, err error) {
	if o, ok := ns.Connection.(io.WriterTo); ok {
		return o.WriteTo(w)
	} else {
		return io.CopyBuffer(w, ns.Connection, make([]byte, buffers.BufferSize))
	}
}

func (ns *NamedConnection) ReadFrom(r io.Reader) (n int64, err error) {
	if o, ok := ns.Connection.(io.ReaderFrom); ok {
		return o.ReadFrom(r)
	} else {
		return io.CopyBuffer(ns.Connection, r, make([]byte, buffers.BufferSize))
	}
}

func (ns *NamedConnection) String() string {
	result := ns.name

	var s net.Conn
	s = ns.Connection
	for true {
		if t, ok := s.(UnwrappedConnection); ok {
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

func (ns *NamedConnection) Unwrap() net.Conn {
	return ns.Connection
}
