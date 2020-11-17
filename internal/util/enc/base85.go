package enc

import (
	"encoding/ascii85"
	"fmt"
	"github.com/pkg/errors"
)

// -------------------------------------------------------

// Base64Encoder encodes 3 bytes to 4 characters
type Base85Encoder struct {
}

func (b *Base85Encoder) Name() string {
	return "Base85"
}

func (b *Base85Encoder) String() string {
	return fmt.Sprintf("%v(%v)", b.Name(), string(b.Code()))
}

func (b *Base85Encoder) Code() byte {
	return 'W'
}

func (b *Base85Encoder) Encode(data []byte) []byte {
	l := ascii85.MaxEncodedLen(len(data))
	dst := make([]byte, l)
	ascii85.Encode(dst, data)
	for k, b := range dst {
		if b == '.' {
			dst[k] = 'v'
		} else if b == '\\' {
			dst[k] = 'w'
		} else if b == '`' {
			dst[k] = 'x'
		}
	}
	return dst
}

func (b *Base85Encoder) Decode(data []byte) ([]byte, error) {
	source := make([]byte, len(data))
	copy(source, data)
	for k, b := range source {
		if b == 'v' {
			source[k] = '.'
		} else if b == 'w' {
			source[k] = '\\'
		} else if b == 'x' {
			source[k] = '`'
		}
	}

	dst := make([]byte, len(source))
	ndst, _, err := ascii85.Decode(dst, source, true)
	if err != nil {
		err = errors.WithStack(err)
		return nil, err
	}
	return dst[:ndst], nil
}

func (b *Base85Encoder) TestPatterns() [][]byte {
	str := make([]byte, 85)
	// 33 (!) through 117 (u)
	for k, _ := range str {
		b := byte(k + 33)
		if b == '.' {
			str[k] = 'v'
		} else if b == '\\' {
			str[k] = 'w'
		} else if b == '`' {
			str[k] = 'x'
		} else {
			str[k] = b
		}
	}

	return [][]byte{
		str,
	}
}

func (b *Base85Encoder) Ratio() float64 {
	return 1.25
}
