package commands

import (
	"bytes"
	"encoding/binary"
	"github.com/bokysan/socketace/v2/internal/streams/dns/util"
	"github.com/bokysan/socketace/v2/internal/util/enc"
	"github.com/pkg/errors"
	"io"
)

var CmdPacket = Command{
	Code:        'c',
	NeedsUserId: true,
	NewRequest: func() Request {
		return &PacketRequest{}
	},
	NewResponse: func() Response {
		return &PacketResponse{}
	},
}

type PacketRequest struct {
	UserId         uint16
	LastAckedSeqNo uint16
	Packet         *util.Packet
}

func (vr *PacketRequest) Command() Command {
	return CmdPacket
}

func (vr *PacketRequest) Encode(e enc.Encoder) (string, error) {
	hostname := EncodeRequestHeader(vr.Command(), vr.UserId)

	data := &bytes.Buffer{}
	if err := binary.Write(data, binary.LittleEndian, vr.LastAckedSeqNo); err != nil {
		return "", nil
	}

	if vr.Packet != nil {
		if err := data.WriteByte(0xFF); err != nil {
			return "", err
		}
		if err := binary.Write(data, binary.LittleEndian, vr.Packet.SeqNo); err != nil {
			return "", nil
		}
		if _, err := data.Write(vr.Packet.Data); err != nil {
			return "", err
		}
	} else {
		if err := data.WriteByte(0x00); err != nil {
			return "", err
		}
	}

	hostname += e.Encode(data.Bytes())
	return hostname, nil
}

func (vr *PacketRequest) Decode(e enc.Encoder, req string) error {
	// Verify the request is of proper command
	if rem, userId, err := DecodeRequestHeader(vr.Command(), req); err != nil {
		return errors.WithStack(err)
	} else {
		req = rem
		vr.UserId = userId
	}

	var data *bytes.Buffer

	if d, err := e.Decode(req); err != nil {
		return errors.WithStack(err)
	} else {
		data = bytes.NewBuffer(d)
	}

	if err := binary.Read(data, binary.LittleEndian, &vr.LastAckedSeqNo); err != nil {
		return errors.WithStack(err)
	}

	if hasData, err := data.ReadByte(); err != nil {
		return err
	} else if hasData&1 != 0 {
		vr.Packet = &util.Packet{}
		if err := binary.Read(data, binary.LittleEndian, &vr.Packet.SeqNo); err != nil {
			return errors.WithStack(err)
		}
		vr.Packet.Data = make([]byte, data.Len())
		if n, err := data.Read(vr.Packet.Data); err != io.EOF && err != nil {
			return errors.WithStack(err)
		} else {
			vr.Packet.Data = vr.Packet.Data[:n]
		}
	}
	return nil
}

type PacketResponse struct {
	Err            error
	LastAckedSeqNo uint16
	Packet         *util.Packet
}

func (vr *PacketResponse) Command() Command {
	return CmdPacket
}

func (vr *PacketResponse) Encode(e enc.Encoder) (string, error) {
	data := &bytes.Buffer{}
	if vr.Err != nil {
		if err := data.WriteByte(0xFF); err != nil {
			return "", err
		}
		if _, err := data.WriteString(vr.Err.Error()); err != nil {
			return "", err
		}
	} else if vr.Packet != nil {
		if err := data.WriteByte(0x01); err != nil {
			return "", err
		}
		if err := binary.Write(data, binary.LittleEndian, vr.LastAckedSeqNo); err != nil {
			return "", nil
		}
		if err := binary.Write(data, binary.LittleEndian, vr.Packet.SeqNo); err != nil {
			return "", nil
		}
		if _, err := data.Write(vr.Packet.Data); err != nil {
			return "", err
		}
	} else {
		if err := data.WriteByte(0x00); err != nil {
			return "", err
		}
		if err := binary.Write(data, binary.LittleEndian, vr.LastAckedSeqNo); err != nil {
			return "", nil
		}
	}
	return vr.Command().String() + e.Encode(data.Bytes()), nil
}
func (vr *PacketResponse) Decode(e enc.Encoder, req string) error {
	// Verify the request is of proper command
	if err := vr.Command().ValidateType(req); err != nil {
		return err
	}

	var data *bytes.Buffer

	if d, err := e.Decode(req[1:]); err != nil {
		return err
	} else {
		data = bytes.NewBuffer(d)
	}

	if status, err := data.ReadByte(); err != nil {
		return err
	} else if status == 0xFF {
		// Error flag raised
		str, err := data.ReadString(0)
		if err != io.EOF {
			return errors.WithStack(err)
		}
		for _, e := range BadErrors {
			if e.Error() == str {
				vr.Err = e
				return nil
			}
		}
		vr.Err = errors.New(str)
		return nil
	} else if status == 0x01 {
		if err := binary.Read(data, binary.LittleEndian, &vr.LastAckedSeqNo); err != nil {
			return err
		}
		vr.Packet = &util.Packet{}
		if err := binary.Read(data, binary.LittleEndian, &vr.Packet.SeqNo); err != nil {
			return err
		}
		vr.Packet.Data = make([]byte, data.Len())
		if n, err := data.Read(vr.Packet.Data); err != io.EOF && err != nil {
			return err
		} else {
			vr.Packet.Data = vr.Packet.Data[:n]
		}
	} else if status == 0x00 {
		if err := binary.Read(data, binary.LittleEndian, &vr.LastAckedSeqNo); err != nil {
			return err
		}
	}

	return nil
}
