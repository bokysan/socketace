package commands

import (
	"bytes"
	"encoding/binary"
	"github.com/bokysan/socketace/v2/internal/util/enc"
	"github.com/pkg/errors"
	"io"
)

var CmdSetOptions = Command{
	Code:        'o',
	NeedsUserId: true,
	NewRequest: func() Request {
		return &SetOptionsRequest{}
	},
	NewResponse: func() Response {
		return &SetOptionsResponse{}
	},
}

type SetOptionsRequest struct {
	UserId                 uint16
	LazyMode               *bool
	MultiQuery             *bool
	Closed                 *bool
	DownstreamEncoder      enc.Encoder
	UpstreamEncoder        enc.Encoder
	DownstreamFragmentSize *uint32
}

func (vr *SetOptionsRequest) Command() Command {
	return CmdSetOptions
}

func (vr *SetOptionsRequest) writeBool(data *bytes.Buffer, b *bool) error {
	val := byte(255)
	if b != nil {
		if *b {
			val = 1
		} else {
			val = 0
		}
	}
	return data.WriteByte(val)
}

func (vr *SetOptionsRequest) readBool(data *bytes.Buffer) (b *bool, err error) {
	val, err := data.ReadByte()
	if err != nil {
		return nil, err
	}
	if val == 1 {
		var t bool
		t = true
		b = &t
	} else if val == 0 {
		var f bool
		f = false
		b = &f
	}

	return
}

func (vr *SetOptionsRequest) Encode(e enc.Encoder) ([]byte, error) {
	hostname := EncodeRequestHeader(vr.Command(), vr.UserId)

	data := &bytes.Buffer{}
	if err := vr.writeBool(data, vr.LazyMode); err != nil {
		return nil, err
	}
	if err := vr.writeBool(data, vr.MultiQuery); err != nil {
		return nil, err
	}
	if err := vr.writeBool(data, vr.Closed); err != nil {
		return nil, err
	}
	downstreamCode := byte(' ')
	if vr.DownstreamEncoder != nil {
		downstreamCode = vr.DownstreamEncoder.Code()
	}
	if err := data.WriteByte(downstreamCode); err != nil {
		return nil, err
	}
	upstreamCode := byte(' ')
	if vr.UpstreamEncoder != nil {
		upstreamCode = vr.UpstreamEncoder.Code()
	}
	if err := data.WriteByte(upstreamCode); err != nil {
		return nil, err
	}

	fragmentSize := uint32(0xFFFFFFFF)
	if vr.DownstreamFragmentSize != nil {
		fragmentSize = *vr.DownstreamFragmentSize
	}
	if err := binary.Write(data, binary.LittleEndian, &fragmentSize); err != nil {
		return nil, err
	}

	hostname = append(hostname, enc.Base32Encoding.Encode(data.Bytes())...)
	return hostname, nil
}

func (vr *SetOptionsRequest) Decode(e enc.Encoder, req []byte) error {
	// Verify the request is of proper command
	if rem, userId, err := DecodeRequestHeader(vr.Command(), req); err != nil {
		return err
	} else {
		req = rem
		vr.UserId = userId
	}

	var data *bytes.Buffer
	if decoded, err := enc.Base32Encoding.Decode(req); err != nil {
		return err
	} else {
		data = bytes.NewBuffer(decoded)
	}

	if b, err := vr.readBool(data); err != nil {
		return nil
	} else {
		vr.LazyMode = b
	}
	if b, err := vr.readBool(data); err != nil {
		return nil
	} else {
		vr.MultiQuery = b
	}
	if b, err := vr.readBool(data); err != nil {
		return nil
	} else {
		vr.Closed = b
	}
	if b, err := data.ReadByte(); err != nil {
		return err
	} else if b != ' ' {
		if e, err := enc.FromCode(b); err != nil {
			return err
		} else {
			vr.DownstreamEncoder = e
		}
	}
	if b, err := data.ReadByte(); err != nil {
		return err
	} else if b != ' ' {
		if e, err := enc.FromCode(b); err != nil {
			return err
		} else {
			vr.UpstreamEncoder = e
		}
	}
	var fragmentSize uint32
	if err := binary.Read(data, binary.LittleEndian, &fragmentSize); err != nil {
		return err
	} else if fragmentSize != uint32(0xFFFFFFFF) {
		vr.DownstreamFragmentSize = &fragmentSize
	}

	return nil
}

type SetOptionsResponse struct {
	Err error
}

func (vr *SetOptionsResponse) Command() Command {
	return CmdSetOptions
}

func (vr *SetOptionsResponse) Encode(e enc.Encoder) ([]byte, error) {
	data := &bytes.Buffer{}
	if vr.Err != nil {
		if err := data.WriteByte(255); err != nil {
			return nil, err
		}
		if _, err := data.WriteString(vr.Err.Error()); err != nil {
			return nil, err
		}
	} else {
		if err := data.WriteByte(0); err != nil {
			return nil, err
		}
	}

	return append([]byte{vr.Command().Code}, enc.Base32Encoding.Encode(data.Bytes())...), nil
}

func (vr *SetOptionsResponse) Decode(e enc.Encoder, response []byte) error {
	if response == nil || len(response) == 0 {
		return errors.Errorf("Empty string for decoding!")
	}

	if err := vr.Command().ValidateType(response); err != nil {
		return err
	}

	val, err := enc.Base32Encoding.Decode(response[1:])
	if err != nil {
		return errors.WithStack(err)
	}

	data := bytes.NewBuffer(val)
	status, err := data.ReadByte()
	if err != nil {
		return errors.WithStack(err)
	}
	if status&1 != 0 {
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
	}
	return nil
}
