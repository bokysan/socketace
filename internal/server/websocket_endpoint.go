package server

import "fmt"

type WebsocketEndpoint struct {
	Channels          []string `json:"channels"`
	Endpoint          string   `json:"endpoint"`
	EnableCompression bool     `json:"enableCompression"`
}

func (wsm *WebsocketEndpoint) String() string {
	return fmt.Sprintf("%s:%s", wsm.Channels, wsm.Endpoint)
}

// ------ // ------ // ------ // ------ // ------ // ------ // ------ //

type WebsocketEndpointList []WebsocketEndpoint

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
