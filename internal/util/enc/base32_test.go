package enc

import (
	"github.com/stretchr/testify/require"
	"testing"
)

var encoderTest = []byte("\000\000\000\000\377\377\377\377\125\125\125\125\252\252\252\252" +
	"\201\143\310\322\307\174\262\027\137\117\316\311\111\055\122\041" +
	"\141\251\161\040\045\263\006\163\346\330\104\060\171\120\127\277")

func Test_Base32Encoder(t *testing.T) {
	encoder := Base32Encoder{}
	encoded := encoder.Encode(encoderTest)
	require.NotContains(t, encoded, "=")
	require.NotContains(t, encoded, ".")
	decoded, err := encoder.Decode(encoded)
	require.NoError(t, err)
	require.Equal(t, encoderTest, decoded)
}
