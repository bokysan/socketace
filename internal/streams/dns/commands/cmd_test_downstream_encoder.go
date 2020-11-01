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
	hostname := vr.Command().String() // Always start with the command ID
	if rnd, err := randomChars(); err != nil {
		return "", err
	} else {
		hostname += rnd
	}
	hostname += string(vr.DownstreamEncoder.Code())
	return hostname, nil
}

func (vr *TestDownstreamEncoderRequest) Decode(e enc.Encoder, req string) error {
	// Verify the request is of proper command
	if err := vr.Command().ValidateType(req); err != nil {
		return err
	}
	data := req[4:]

	var err error
	vr.DownstreamEncoder, err = enc.FromCode(data[0])
	return err
}

type TestDownstreamEncoderResponse struct {
	Data []byte // []byte(util.DownloadCodecCheck)
}

func (vr *TestDownstreamEncoderResponse) Command() Command {
	return CmdTestDownstreamEncoder
}

func (vr *TestDownstreamEncoderResponse) Encode(e enc.Encoder) (string, error) {
	return vr.Command().String() + e.Encode(vr.Data), nil
}

func (vr *TestDownstreamEncoderResponse) Decode(e enc.Encoder, request string) error {
	if request == "" {
		return errors.Errorf("Empty string for decoding!")
	}

	if err := vr.Command().ValidateType(request); err != nil {
		return err
	}
	data, err := e.Decode(request[1:])
	vr.Data = data
	return err
}
