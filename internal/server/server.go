package server

import (
	"encoding/json"
	"github.com/pkg/errors"
)

type ServerList []Server

type Server interface {
	SetService(host *Service)
	Execute(args []string) error
	Shutdown() error
}

func NewServer(kind string) (Server, error) {
	switch kind {
	case "websocket":
		return NewWebsocketServer(), nil
	case "stdin":
		return NewStdinServer(), nil
	case "socket":
		return NewSocketServer(), nil
	}

	return nil, errors.Errorf("Unknown server type: %s", kind)
}

func (se *ServerList) UnmarshalFlag(value string) error {
	// Unmarshall from command line
	return se.UnmarshalJSON([]byte(value))
}

func (se *ServerList) UnmarshalYAML(unmarshal func(interface{}) error) error {
	stuff := make([]interface{}, 0)
	if err := unmarshal(&stuff); err != nil {
		return errors.WithStack(err)
	}

	res := make(ServerList, 0)
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

func (se *ServerList) UnmarshalJSON(b []byte) error {
	stuff := make([]interface{}, 0)
	if err := json.Unmarshal(b, &stuff); err != nil {
		return errors.WithStack(err)
	}

	res := make(ServerList, 0)
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

	if val, ok := stuff["kind"]; ok {
		if kind, ok := val.(string); ok {
			server, err := NewServer(kind)
			if err != nil {
				return nil, errors.Wrapf(err,"Failed creating a server of kind: %s", kind)
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
