package enc

type Encoder interface {
	// Name is the user-friendly name of this encoder
	Name() string
	// Code represends the short (one-letter) code for the encoder
	Code() byte

	// Encode will take an array of bytes and encode it using this encoder
	Encode([]byte) string

	// Decode is the reverse proces of encoding
	Decode(string) ([]byte, error)

	// PlacesDots returns true if this encoder takes care of placing the dots automatically in the domain name
	PlacesDots() bool

	// EatsDots returns true if this encoder "eats" / processes the dots
	EatsDots() bool

	// BlocksizeRaw returns the block size (number of bytes) this encoder takes at one time
	BlocksizeRaw() int

	// BlocksizeEncoded returns the block size (number of bytes) output by this encoder for every input block
	BlocksizeEncoded() int

	// Return a list of test patterns for the specified encoding
	TestPatterns() []string
}
