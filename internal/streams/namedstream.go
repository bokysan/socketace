package streams

import (
	"io"
)

type NamedStream struct {
	wrapper io.ReadWriteCloser
	name string
}

func NewNamedStream(wrapper io.ReadWriteCloser, name string) *NamedStream {
	return &NamedStream{
		wrapper: wrapper,
		name: name,
	}
}

func (ns *NamedStream) Read(p []byte) (int, error) {
	return ns.wrapper.Read(p)
}

func (ns *NamedStream) Write(p []byte) (int, error) {
	return ns.wrapper.Write(p)
}

func (ns *NamedStream) Close() error {
	return ns.wrapper.Close()
}

func (ns *NamedStream) String() string {
	return ns.name
}