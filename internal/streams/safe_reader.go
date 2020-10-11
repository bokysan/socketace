package streams

import (
	"github.com/bokysan/socketace/v2/internal/util/buffers"
	"io"
)

// SafeReader implements the io.ReadCloser and makes sure that `Close()` can be called safely multiple times.
// Calling `Close()` on a closed object will simply succeed without an error.
type SafeReader struct {
	io.ReadCloser
	closed bool
}

func NewSafeReader(wrapped io.ReadCloser) *SafeReader {
	if scs, ok := wrapped.(*SafeReader); ok {
		return scs
	}

	return &SafeReader{
		ReadCloser: wrapped,
	}
}

func (ns *SafeReader) WriteTo(w io.Writer) (n int64, err error) {
	if o, ok := ns.ReadCloser.(io.WriterTo); ok {
		return o.WriteTo(w)
	} else {
		return io.CopyBuffer(w, ns.ReadCloser, make([]byte, buffers.BufferSize))
	}
}

// Close will close the underlying stream. If the Close has already been called, it will do nothing
func (ns *SafeReader) Close() error {
	if ns.closed {
		return nil
	}
	err := LogClose(ns.ReadCloser)
	ns.closed = true

	return err
}

// Closed will return `true` if SafeReader.Close has been called at least once
func (ns *SafeReader) Closed() bool {
	return ns.closed
}

// Unwrap returns the embedded io.ReadCloser
func (ns *SafeReader) Unwrap() io.ReadCloser {
	return ns.ReadCloser
}
