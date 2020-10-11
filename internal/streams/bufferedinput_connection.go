package streams

import (
	"bufio"
	"net"
)

const (
	MaxHeaderSize = 4096
)

// BufferedInputConnection will buffer the incoming data (the "read") part of the io.ReadWriteCloser
type BufferedInputConnection struct {
	Connection
	Reader *bufio.Reader
}

func NewBufferedInputConnection(c net.Conn) *BufferedInputConnection {
	stream := NewSafeConnection(c)
	reader := bufio.NewReaderSize(stream, MaxHeaderSize)

	b := &BufferedInputConnection{
		Connection: stream,
		Reader:     reader,
	}

	return b
}

func (bf BufferedInputConnection) Read(p []byte) (n int, err error) {
	return bf.Reader.Read(p)
}
