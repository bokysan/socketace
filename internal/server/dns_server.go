package server

import (
	"fmt"
	"github.com/bokysan/socketace/v2/internal/streams"
	"github.com/bokysan/socketace/v2/internal/streams/dns"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"net"
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

	a := st.Address
	switch a.Scheme {
	case "dns":
		a.Scheme = "udp"
	case "dns+unix", "dns+unixgram":
		a.Scheme = "unixgram"
	default:
		return errors.Errorf("This impementation does not know how to handle %s", st.Address.String())
	}

	log.Infof("Starting UDP socket server at %s", st.String())
	_, err := net.ListenPacket(a.Scheme, a.Host)
	if err != nil {
		return errors.WithStack(err)
	}

	conn, err := dns.NewServerDnsListener("", nil)
	if err != nil {
		return errors.WithStack(err)
	}

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
