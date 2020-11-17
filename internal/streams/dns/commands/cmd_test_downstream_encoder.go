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

func (vr *TestDownstreamEncoderRequest) Encode(e enc.Encoder) ([]byte, error) {
	hostname := EncodeRequestHeader(vr.Command(), 0)
	hostname = append(hostname, vr.DownstreamEncoder.Code())
	return hostname, nil
}

func (vr *TestDownstreamEncoderRequest) Decode(e enc.Encoder, req []byte) error {
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

func (vr *TestDownstreamEncoderResponse) Encode(e enc.Encoder) ([]byte, error) {
	if vr.Err != nil {
		data := make([]byte, 0)
		data = append(data, vr.Command().Code)
		data = append(data, 'e')
		data = append(data, enc.Base32Encoding.Encode([]byte(vr.Err.Error()))...)
		return data, nil
	} else {
		data := make([]byte, 0)
		data = append(data, vr.Command().Code)
		data = append(data, 'o')
		data = append(data, e.Encode(vr.Data)...)
		return data, nil
	}
}

func (vr *TestDownstreamEncoderResponse) Decode(e enc.Encoder, response []byte) error {
	if response == nil || len(response) == 0 {
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
