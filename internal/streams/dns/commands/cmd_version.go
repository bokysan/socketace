package commands

import (
	"bytes"
	"encoding/binary"
	"github.com/bokysan/socketace/v2/internal/util/enc"
	"github.com/pkg/errors"
	"io"
	"strconv"
)

var CmdVersion = Command{
	Code:        'v',
	NeedsUserId: false,
	NewRequest: func() Request {
		return &VersionRequest{}
	},
	NewResponse: func() Response {
		return &VersionResponse{}
	},
}

type VersionRequest struct {
	ClientVersion uint32
}

func (vr *VersionRequest) Command() Command {
	return CmdVersion
}

func (vr *VersionRequest) Encode(e enc.Encoder) ([]byte, error) {
	hostname := EncodeRequestHeader(vr.Command(), 0)

	data := &bytes.Buffer{}
	if err := binary.Write(data, binary.LittleEndian, vr.ClientVersion); err != nil {
		return nil, err
	}
	return append(hostname, enc.Base32Encoding.Encode(data.Bytes())...), nil
}

func (vr *VersionRequest) Decode(e enc.Encoder, req []byte) error {
	// Verify the request is of proper command
	if rem, _, err := DecodeRequestHeader(vr.Command(), req); err != nil {
		return err
	} else {
		req = rem
	}

	decode, err := enc.Base32Encoding.Decode(req)
	if err != nil {
		return err
	}
	buf := bytes.NewBuffer(decode) // Strip command type and 3 random characters
	return binary.Read(buf, binary.LittleEndian, &vr.ClientVersion)
}

type VersionResponse struct {
	ServerVersion uint32
	UserId        uint16
	Err           error
}

func (vr *VersionResponse) Command() Command {
	return CmdVersion
}

func (vr *VersionResponse) Encode(e enc.Encoder) ([]byte, error) {
	data := &bytes.Buffer{}
	if err := binary.Write(data, binary.LittleEndian, vr.ServerVersion); err != nil {
		return nil, err
	}
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

	res := make([]byte, 0)
	res = append(res, vr.Command().Code)
	res = append(res, []byte(EncodeUserId(vr.UserId))...)
	res = append(res, enc.Base32Encoding.Encode(data.Bytes())...)
	return res, nil
}

func (vr *VersionResponse) Decode(e enc.Encoder, response []byte) error {
	if response == nil || len(response) == 0 {
		return errors.Errorf("Empty string for decoding!")
	}

	if err := vr.Command().ValidateType(response); err != nil {
		return err
	}

	response = response[1:]

	u, err := strconv.ParseInt(string(response[0:2]), 36, 16)
	if err != nil {
		return err
	} else {
		vr.UserId = uint16(u)
	}

	response = response[2:]

	val, err := enc.Base32Encoding.Decode(response)
	if err != nil {
		return errors.WithStack(err)
	}
	data := bytes.NewBuffer(val)
	if err := binary.Read(data, binary.LittleEndian, &vr.ServerVersion); err != nil {
		return errors.WithStack(err)
	}
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
