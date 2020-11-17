package commands

import (
	"bytes"
	"encoding/binary"
	"github.com/bokysan/socketace/v2/internal/util/enc"
	"github.com/pkg/errors"
	"io"
)

var CmdTestDownstreamFragmentSize = Command{
	Code:        'r',
	NeedsUserId: true,
	NewRequest: func() Request {
		return &TestDownstreamFragmentSizeRequest{}
	},
	NewResponse: func() Response {
		return &TestDownstreamFragmentSizeResponse{}
	},
}

type TestDownstreamFragmentSizeRequest struct {
	UserId       uint16
	FragmentSize uint32
}

func (vr *TestDownstreamFragmentSizeRequest) Command() Command {
	return CmdTestDownstreamFragmentSize
}

func (vr *TestDownstreamFragmentSizeRequest) Encode(e enc.Encoder) ([]byte, error) {
	hostname := EncodeRequestHeader(vr.Command(), vr.UserId)
	data := &bytes.Buffer{}
	if err := binary.Write(data, binary.LittleEndian, &vr.FragmentSize); err != nil {
		return nil, err
	}
	return append(hostname, enc.Base32Encoding.Encode(data.Bytes())...), nil
}

func (vr *TestDownstreamFragmentSizeRequest) Decode(e enc.Encoder, req []byte) error {
	// Verify the request is of proper command
	if rem, userId, err := DecodeRequestHeader(vr.Command(), req); err != nil {
		return err
	} else {
		req = rem
		vr.UserId = userId
	}

	var err error
	data, err := enc.Base32Encoding.Decode(req)
	if err != nil {
		return err
	}

	b := bytes.NewBuffer(data)
	err = binary.Read(b, binary.LittleEndian, &vr.FragmentSize)
	if err != nil {
		return err
	}

	return nil
}

type TestDownstreamFragmentSizeResponse struct {
	FragmentSize uint32
	Data         []byte
	Err          error
}

func (vr *TestDownstreamFragmentSizeResponse) Command() Command {
	return CmdTestDownstreamFragmentSize
}

func (vr *TestDownstreamFragmentSizeResponse) Encode(e enc.Encoder) ([]byte, error) {
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
		if err := binary.Write(data, binary.LittleEndian, &vr.FragmentSize); err != nil {
			return nil, err
		}
		if _, err := data.Write(vr.Data); err != nil {
			return nil, err
		}
	}

	return append([]byte{vr.Command().Code}, e.Encode(data.Bytes())...), nil
}

func (vr *TestDownstreamFragmentSizeResponse) Decode(e enc.Encoder, response []byte) error {
	if response == nil || len(response) == 0 {
		return errors.Errorf("Empty string for decoding!")
	}

	if err := vr.Command().ValidateType(response); err != nil {
		return err
	}

	val, err := e.Decode(response[1:])
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
	} else {
		err = binary.Read(data, binary.LittleEndian, &vr.FragmentSize)
		if err != nil {
			return err
		}
		vr.Data = make([]byte, len(val))
		n, err := data.Read(vr.Data)
		if err != nil && err != io.EOF {
			return err
		}
		vr.Data = vr.Data[:n]
	}
	return nil
}
