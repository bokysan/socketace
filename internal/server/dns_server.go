package server

import (
	"github.com/bokysan/socketace/v2/internal/util/cert"
	"net/http"
)

type DnsServer struct {
	cert.ServerConfig

	Kind   string `json:"kind"`
	Listen string `json:"listen"`

	server        *http.Server
	couldNotStart chan struct{}
}

func NewDnsServer() *DnsServer {
	return &DnsServer{
		couldNotStart: make(chan struct{}),
	}
}
