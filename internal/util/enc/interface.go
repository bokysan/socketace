package enc

import (
	"github.com/pkg/errors"
	"strings"
)

type Encoder interface {
	// Name is the user-friendly name of this encoder
	Name() string

	// Code represends the short (one-letter) code for the encoder
	Code() byte

	// Encode will take an array of bytes and encode it using this encoder
	Encode([]byte) []byte

	// Decode is the reverse proces of encoding
	Decode([]byte) ([]byte, error)

	// Return a list of test patterns for the specified encoding
	TestPatterns() [][]byte

	// Expansion ratio; e.g. the encoded array is this times longer than the original input
	Ratio() float64
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

// FromCode will return an encoder based on encoder code
func FromCode(code byte) (Encoder, error) {
	code = strings.ToUpper(string(code))[0]
	for _, enc := range []Encoder{
		Base32Encoding,
		Base64Encoding,
		Base64uEncoding,
		Base85Encoding,
		Base91Encoding,
		Base128Encoding,
		RawEncoding,
	} {
		if enc.Code() == code {
			return enc, nil
		}
	}
	return nil, errors.Errorf("Unknown codec type: %v", code)
}
