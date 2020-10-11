package streams

import (
	"github.com/bokysan/socketace/v2/internal/util/buffers"
	"io"
)

// SafeWriter implements the io.WriteCloser and makes sure that `Close()` can be called safely multiple times.
// Calling `Close()` on a closed object will simply succeed without an error.
type SafeWriter struct {
	io.WriteCloser
	closed bool
}

func NewSafeWriter(wrapped io.WriteCloser) *SafeWriter {
	if scs, ok := wrapped.(*SafeWriter); ok {
		return scs
	}

	return &SafeWriter{
		WriteCloser: wrapped,
	}
}

func (ns *SafeWriter) ReadFrom(r io.Reader) (n int64, err error) {
	if o, ok := ns.WriteCloser.(io.ReaderFrom); ok {
		return o.ReadFrom(r)
	} else {
		return io.CopyBuffer(ns.WriteCloser, r, make([]byte, buffers.BufferSize))
	}
}

// Close will close the underlying stream. If the Close has already been called, it will do nothing
func (ns *SafeWriter) Close() error {
	if ns.closed {
		return nil
	}
	err := LogClose(ns.WriteCloser)
	ns.closed = true

	return err
}

// Closed will return `true` if SafeWriter.Close has been called at least once
func (ns *SafeWriter) Closed() bool {
	return ns.closed
}

// Unwrap returns the embedded io.WriteCloser
func (ns *SafeWriter) Unwrap() io.WriteCloser {
	return ns.WriteCloser
}
