package server

import "fmt"

type HttpEndpoint struct {
	Channels          []string `json:"channels"`
	Endpoint          string   `json:"endpoint"`
	EnableCompression bool     `json:"enableCompression"`
}

func (wsm *HttpEndpoint) String() string {
	return fmt.Sprintf("%s:%s", wsm.Channels, wsm.Endpoint)
}

// ------ // ------ // ------ // ------ // ------ // ------ // ------ //

type WebsocketEndpointList []HttpEndpoint

func (epl *WebsocketEndpointList) String() string {
	s := ""
	for _, ep := range *epl {
		if s != "" {
			s = s + ","
		}
		s = s + ep.String()
	}

	return s
}
