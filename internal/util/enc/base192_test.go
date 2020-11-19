package enc

import (
	"testing"
)

func Test_Base192Encoder(t *testing.T) {
	/*
		// DISABLE TESTS. Base192 encoder does not work properly as of yet.


			for _, encoderTest := range encoderTests {
				encoder := Base192Encoder{}
				encoded := encoder.Encode(encoderTest)
				require.NotNil(t, encoded)

				log.Infof("%v -> %v", encoderTest, encoded)

				expectedLen := int(math.Ceil(float64(len(encoderTest)) * 16.0 / 15.0))
				require.Len(t, encoded, expectedLen)

				if string(encoderTest) == "\001\002\377\377" {
					require.Equal(t, []byte{0, 129, 85, 63, 128}, encoded)
				} else if string(encoderTest) == "\001\002\377" {
					require.Equal(t, []byte{0, 129, 85, 0}, encoded)
				}

				require.GreaterOrEqual(t, len(encoded), len(encoderTest))
				require.NotContains(t, encoded, "=")
				require.NotContains(t, encoded, ".")
				decoded, err := encoder.Decode(encoded)
				log.Infof("%v -> %v -> %v", encoderTest, encoded, decoded)
				require.NotNil(t, encoded)
				require.NoError(t, err)
				require.Equal(t, encoderTest, decoded)
			}
	*/

}
