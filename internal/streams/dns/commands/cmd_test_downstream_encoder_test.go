package commands

import (
	"github.com/bokysan/socketace/v2/internal/streams/dns/util"
	"github.com/bokysan/socketace/v2/internal/util/enc"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"testing"
)

func Test_TestDownstreamEncoderRequest(t *testing.T) {
	for _, e := range []enc.Encoder{
		enc.Base32Encoding,
		enc.Base64Encoding,
		enc.Base64uEncoding,
		enc.Base85Encoding,
		enc.Base91Encoding,
		enc.Base128Encoding,
		enc.RawEncoding,
	} {

		r1 := &TestDownstreamEncoderRequest{
			DownstreamEncoder: e,
		}
		encoded, err := r1.Encode(enc.Base32Encoding)
		require.NoError(t, err)
		log.Infof("Encoded request: %v", encoded)

		r2 := &TestDownstreamEncoderRequest{}
		err = r2.Decode(enc.Base32Encoding, encoded)
		require.NoError(t, err)

		require.Equal(t, r1.DownstreamEncoder.Code(), r2.DownstreamEncoder.Code())
	}
}

func Test_TestDownstreamEncoderResponse(t *testing.T) {
	for _, e := range []enc.Encoder{enc.Base32Encoding, enc.RawEncoding} {
		r1 := &TestDownstreamEncoderResponse{
			Data: []byte(util.DownloadCodecCheck),
		}
		encoded, err := r1.Encode(e)
		require.NoError(t, err)
		log.Infof("Encoded response using %v: %v", e, encoded)

		r2 := &TestDownstreamEncoderResponse{
			Data: []byte(util.DownloadCodecCheck),
		}
		err = r2.Decode(e, encoded)
		require.NoError(t, err)
		require.Equal(t, r1.Data, r2.Data)
	}
}
