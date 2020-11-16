package commands

import (
	"github.com/bokysan/socketace/v2/internal/streams/dns/util"
	"github.com/bokysan/socketace/v2/internal/util/enc"
	"github.com/stretchr/testify/require"
	"testing"
)

func Test_ChunkRequest(t *testing.T) {
	r1 := &PacketRequest{
		UserId:         123,
		LastAckedSeqNo: 64321,
		Packet: &util.Packet{
			SeqNo: 12345,
			Data:  []byte{0x17, 0x18, 0x19, 0x1A},
		},
	}
	encoded, err := r1.Encode(enc.Base91Encoding)
	require.NoError(t, err)

	r2 := &PacketRequest{}
	err = r2.Decode(enc.Base91Encoding, encoded)
	require.NoError(t, err)

	require.Equal(t, r1.UserId, r2.UserId)
	require.Equal(t, r1.LastAckedSeqNo, r2.LastAckedSeqNo)
	require.Equal(t, r1.Packet.SeqNo, r2.Packet.SeqNo)
	require.Equal(t, r1.Packet.Data, r2.Packet.Data)
}

func Test_PacketResponse(t *testing.T) {
	r1 := &PacketResponse{
		LastAckedSeqNo: 64321,
		Packet: &util.Packet{
			SeqNo: 12345,
			Data:  []byte{0x17, 0x18, 0x19, 0x1A},
		},
	}
	encoded, err := r1.Encode(enc.Base91Encoding)
	require.NoError(t, err)

	r2 := &PacketResponse{}
	err = r2.Decode(enc.Base91Encoding, encoded)
	require.NoError(t, err)

	require.Equal(t, r1.LastAckedSeqNo, r2.LastAckedSeqNo)
	require.Equal(t, r1.Packet.SeqNo, r2.Packet.SeqNo)
	require.Equal(t, r1.Packet.Data, r2.Packet.Data)
	require.Equal(t, r1.Err, r2.Err)
}

func Test_PacketResponseErr(t *testing.T) {
	r1 := &PacketResponse{
		Err: NoData,
	}
	encoded, err := r1.Encode(enc.Base91Encoding)
	require.NoError(t, err)

	r2 := &PacketResponse{}
	err = r2.Decode(enc.Base91Encoding, encoded)
	require.NoError(t, err)

	require.Equal(t, r1.Packet, r2.Packet)
	require.Equal(t, r1.Err, r2.Err)
}
