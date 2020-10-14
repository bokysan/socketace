package server

import (
	"fmt"
	"github.com/bokysan/socketace/v2/internal/util/addr"
	"github.com/bokysan/socketace/v2/internal/util/cert"
	"net/http"
)

type DnsServer struct {
	cert.ServerConfig

	Address *addr.ProtoAddress `json:"address"`
	Listen  string             `json:"listen"`

	secure        bool
	server        *http.Server
	couldNotStart chan struct{}
}

func (ds *DnsServer) String() string {
	var addr addr.ProtoAddress
	addr = *ds.Address
	if ds.secure {
		addr.Scheme = addr.Scheme + "+tls"
	}

	return fmt.Sprintf("%s", addr.String())
}
func NewDnsServer() *DnsServer {
	return &DnsServer{
		couldNotStart: make(chan struct{}),
	}
}
