package commands

import (
	"github.com/bokysan/socketace/v2/internal/util/enc"
	"github.com/stretchr/testify/require"
	"testing"
)

func Test_SetOptionsRequest1(t *testing.T) {
	r1 := &SetOptionsRequest{
		UserId: 123,
	}
	encoded, err := r1.Encode(enc.Base32Encoding)
	require.NoError(t, err)

	r2 := &SetOptionsRequest{}
	err = r2.Decode(enc.Base32Encoding, encoded)
	require.NoError(t, err)

	require.Equal(t, r1.UserId, r2.UserId)
	require.Equal(t, r1.LazyMode, r2.LazyMode)
	require.Equal(t, r1.MultiQuery, r2.MultiQuery)
	require.Equal(t, r1.DownstreamEncoder, r2.DownstreamEncoder)
	require.Equal(t, r1.UpstreamEncoder, r2.UpstreamEncoder)
	require.Equal(t, r1.DownstreamFragmentSize, r2.DownstreamFragmentSize)
}

func Test_SetOptionsRequest2(t *testing.T) {
	tf := true
	fs := uint32(0x1234)

	r1 := &SetOptionsRequest{
		UserId:                 123,
		LazyMode:               &tf,
		MultiQuery:             nil,
		DownstreamEncoder:      enc.Base85Encoding,
		UpstreamEncoder:        enc.Base91Encoding,
		DownstreamFragmentSize: &fs,
	}

	encoded, err := r1.Encode(nil)
	require.NoError(t, err)

	r2 := &SetOptionsRequest{}
	err = r2.Decode(nil, encoded)
	require.NoError(t, err)

	require.Equal(t, r1.UserId, r2.UserId)
	require.Equal(t, r1.LazyMode, r2.LazyMode)
	require.Equal(t, r1.MultiQuery, r2.MultiQuery)
	require.Equal(t, r1.DownstreamEncoder, r2.DownstreamEncoder)
	require.Equal(t, r1.UpstreamEncoder, r2.UpstreamEncoder)
	require.Equal(t, r1.DownstreamFragmentSize, r2.DownstreamFragmentSize)
}

func Test_SetOptionsResponse(t *testing.T) {
	r1 := &SetOptionsResponse{}
	encoded, err := r1.Encode(enc.Base85Encoding)
	require.NoError(t, err)
	r2 := &SetOptionsResponse{}
	err = r2.Decode(enc.Base85Encoding, encoded)
	require.NoError(t, err)
	require.Nil(t, r2.Err)
}

func Test_SetOptionsResponseErr(t *testing.T) {
	r1 := &SetOptionsResponse{
		Err: BadCodec,
	}
	encoded, err := r1.Encode(enc.Base128Encoding)
	require.NoError(t, err)

	r2 := &SetOptionsResponse{
		Err: BadCodec,
	}
	err = r2.Decode(enc.Base128Encoding, encoded)
	require.NoError(t, err)
	require.Equal(t, r1.Err, r2.Err)
}
