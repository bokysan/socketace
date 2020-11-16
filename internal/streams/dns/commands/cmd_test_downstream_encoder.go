package commands

import (
	"github.com/bokysan/socketace/v2/internal/util/enc"
	"github.com/pkg/errors"
)

var CmdTestDownstreamEncoder = Command{
	Code: 'y',
	NewRequest: func() Request {
		return &TestDownstreamEncoderRequest{}
	},
	NewResponse: func() Response {
		return &TestDownstreamEncoderResponse{}
	},
}

type TestDownstreamEncoderRequest struct {
	DownstreamEncoder enc.Encoder
}

func (vr *TestDownstreamEncoderRequest) Command() Command {
	return CmdTestDownstreamEncoder
}

func (vr *TestDownstreamEncoderRequest) Encode(e enc.Encoder) (string, error) {
	hostname := EncodeRequestHeader(vr.Command(), 0)
	hostname += string(vr.DownstreamEncoder.Code())
	return hostname, nil
}

func (vr *TestDownstreamEncoderRequest) Decode(e enc.Encoder, req string) error {
	// Verify the request is of proper command
	if rem, _, err := DecodeRequestHeader(vr.Command(), req); err != nil {
		return err
	} else {
		req = rem
	}

	var err error
	vr.DownstreamEncoder, err = enc.FromCode(req[0])
	if err != nil {
		return err
	}
	return nil
}

type TestDownstreamEncoderResponse struct {
	Data []byte // []byte(util.DownloadCodecCheck)
	Err  error
}

func (vr *TestDownstreamEncoderResponse) Command() Command {
	return CmdTestDownstreamEncoder
}

func (vr *TestDownstreamEncoderResponse) Encode(e enc.Encoder) (string, error) {
	if vr.Err != nil {
		return vr.Command().String() + "e" + enc.Base32Encoding.Encode([]byte(vr.Err.Error())), nil
	} else {
		return vr.Command().String() + "o" + e.Encode(vr.Data), nil
	}
}

func (vr *TestDownstreamEncoderResponse) Decode(e enc.Encoder, response string) error {
	if response == "" {
		return errors.Errorf("Empty string for decoding!")
	}

	if err := vr.Command().ValidateType(response); err != nil {
		return err
	}

	if len(response) > 1 {
		if response[1] == 'e' {
			d, err := enc.Base32Encoding.Decode(response[2:])
			str := string(d)
			if err != nil {
				return err
			}
			for _, e := range BadErrors {
				if e.Error() == str {
					vr.Err = e
					return nil
				}
			}
			vr.Err = errors.New(str)
			return nil
		} else if response[1] == 'o' {
			data, err := e.Decode(response[2:])
			vr.Data = data
			return err
		} else {
			return errors.Errorf("Invalid response: %v", response[1:])
		}
	} else {
		return errors.Errorf("No data in donwstream encoder response: %q", response)
	}
}
