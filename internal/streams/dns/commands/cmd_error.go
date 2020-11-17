package commands

import (
	"bytes"
	"github.com/bokysan/socketace/v2/internal/util/enc"
	"github.com/pkg/errors"
	"io"
)

var CmdError = Command{
	Code: 'e',
	NewResponse: func() Response {
		return &ErrorResponse{}
	},
}

type ErrorResponse struct {
	Err error
}

func (vr *ErrorResponse) Command() Command {
	return CmdError
}

func (vr *ErrorResponse) Encode(e enc.Encoder) ([]byte, error) {
	data := &bytes.Buffer{}
	if _, err := data.WriteString(vr.Err.Error()); err != nil {
		return nil, err
	}

	return append([]byte{vr.Command().Code}, enc.Base32Encoding.Encode(data.Bytes())...), nil
}

func (vr *ErrorResponse) Decode(e enc.Encoder, response []byte) error {
	if response == nil || len(response) == 0 {
		return errors.Errorf("Empty string for decoding!")
	}

	if err := vr.Command().ValidateType(response); err != nil {
		return err
	}

	response = response[1:]

	val, err := enc.Base32Encoding.Decode(response)
	if err != nil {
		return errors.WithStack(err)
	}
	data := bytes.NewBuffer(val)
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
}
