package enc

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func Test_Base64uEncoder(t *testing.T) {
	encoder := Base64uEncoder{}
	encoded := encoder.Encode(encoderTest)
	require.NotContains(t, encoded, "=")
	require.NotContains(t, encoded, ".")
	decoded, err := encoder.Decode(encoded)
	require.NoError(t, err)
	require.Equal(t, encoderTest, decoded)
}
