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

	// Return a list of test patterns for the specified encoding
	TestPatterns() []string
}

// Declare a list of encodings
var (
	Base32Encoding  Encoder = &Base32Encoder{}
	Base64Encoding  Encoder = &Base64Encoder{}
	Base64uEncoding Encoder = &Base64uEncoder{}
	Base85Encoding  Encoder = &Base85Encoder{}
	Base91Encoding  Encoder = &Base91Encoder{}
	Base128Encoding Encoder = &Base128Encoder{}
	RawEncoding     Encoder = &RawEncoder{}
)
