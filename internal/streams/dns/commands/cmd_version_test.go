package commands

import (
	"github.com/bokysan/socketace/v2/internal/streams/dns/util"
	"github.com/bokysan/socketace/v2/internal/util/enc"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"testing"
)

const testProtocolVersion = 0x00001000

func Test_VersionRequest(t *testing.T) {
	r1 := &VersionRequest{
		ClientVersion: testProtocolVersion,
	}
	encoded, err := r1.Encode(enc.Base32Encoding)
	require.NoError(t, err)
	log.Infof("Encoded request: %v", encoded)

	r2 := &VersionRequest{}
	err = r2.Decode(enc.Base32Encoding, encoded)
	require.NoError(t, err)

	require.Equal(t, r1.ClientVersion, r2.ClientVersion)
}

func Test_VersionResponse1(t *testing.T) {
	for _, qt := range util.QueryTypesByPriority {
		r1 := &VersionResponse{
			ServerVersion: testProtocolVersion,
			UserId:        137,
		}
		encoded, err := r1.Encode(nil)
		require.NoError(t, err)
		log.Debugf("Encoded using %v: %v", qt, encoded)

		r2 := &VersionResponse{}
		err = r2.Decode(nil, encoded)
		require.NoError(t, err)

		require.Equal(t, r1.UserId, r2.UserId)
		require.Equal(t, r1.ServerVersion, r2.ServerVersion)
	}
}

func Test_VersionResponse2(t *testing.T) {
	for _, qt := range util.QueryTypesByPriority {
		r1 := &VersionResponse{
			ServerVersion: testProtocolVersion,
			Err:           BadServerFull,
		}
		encoded, err := r1.Encode(enc.Base32Encoding)
		require.NoError(t, err)
		log.Debugf("Encoded using %v: %v", qt, encoded)

		r2 := &VersionResponse{}
		err = r2.Decode(enc.Base32Encoding, encoded)
		require.NoError(t, err)

		require.Equal(t, r1.Err.Error(), BadServerFull.Error())
	}
}
