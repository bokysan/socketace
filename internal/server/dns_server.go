package server

import (
	"crypto/tls"
	"fmt"
	"github.com/bokysan/socketace/v2/internal/streams"
	"github.com/bokysan/socketace/v2/internal/streams/dns"
	dns2 "github.com/miekg/dns"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"strings"
)

type DnsServer struct {
	SocketServer
	Domain string `json:"domain"`
}

func NewDnsServer() *DnsServer {
	return &DnsServer{
		SocketServer: SocketServer{
			name: "dns",
		},
	}
}

func (st *DnsServer) String() string {
	return fmt.Sprintf("%s", st.Address.String())
}

func (st *DnsServer) Startup(channels Channels) error {
	if upstreams, err := channels.Filter(st.Channels); err != nil {
		return errors.WithStack(err)
	} else {
		st.upstreams = upstreams
	}

	if strings.HasSuffix(st.Address.Scheme, "+tls") {
		st.Address.Scheme = st.Address.Scheme[:len(st.Address.Scheme)-4]
		st.secure = true
	} else {
		st.secure = false
	}

	a := st.Address
	switch a.Scheme {
	case "dns", "dns+udp":
		a.Scheme = "udp"
	case "dns+tcp":
		a.Scheme = "tcp"
	default:
		return errors.Errorf("DNS server can only handle 'dns', 'dns+udp', 'dns+tcp' schemes, not %q", st.Address.String())
	}

	var comm *dns.NetConnectionServerCommunicator
	server := &dns2.Server{
		Addr: a.Host,
		Net:  a.Scheme,
	}

	if st.secure {
		var err error
		var tlsConfig *tls.Config
		if tlsConfig, err = st.ServerConfig.GetTlsConfig(); err != nil {
			return errors.Wrapf(err, "Could not configure TLS")
		}
		server.Net = server.Net + "-tls"
		server.TLSConfig = tlsConfig
	}

	var err error
	comm, err = dns.NewNetConnectionServerCommunicator(server)
	if err != nil {
		return errors.Wrapf(err, "Could not start DNS listener")
	}

	log.Infof("Starting DNS server at %v, listening to requests for '%v'", st.String(), st.Domain)
	conn := dns.NewServerDnsListener(st.Domain, comm)
	st.listener = conn

	go func() {
		st.acceptConnection()
	}()

	return nil
}

func (st *DnsServer) Shutdown() error {
	st.done = true
	return streams.LogClose(st.listener)
}
