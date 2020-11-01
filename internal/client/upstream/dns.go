package upstream

import (
	"github.com/bokysan/socketace/v2/internal/socketace"
	"github.com/bokysan/socketace/v2/internal/streams"
	"github.com/bokysan/socketace/v2/internal/streams/dns"
	"github.com/bokysan/socketace/v2/internal/util/addr"
	"github.com/bokysan/socketace/v2/internal/util/cert"
	dns2 "github.com/miekg/dns"
	"github.com/pkg/errors"
)

// Socket connects to the server via a socket connection
type Dns struct {
	streams.Connection

	// Address is the parsed representation of the address and calculated automatically while unmarshalling
	Address addr.ProtoAddress
}

func (ups *Dns) String() string {
	return ups.Address.String()
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

	topDomain := a.Hostname()

	var dnsList []string

	if d, ok := a.Query()["dns"]; ok {
		dnsList = d
	}

	conf := &dns2.ClientConfig{
		Servers: dnsList,
	}
	comm, err := dns.NewNetConnectionClientCommunicator(conf)
	if err != nil {
		return err
	}

	conn, err := dns.NewClientDnsConnection(topDomain, comm)
	if err != nil {
		return err
	}

	if err = conn.Handshake(); err != nil {
		return err
	}

	cc, err := socketace.NewClientConnection(conn, manager, false, ups.Address.Host)
	if err != nil {
		return errors.Wrapf(err, "Could not open connection")
	} else if mustSecure && !cc.Secure() {
		return errors.Errorf("Could not establish a secure connection to %v", ups.Address)
	}

	ups.Connection = streams.NewNamedConnection(streams.NewNamedConnection(cc, ups.Address.String()), "dns")

	return nil
}
