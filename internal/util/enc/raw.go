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

func (b *RawEncoder) Encode(data []byte) []byte {
	return data
}

func (b *RawEncoder) Decode(data []byte) ([]byte, error) {
	return data, nil
}

func (b *RawEncoder) TestPatterns() [][]byte {
	return [][]byte{}
}

func (b *RawEncoder) Ratio() float64 {
	return 1.00
}
