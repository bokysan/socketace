package enc

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func Test_Base91Encoder(t *testing.T) {
	encoder := Base91Encoder{}
	encoded := encoder.Encode(encoderTest)
	require.NotContains(t, encoded, ".")
	decoded, err := encoder.Decode(encoded)
	require.NoError(t, err)
	require.Equal(t, encoderTest, decoded)
}
