package streams

import (
	"github.com/bokysan/socketace/v2/internal/util/buffers"
	"github.com/hashicorp/go-multierror"
	"io"
)

// ReadWriteCloser converts one input stream and one output stream into a `ReadWriteCloser`.
type ReadWriteCloser struct {
	ReadCloserClosed
	WriteCloserClosed
}

func NewReadWriteCloser(reader io.ReadCloser, writer io.WriteCloser) *ReadWriteCloser {
	return &ReadWriteCloser{
		ReadCloserClosed:  NewSafeReader(reader),
		WriteCloserClosed: NewSafeWriter(writer),
	}
}

func (sc *ReadWriteCloser) WriteTo(w io.Writer) (n int64, err error) {
	if o, ok := sc.ReadCloserClosed.(io.WriterTo); ok {
		return o.WriteTo(w)
	} else {
		return io.CopyBuffer(w, sc.ReadCloserClosed, make([]byte, buffers.BufferSize))
	}
}

func (sc *ReadWriteCloser) ReadFrom(r io.Reader) (n int64, err error) {
	if o, ok := sc.WriteCloserClosed.(io.ReaderFrom); ok {
		return o.ReadFrom(r)
	} else {
		return io.CopyBuffer(sc.WriteCloserClosed, r, make([]byte, buffers.BufferSize))
	}
}

func (sc *ReadWriteCloser) Close() error {
	var errs error
	if err := LogClose(sc.ReadCloserClosed); err != nil {
		errs = multierror.Append(errs, err)
	}
	if err := LogClose(sc.WriteCloserClosed); err != nil {
		errs = multierror.Append(errs, err)
	}
	return errs
}

// Closed will return `true` if both reader and writer are closed
func (sc *ReadWriteCloser) Closed() bool {
	return sc.ReadCloserClosed.Closed() && sc.WriteCloserClosed.Closed()
}
