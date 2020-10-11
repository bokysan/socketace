package streams

import (
	"github.com/bokysan/socketace/v2/internal/util/buffers"
	"io"
	"net"
)

// SafeStream makes sure that `Close()` can be called safely multiple times. Calling `Close()` on a closed object
// will simply succeed without an error.
type SafeConnection struct {
	net.Conn
	closed bool
}

// NewSafeStream will, create a new SafeStream with a given name. It *WILL NOT* create a new instance
// if the provided argument is already a SafeStream
func NewSafeConnection(wrapped net.Conn) *SafeConnection {
	if scs, ok := wrapped.(*SafeConnection); ok {
		return scs
	}

	return &SafeConnection{
		Conn: wrapped,
	}
}

func (ns *SafeConnection) WriteTo(w io.Writer) (n int64, err error) {
	if o, ok := ns.Conn.(io.WriterTo); ok {
		return o.WriteTo(w)
	} else {
		return io.CopyBuffer(w, ns.Conn, make([]byte, buffers.BufferSize))
	}
}

func (ns *SafeConnection) ReadFrom(r io.Reader) (n int64, err error) {
	if o, ok := ns.Conn.(io.ReaderFrom); ok {
		return o.ReadFrom(r)
	} else {
		return io.CopyBuffer(ns.Conn, r, make([]byte, buffers.BufferSize))
	}
}

// Close will close the underlying stream. If the Close has already been called, it will do nothing
func (ns *SafeConnection) Close() error {
	if ns.closed {
		return nil
	}
	err := LogClose(ns.Conn)
	ns.closed = true

	return err
}

// Closed will return `true` if SafeStream.Close has been called at least once
func (ns *SafeConnection) Closed() bool {
	return ns.closed
}

// Unwrap returns the embedded net.Conn
func (ns *SafeConnection) Unwrap() net.Conn {
	return ns.Conn
}
