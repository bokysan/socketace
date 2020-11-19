package enc

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func Test_Base85Encoder(t *testing.T) {
	for _, encoderTest := range encoderTests {
		encoder := Base85Encoder{}
		encoded := encoder.Encode(encoderTest)
		require.NotContains(t, encoded, ".")
		decoded, err := encoder.Decode(encoded)
		require.NoError(t, err)
		require.Equal(t, encoderTest, decoded)
	}
}
