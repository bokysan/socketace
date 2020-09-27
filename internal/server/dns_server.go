package server

import (
	"github.com/bokysan/socketace/v2/internal/cert"
	"net/http"
)

type DnsServer struct {
	cert.Manager
	cert.ClientAuthentication

	Kind      string                `json:"kind"`
	Listen    string                `json:"listen"`
	Endpoints WebsocketEndpointList `json:"endpoints"`

	service       *Service
	server        *http.Server
	couldNotStart chan struct{}
}

func NewDnsServer() *DnsServer {
	return &DnsServer{
		couldNotStart: make(chan struct{}),
	}
}
