package server

import (
	"encoding/json"
	"github.com/pkg/errors"
)

type Servers []Server

type Server interface {
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

	if val, ok := stuff["network"]; ok {
		if network, ok := val.(string); ok {
			var server Server

			switch network {
			case "http", "https", "ws", "wss", "http+tls", "ws+tls":
				server = NewHttpServer()
			case "stdin", "stdin+tls":
				server = NewIoServer()
			case "tcp", "unix", "unixpacket", "tcp+tls", "unix+tls", "unixpacket+tls":
				server = NewSocketServer()
			default:
				return nil, errors.Errorf("Unknown server type: %s", network)
			}

			data, err := json.Marshal(s)
			if err != nil {
				return nil, errors.Errorf("Failed marshalling data: %v", s)
			}

			err = json.Unmarshal(data, server)
			if err != nil {
				return nil, errors.Errorf("Failed unmarshalling data: %v", data)
			}

			return server, nil

		} else {
			return nil, errors.Errorf("'kind' is not a string: %+v", val)
		}
	}

	return nil, errors.Errorf("Missing server type!")

}
