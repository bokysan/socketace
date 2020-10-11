package streams

import (
	"github.com/bokysan/socketace/v2/internal/util/buffers"
	"io"
	"net"
	"time"
)

var Localhost = MustResolveTcpAddress("tcp", "localhost:0")

// SimulatedConnection will simulate a net.Conn (implement its interfaces) while proxying calls to
// Read and Write to the undrlying input/output stream
type SimulatedConnection struct {
	ReadWriteCloserClosed
	localAddress  net.Addr
	remoteAddress net.Addr
}

// StreamWrappedConnection will wrap an input/output stream into a net.Conn interface while proxying
// the connection-specific calls to the underlying connection
type StreamWrappedConnection struct {
	ReadWriteCloserClosed
	underlying net.Conn
}

func MustResolveTcpAddress(network, address string) *net.TCPAddr {
	addr, err := net.ResolveTCPAddr(network, address)
	if err != nil {
		panic(err)
	}
	return addr
}

func NewSimulatedConnection(wrapped io.ReadWriteCloser, local, remote net.Addr) *SimulatedConnection {
	return &SimulatedConnection{
		ReadWriteCloserClosed: NewSafeStream(wrapped),
		localAddress:          local,
		remoteAddress:         remote,
	}
}

func NewStreamConnection(wrapped io.ReadWriteCloser, underlying net.Conn) *StreamWrappedConnection {
	return &StreamWrappedConnection{
		ReadWriteCloserClosed: NewSafeStream(wrapped),
		underlying:            underlying,
	}
}

func (sc *SimulatedConnection) LocalAddr() net.Addr {
	return sc.localAddress
}

func (sc *SimulatedConnection) RemoteAddr() net.Addr {
	return sc.remoteAddress
}

func (sc *SimulatedConnection) SetDeadline(t time.Time) error {
	return nil
}

func (sc *SimulatedConnection) SetReadDeadline(t time.Time) error {
	return nil
}

func (sc *SimulatedConnection) SetWriteDeadline(t time.Time) error {
	return nil
}

func (sc *SimulatedConnection) WriteTo(w io.Writer) (n int64, err error) {
	if o, ok := sc.ReadWriteCloserClosed.(io.WriterTo); ok {
		return o.WriteTo(w)
	} else {
		return io.CopyBuffer(w, sc.ReadWriteCloserClosed, make([]byte, buffers.BufferSize))
	}
}

func (sc *SimulatedConnection) ReadFrom(r io.Reader) (n int64, err error) {
	if o, ok := sc.ReadWriteCloserClosed.(io.ReaderFrom); ok {
		return o.ReadFrom(r)
	} else {
		return io.CopyBuffer(sc.ReadWriteCloserClosed, r, make([]byte, buffers.BufferSize))
	}
}

func (sc *SimulatedConnection) Unwrap() io.ReadWriteCloser {
	return sc.ReadWriteCloserClosed
}

func (sc *StreamWrappedConnection) LocalAddr() net.Addr {
	return sc.underlying.LocalAddr()
}

// RemoteAddr returns the remote network address.
func (sc *StreamWrappedConnection) RemoteAddr() net.Addr {
	return sc.underlying.RemoteAddr()
}

func (sc *StreamWrappedConnection) SetDeadline(t time.Time) error {
	return sc.underlying.SetDeadline(t)
}

func (sc *StreamWrappedConnection) SetReadDeadline(t time.Time) error {
	return sc.underlying.SetReadDeadline(t)
}

func (sc *StreamWrappedConnection) SetWriteDeadline(t time.Time) error {
	return sc.underlying.SetWriteDeadline(t)
}

func (sc *StreamWrappedConnection) WriteTo(w io.Writer) (n int64, err error) {
	if o, ok := sc.ReadWriteCloserClosed.(io.WriterTo); ok {
		return o.WriteTo(w)
	} else {
		return io.CopyBuffer(w, sc.ReadWriteCloserClosed, make([]byte, buffers.BufferSize))
	}
}

func (sc *StreamWrappedConnection) ReadFrom(r io.Reader) (n int64, err error) {
	if o, ok := sc.ReadWriteCloserClosed.(io.ReaderFrom); ok {
		return o.ReadFrom(r)
	} else {
		return io.CopyBuffer(sc.ReadWriteCloserClosed, r, make([]byte, buffers.BufferSize))
	}
}

func (sc *StreamWrappedConnection) Unwrap() io.ReadWriteCloser {
	return sc.ReadWriteCloserClosed
}
