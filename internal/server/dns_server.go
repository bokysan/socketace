package server

import (
	"crypto/tls"
	"fmt"
	"github.com/bokysan/socketace/v2/internal/streams"
	"github.com/bokysan/socketace/v2/internal/streams/dns"
	dns2 "github.com/miekg/dns"
	"github.com/pkg/errors"
	"strings"
)

type DnsServer struct {
	SocketServer
}

func NewDnsServer() *DnsServer {
	return &DnsServer{
		SocketServer{
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
	case "dns":
		a.Scheme = "udp"
	case "dns+tcp":
		a.Scheme = "tcp"
	case "dns+unix", "dns+unixgram":
		a.Scheme = "unixgram"
	default:
		return errors.Errorf("This implementation does not know how to handle %s", st.Address.String())
	}

	var comm *dns.NetConnectionServerCommunicator
	server := &dns2.Server{
		Addr: "127.0.0.1:42000",
		Net:  "udp",
	}
	if true {
		return errors.Errorf("Configuration not complete!")
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

	conn := dns.NewServerDnsListener(a.Host, comm)

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
