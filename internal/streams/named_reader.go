package streams

import (
	"fmt"
	"github.com/bokysan/socketace/v2/internal/util/buffers"
	"io"
)

// NamedReader implements the io.ReadCloser interface as well as fmt.Stringer. It allows the caller to setup
// a name for the stream which will be returned when outputing the stream with `%v`.
// It also makes sure that `Close()` can be called safely multiple times. Calling `Close()` on a closed object
// will simply succeed without an error.
type NamedReader struct {
	ReadCloserClosed
	name string
}

func NewNamedReader(wrapped io.ReadCloser, name string) *NamedReader {
	return &NamedReader{
		ReadCloserClosed: NewSafeReader(wrapped),
		name:             name,
	}
}

func (ns *NamedReader) WriteTo(w io.Writer) (n int64, err error) {
	if o, ok := ns.ReadCloserClosed.(io.WriterTo); ok {
		return o.WriteTo(w)
	} else {
		return io.CopyBuffer(w, ns.ReadCloserClosed, make([]byte, buffers.BufferSize))
	}
}

func (ns *NamedReader) String() string {
	result := ns.name

	var s io.ReadCloser
	s = ns.ReadCloserClosed
	for true {
		if t, ok := s.(UnwrappedReadCloser); ok {
			u := t.Unwrap()
			if v, ok := u.(fmt.Stringer); ok {
				result += "->" + v.String()
				break
			}
			s = u
		} else {
			break
		}
	}

	return result
}

func (ns *NamedReader) Unwrap() io.ReadCloser {
	return ns.ReadCloserClosed
}
