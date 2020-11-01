package enc

import "fmt"

// -------------------------------------------------------

// RawEncoder encodes 8 bytes to 8 characters -- it simply does not do any translation whatsoever
type RawEncoder struct {
}

func (b *RawEncoder) Name() string {
	return "Raw"
}

func (b *RawEncoder) String() string {
	return fmt.Sprintf("%v(%v)", b.Name(), string(b.Code()))
}

func (b *RawEncoder) Code() byte {
	return 'R'
}

func (b *RawEncoder) Encode(data []byte) string {
	return string(data)
}

func (b *RawEncoder) Decode(data string) ([]byte, error) {
	return []byte(data), nil
}

func (b *RawEncoder) TestPatterns() []string {
	return []string{}
}
