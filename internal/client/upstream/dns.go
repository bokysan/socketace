package upstream

import (
	"github.com/bokysan/socketace/v2/internal/streams"
	"github.com/bokysan/socketace/v2/internal/util/cert"
	"github.com/pkg/errors"
	"net"
)

// Socket connects to the server via a socket connection
type Dns struct {
	Packet
}

func (ups *Dns) Connect(manager cert.TlsConfig, mustSecure bool) error {

	a := ups.Address
	switch a.Scheme {
	case "dns":
		a.Scheme = "udp"
	case "dns+unix", "dns+unixgram":
		a.Scheme = "unixgram"
	default:
		return errors.Errorf("This impementation does not know how to handle %s", ups.Address.String())
	}

	n, err := a.Addr()
	if err != nil {
		return errors.WithStack(err)
	}

	if conn, err := net.ListenPacket(n.Network(), n.String()); err != nil {
		return errors.WithStack(err)
	} else {
		ups.PacketConnection = streams.NewDnsClientPacketConnection(conn, ups.Address.String())
	}

	return ups.Packet.Connect(manager, mustSecure)
}
