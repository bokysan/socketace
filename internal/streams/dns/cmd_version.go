package dns

import (
	"bytes"
	"encoding/binary"
	"github.com/bokysan/socketace/v2/internal/util/enc"
	"github.com/miekg/dns"
	"github.com/pkg/errors"
	"golang.org/x/net/dns/dnsmessage"
	"io"
)

const CmdVersion Command = 'v'

type VersionRequest struct {
	ClientVersion uint32
}

func (vr *VersionRequest) Command() Command {
	return CmdVersion
}

func (vr *VersionRequest) Encode(e enc.Encoder, domain string) (string, error) {
	data := &bytes.Buffer{}
	if err := binary.Write(data, binary.LittleEndian, vr.ClientVersion); err != nil {
		return "", err
	}
	encoded := Base32Encoding.Encode(data.Bytes())

	hostname := string(vr.Command()) // Always start with the command ID
	if rnd, err := randomChars(); err != nil {
		return "", err
	} else {
		hostname += rnd
	}
	hostname += encoded
	return prepareHostname(hostname, domain)
}

func (vr *VersionRequest) Decode(e enc.Encoder, req, domain string) error {
	// Verify the request is of proper command
	if err := vr.Command().ValidateType(req); err != nil {
		return err
	}
	data := stripDomain(req, domain)[4:]
	decode, err := Base32Encoding.Decode(data)
	if err != nil {
		return err
	}
	buf := bytes.NewBuffer(decode) // Strip command type and 3 random characters
	return binary.Read(buf, binary.LittleEndian, &vr.ClientVersion)
}

type VersionResponse struct {
	ServerVersion uint32
	UserId        byte
	Err           error
}

func (vr *VersionResponse) Command() Command {
	return CmdVersion
}

func (vr *VersionResponse) Encode(e enc.Encoder, queryType dnsmessage.Type) (*dns.Msg, error) {
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
		if err := data.WriteByte(255); err != nil {
			return nil, err
		}
		if err := data.WriteByte(vr.UserId); err != nil {
			return nil, err
		}
	}

	val := append([]byte{byte(vr.Command())}, []byte(Base32Encoding.Encode(data.Bytes()))...)
	return EncodeDnsResponse(val, queryType)
}

func (vr *VersionResponse) Decode(e enc.Encoder, msg *dns.Msg) error {
	request := DecodeDnsResponse(msg)

	if err := vr.Command().ValidateType(request); err != nil {
		return err
	}

	val, err := Base32Encoding.Decode(request[1:])
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
				vr.Err = &e
				return nil
			}
		}
		vr.Err = errors.New(str)
	} else {
		vr.UserId, err = data.ReadByte()
		if err != nil {
			return errors.WithStack(err)
		}
	}
	return nil
}
