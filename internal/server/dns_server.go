package server

import (
	"fmt"
	"github.com/bokysan/socketace/v2/internal/streams"
	"github.com/pkg/errors"
	"net"
)

type DnsServer struct {
	PacketServer
}

func NewDnsServer() *DnsServer {
	return &DnsServer{}
}

func (st *DnsServer) String() string {
	return fmt.Sprintf("%s", st.Address.String())
}

func (st *DnsServer) Startup(channels Channels) error {
	a := st.Address
	switch a.Scheme {
	case "dns":
		a.Scheme = "udp"
	case "dns+unix", "dns+unixgram":
		a.Scheme = "unixgram"
	default:
		return errors.Errorf("This impementation does not know how to handle %s", st.Address.String())
	}

	n, err := a.Addr()
	if err != nil {
		return errors.WithStack(err)
	}

	if conn, err := net.ListenPacket(n.Network(), n.String()); err != nil {
		return errors.WithStack(err)
	} else {
		st.PacketConnection = streams.NewDnsServerPacketConnection(conn, st.Address.String())
	}

	return st.PacketServer.Startup(channels)
}

func (st *DnsServer) Shutdown() error {
	st.done = true
	return streams.LogClose(st.listener)
}
