package streams

import (
	"github.com/bokysan/socketace/v2/internal/util/buffers"
	"io"
)

// SafeStream makes sure that `Close()` can be called safely multiple times. Calling `Close()` on a closed object
// will simply succeed without an error.
type SafeStream struct {
	io.ReadWriteCloser
	closed bool
}

// NewSafeStream will, create a new SafeStream with a given name. It *WILL NOT* create a new instance
// if the provided argument is already a SafeStream
func NewSafeStream(wrapped io.ReadWriteCloser) *SafeStream {
	if scs, ok := wrapped.(*SafeStream); ok {
		return scs
	}

	return &SafeStream{
		ReadWriteCloser: wrapped,
	}
}

func (ns *SafeStream) WriteTo(w io.Writer) (n int64, err error) {
	if o, ok := ns.ReadWriteCloser.(io.WriterTo); ok {
		return o.WriteTo(w)
	} else {
		return io.CopyBuffer(w, ns.ReadWriteCloser, make([]byte, buffers.BufferSize))
	}
}

func (ns *SafeStream) ReadFrom(r io.Reader) (n int64, err error) {
	if o, ok := ns.ReadWriteCloser.(io.ReaderFrom); ok {
		return o.ReadFrom(r)
	} else {
		return io.CopyBuffer(ns.ReadWriteCloser, r, make([]byte, buffers.BufferSize))
	}
}

// Close will close the underlying stream. If the Close has already been called, it will do nothing
func (ns *SafeStream) Close() error {
	if ns.closed {
		return nil
	}
	err := LogClose(ns.ReadWriteCloser)
	ns.closed = true

	return err
}

// Closed will return `true` if SafeStream.Close has been called at least once
func (ns *SafeStream) Closed() bool {
	return ns.closed
}

// Unwrap returns the embedded io.ReadWriteCloser
func (ns *SafeStream) Unwrap() io.ReadWriteCloser {
	return ns.ReadWriteCloser
}
