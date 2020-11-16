package commands

import (
	"github.com/bokysan/socketace/v2/internal/util/enc"
	"github.com/stretchr/testify/require"
	"testing"
)

func Test_TestUpstreamEncoderRequest(t *testing.T) {
	for _, e := range []enc.Encoder{
		enc.Base32Encoding,
		enc.Base64Encoding,
		enc.Base64uEncoding,
		enc.Base85Encoding,
		enc.Base91Encoding,
		enc.Base128Encoding,
	} {
		for _, pat := range e.TestPatterns() {
			r1 := &TestUpstreamEncoderRequest{
				UserId:  123,
				Pattern: pat,
			}

			encoded, err := r1.Encode(e)
			require.NoError(t, err)

			r2 := &TestUpstreamEncoderRequest{}
			err = r2.Decode(e, encoded)
			require.NoError(t, err)

			require.Equal(t, r1.Pattern, r2.Pattern)
			require.Equal(t, r1.UserId, r2.UserId)

		}
	}
}

func Test_TestUpstreamEncoderResponse(t *testing.T) {
	for _, e := range []enc.Encoder{
		enc.Base32Encoding,
		enc.Base64Encoding,
		enc.Base64uEncoding,
		enc.Base85Encoding,
		enc.Base91Encoding,
		enc.Base128Encoding,
	} {
		for _, pat := range e.TestPatterns() {
			r1 := &TestUpstreamEncoderResponse{
				Data: pat,
			}
			encoded, err := r1.Encode(enc.RawEncoding)
			require.NoError(t, err)

			r2 := &TestUpstreamEncoderResponse{}
			err = r2.Decode(enc.RawEncoding, encoded)
			require.NoError(t, err)
			require.Equal(t, r1.Data, r2.Data)

		}
	}
}

func Test_TestUpstreamEncoderResponseErr(t *testing.T) {
	r1 := &TestUpstreamEncoderResponse{
		Err: BadCodec,
	}
	encoded, err := r1.Encode(enc.RawEncoding)
	require.NoError(t, err)

	r2 := &TestUpstreamEncoderResponse{}
	err = r2.Decode(enc.RawEncoding, encoded)
	require.NoError(t, err)
	require.Equal(t, r1.Err, BadCodec)
}
