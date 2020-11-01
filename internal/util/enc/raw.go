package enc

// -------------------------------------------------------

// RawEncoder encodes 8 bytes to 8 characters -- it simply does not do any translation whatsoever
type RawEncoder struct {
}

func (b *RawEncoder) Name() string {
	return "Raw"
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

func (b *RawEncoder) PlacesDots() bool {
	return false
}

func (b *RawEncoder) EatsDots() bool {
	return false
}

func (b *RawEncoder) BlocksizeRaw() int {
	return 8
}

func (b *RawEncoder) BlocksizeEncoded() int {
	return 8
}

func (b *RawEncoder) TestPatterns() []string {
	return []string{}
}
