package streams

import (
	"github.com/hashicorp/go-multierror"
	"io"
)

// ReadWriteCloser converts one input stream and one output stream into a `ReadWriteCloser`.
type ReadWriteCloser struct {
	reader io.ReadCloser
	writer io.WriteCloser
}

func NewReadWriteCloser(reader io.ReadCloser, writer io.WriteCloser) *ReadWriteCloser {
	return &ReadWriteCloser{
		reader: reader,
		writer: writer,
	}
}

func (sc *ReadWriteCloser) Read(p []byte) (int, error) {
	return sc.reader.Read(p)
}

func (sc *ReadWriteCloser) Write(p []byte) (int, error) {
	return sc.writer.Write(p)
}

func (sc *ReadWriteCloser) Close() error {
	var errs error
	errs = multierror.Append(errs, sc.reader.Close())
	errs = multierror.Append(errs, sc.writer.Close())
	return errs
}
