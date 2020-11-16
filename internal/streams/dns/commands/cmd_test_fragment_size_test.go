package commands

import (
	"github.com/bokysan/socketace/v2/internal/streams/dns/util"
	"github.com/bokysan/socketace/v2/internal/util/enc"
	"github.com/stretchr/testify/require"
	"testing"
)

func Test_TestDownstreamFragmentSizeRequest(t *testing.T) {
	r1 := &TestDownstreamFragmentSizeRequest{
		UserId:       123,
		FragmentSize: 123456,
	}
	encoded, err := r1.Encode(enc.Base32Encoding)
	require.NoError(t, err)
	// log.Infof("Encoded request: %v", encoded)

	r2 := &TestDownstreamFragmentSizeRequest{}
	err = r2.Decode(enc.Base32Encoding, encoded)
	require.NoError(t, err)

	require.Equal(t, r1.UserId, r2.UserId)
	require.Equal(t, r1.FragmentSize, r2.FragmentSize)
}

func Test_TestDownstreamFragmentSizeResponse(t *testing.T) {
	for _, e := range []enc.Encoder{enc.Base32Encoding, enc.Base91Encoding, enc.RawEncoding} {
		r1 := &TestDownstreamFragmentSizeResponse{
			Data: []byte(util.DownloadCodecCheck),
		}
		encoded, err := r1.Encode(e)
		require.NoError(t, err)
		// log.Infof("Encoded response using %v: %v", e, encoded)

		r2 := &TestDownstreamFragmentSizeResponse{}
		err = r2.Decode(e, encoded)
		require.NoError(t, err)
		require.Equal(t, r1.Data, r2.Data)
	}
}

func Test_TestDownstreamFragmentSizeResponseErr(t *testing.T) {
	r1 := &TestDownstreamFragmentSizeResponse{
		Data: nil,
		Err:  BadCodec,
	}
	encoded, err := r1.Encode(enc.Base128Encoding)
	require.NoError(t, err)

	r2 := &TestDownstreamFragmentSizeResponse{
		Data: nil,
		Err:  BadCodec,
	}
	err = r2.Decode(enc.Base128Encoding, encoded)
	require.NoError(t, err)
	require.Equal(t, r1.Data, r2.Data)
	require.Equal(t, r1.Err, r2.Err)
}
