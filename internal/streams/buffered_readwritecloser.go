package streams

import (
	"bufio"
	"io"
)

type BufferedReadWriteCloser struct {
	conn   io.ReadWriteCloser
	Reader *bufio.Reader
}

func NewBufferedReadWriteCloser(c io.ReadWriteCloser) *BufferedReadWriteCloser {
	reader := bufio.NewReaderSize(c, MaxHeaderSize)

	b := &BufferedReadWriteCloser{
		conn:   c,
		Reader: reader,
	}

	return b
}

func (bf BufferedReadWriteCloser) Read(p []byte) (n int, err error) {
	return bf.Reader.Read(p)
}

func (bf BufferedReadWriteCloser) Write(p []byte) (n int, err error) {
	return bf.conn.Write(p)
}

func (bf BufferedReadWriteCloser) Close() error {
	return bf.conn.Close()
}