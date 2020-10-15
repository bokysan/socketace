package server

import (
	"encoding/json"
	"fmt"
	"github.com/bokysan/socketace/v2/internal/util/addr"
	"github.com/pkg/errors"
)

type Servers []Server

type Server interface {
	fmt.Stringer

	Startup(channels Channels) error
	Shutdown() error
}

func (se *Servers) UnmarshalFlag(value string) error {
	// Unmarshall from command line
	return se.UnmarshalJSON([]byte(value))
}

func (se *Servers) UnmarshalYAML(unmarshal func(interface{}) error) error {
	stuff := make([]interface{}, 0)
	if err := unmarshal(&stuff); err != nil {
		return errors.WithStack(err)
	}

	res := make(Servers, 0)
	for _, s := range stuff {
		if server, err := unmarshalServer(s); err != nil {
			return errors.WithStack(err)
		} else {
			res = append(res, server)
		}
	}

	*se = res
	return nil
}

func (se *Servers) UnmarshalJSON(b []byte) error {
	stuff := make([]interface{}, 0)
	if err := json.Unmarshal(b, &stuff); err != nil {
		return errors.WithStack(err)
	}

	res := make(Servers, 0)
	for _, s := range stuff {
		if server, err := unmarshalServer(s); err != nil {
			return errors.WithStack(err)
		} else {
			res = append(res, server)
		}
	}

	*se = res
	return nil
}

func unmarshalServer(s interface{}) (Server, error) {
	stuff, ok := s.(map[string]interface{})
	if !ok {
		return nil, errors.Errorf("Invalid type. Expected map[string]interface{}, got: %+v", stuff)
	}

	if val, ok := stuff["address"]; ok {
		if a, ok := val.(string); ok {
			address, err := addr.ParseAddress(a)
			if err != nil {
				return nil, errors.Wrapf(err, "Failed parsing address '%v'", a)
			}

			var server Server

			switch address.Scheme {
			case "http", "https", "ws", "wss", "http+tls", "ws+tls":
				server = NewHttpServer()
			case "stdin", "stdin+tls", "stdio", "stdio+tls":
				server = NewIoServer()
			case "tcp", "unix", "unixpacket", "tcp+tls", "unix+tls", "unixpacket+tls":
				server = NewSocketServer()
			case "upd", "udp4", "upd6", "unixgram":
				server = NewPacketServer()
			default:
				return nil, errors.Errorf("Unknown network type: %s", address.Scheme)
			}

			data, err := json.Marshal(s)
			if err != nil {
				return nil, errors.Errorf("Failed marshalling data: %v", s)
			}

			err = json.Unmarshal(data, server)
			if err != nil {
				return nil, errors.Wrapf(err, "Failed unmarshalling data: %v", string(data))
			}

			return server, nil

		} else {
			return nil, errors.Errorf("'kind' is not a string: %+v", val)
		}
	}

	return nil, errors.Errorf("Missing server type!")

}
