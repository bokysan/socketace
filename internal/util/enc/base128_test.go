package enc

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func Test_Base128Transliterate(t *testing.T) {
	str := make([]byte, 127)
	for k, _ := range str {
		str[k] = byte(k)
	}
	trans := escape128(str)
	require.Equal(t, len(str), len(trans))

	back := unescape128(trans)
	require.Equal(t, len(str), len(back))
	require.Equal(t, str, back)
}

func Test_Base128Encoder(t *testing.T) {
	encoder := Base128Encoder{}
	encoded := encoder.Encode(encoderTest)
	require.NotContains(t, encoded, ".")
	decoded, err := encoder.Decode(encoded)
	require.NoError(t, err)
	require.Equal(t, encoderTest, decoded)
}
