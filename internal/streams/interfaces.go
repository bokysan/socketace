package streams

import (
	"io"
	"net"
)

// Closed is an interface which defines if a method to check if a stream is closed or not
type Closed interface {
	Closed() bool
}

type ReadCloserClosed interface {
	io.ReadCloser
	Closed
}

type WriteCloserClosed interface {
	io.WriteCloser
	Closed
}

type ReadWriteCloserClosed interface {
	io.ReadWriteCloser
	Closed
}

// Connection combines the net.Connection and Closed interfaces to provide the way to query if the
// connection has been closed or not.
type Connection interface {
	net.Conn
	Closed
}

type UnwrappedReadCloser interface {
	Unwrap() io.ReadCloser
}

type UnwrappedWriteCloser interface {
	Unwrap() io.WriteCloser
}

type UnwrappedReadWriteCloser interface {
	Unwrap() io.ReadWriteCloser
}

type UnwrappedConnection interface {
	Unwrap() net.Conn
}
