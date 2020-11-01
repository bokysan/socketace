package dns

import (
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"testing"
)

func Test_VersionRequest(t *testing.T) {
	r1 := &VersionRequest{
		ClientVersion: ProtocolVersion,
	}
	encoded, err := r1.Encode(Base32Encoding, testDomain)
	require.NoError(t, err)
	log.Infof("Encoded request: %v", encoded)

	r2 := &VersionRequest{}
	err = r2.Decode(Base32Encoding, encoded, testDomain)
	require.NoError(t, err)

	require.Equal(t, r1.ClientVersion, r2.ClientVersion)
}

func Test_VersionResponse1(t *testing.T) {
	for _, qt := range QueryTypesByPriority {
		r1 := &VersionResponse{
			ServerVersion: ProtocolVersion,
			UserId:        137,
		}
		encoded, err := r1.Encode(Base32Encoding, qt)
		require.NoError(t, err)
		log.Infof("Encoded using %v: %v", qt, encoded)

		r2 := &VersionResponse{}
		err = r2.Decode(Base32Encoding, encoded)
		require.NoError(t, err)

		require.Equal(t, r1.ServerVersion, r2.ServerVersion)
	}
}

func Test_VersionResponse2(t *testing.T) {
	for _, qt := range QueryTypesByPriority {
		r1 := &VersionResponse{
			ServerVersion: ProtocolVersion,
			Err:           &BadServerFull,
		}
		encoded, err := r1.Encode(Base32Encoding, qt)
		require.NoError(t, err)
		log.Infof("Encoded using %v: %v", qt, encoded)

		r2 := &VersionResponse{}
		err = r2.Decode(Base32Encoding, encoded)
		require.NoError(t, err)

		require.Equal(t, r1.Err.Error(), BadServerFull.Error())
	}
}
