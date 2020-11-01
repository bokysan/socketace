package enc

import (
	"encoding/ascii85"
	"github.com/pkg/errors"
	"strings"
)

// -------------------------------------------------------

// Base64Encoder encodes 3 bytes to 4 characters
type Base85Encoder struct {
}

func (b *Base85Encoder) Name() string {
	return "Base85"
}

func (b *Base85Encoder) Code() byte {
	return 'W'
}

func (b *Base85Encoder) Encode(data []byte) string {
	l := ascii85.MaxEncodedLen(len(data))
	dst := make([]byte, l)
	ascii85.Encode(dst, data)
	return strings.Replace(string(dst), ".", "x", -1)
}

func (b *Base85Encoder) Decode(data string) ([]byte, error) {
	dst := make([]byte, len(data))
	ndst, _, err := ascii85.Decode(dst, []byte(strings.Replace(data, "x", ".", -1)), true)
	if err != nil {
		err = errors.WithStack(err)
		return nil, err
	}
	return dst[:ndst], nil
}

func (b *Base85Encoder) PlacesDots() bool {
	return false
}

func (b *Base85Encoder) EatsDots() bool {
	return false
}

func (b *Base85Encoder) BlocksizeRaw() int {
	return 4
}

func (b *Base85Encoder) BlocksizeEncoded() int {
	return 5
}

func (b *Base85Encoder) TestPatterns() []string {
	str := make([]byte, 85)
	// 33 (!) through 117 (u)
	for k, _ := range str {
		str[k] = byte(k + 33)
	}

	return []string{
		strings.Replace(string(str), ".", "x", -1),
	}
}
