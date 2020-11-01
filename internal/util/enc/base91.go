package enc

import (
	"github.com/mtraver/base91"
	"github.com/pkg/errors"
)

const (
	cb91 = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789!#$%&()*+,-/:;<=>?@[]^_`{|}~\""
)

var iodineBase91Encoding = base91.NewEncoding(cb91)

// -------------------------------------------------------

// Base91Encoder when encoding, each group of 13 bits is converted into 2 radix-91 digits.
type Base91Encoder struct {
}

func (b *Base91Encoder) Name() string {
	return "Base91"
}

func (b *Base91Encoder) Code() byte {
	return 'X'
}

func (b *Base91Encoder) Encode(data []byte) string {
	return iodineBase91Encoding.EncodeToString(data)
}

func (b *Base91Encoder) Decode(data string) ([]byte, error) {
	res, err := iodineBase91Encoding.DecodeString(data)
	if err != nil {
		err = errors.WithStack(err)
		return nil, err
	}
	return res, nil
}

func (b *Base91Encoder) TestPatterns() []string {
	return []string{
		cb91,
	}
}
