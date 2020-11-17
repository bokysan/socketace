package commands

import (
	"bytes"
	"github.com/bokysan/socketace/v2/internal/util/enc"
	"github.com/pkg/errors"
	"io"
)

var CmdTestUpstreamEncoder = Command{
	Code:        'z',
	NeedsUserId: true,
	NewRequest: func() Request {
		return &TestUpstreamEncoderRequest{}
	},
	NewResponse: func() Response {
		return &TestUpstreamEncoderResponse{}
	},
}

type TestUpstreamEncoderRequest struct {
	UserId  uint16
	Pattern []byte
}

func (vr *TestUpstreamEncoderRequest) Command() Command {
	return CmdTestUpstreamEncoder
}

// Encode does not really encode, as the main point is to test if the charset goes through or not
func (vr *TestUpstreamEncoderRequest) Encode(e enc.Encoder) ([]byte, error) {
	hostname := EncodeRequestHeader(vr.Command(), vr.UserId)
	hostname = append(hostname, vr.Pattern...)
	return hostname, nil
}

// Decode does not really decode, as the main point is to test if the charset goes through or not
func (vr *TestUpstreamEncoderRequest) Decode(e enc.Encoder, req []byte) error {
	// Verify the request is of proper command
	if rem, userId, err := DecodeRequestHeader(vr.Command(), req); err != nil {
		return err
	} else {
		req = rem
		vr.UserId = userId
	}
	vr.Pattern = []byte(req)
	return nil
}

type TestUpstreamEncoderResponse struct {
	Data []byte
	Err  error
}

func (vr *TestUpstreamEncoderResponse) Command() Command {
	return CmdTestUpstreamEncoder
}

// Encode happens before downstream encoder is selected, so encode with Base32 always
func (vr *TestUpstreamEncoderResponse) Encode(e enc.Encoder) ([]byte, error) {
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
		if _, err := data.Write([]byte(vr.Data)); err != nil {
			return nil, err
		}
	}

	return append([]byte{vr.Command().Code}, enc.Base32Encoding.Encode(data.Bytes())...), nil
}

// Decode happens before downstream encoder is selected, so encode with Base32 always
func (vr *TestUpstreamEncoderResponse) Decode(e enc.Encoder, response []byte) error {
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
	} else {
		d := make([]byte, len(val))
		cnt, err := data.Read(d)
		if err != nil && err != io.EOF {
			return errors.WithStack(err)
		}
		vr.Data = d[0:cnt]
	}
	return nil
}
