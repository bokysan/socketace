package dns

import (
	"github.com/bokysan/socketace/v2/internal/streams"
	"github.com/miekg/dns"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"io"
	"net"
	"strings"
	"time"
)

// ClientCommunicator is an anbstract interface which executes a communication with the upstream DNS server. We
// are using this abstraction to make it possible to do high-level communication with DNS over multiple different
// connections -- e.g. UDP, TCP, pipe or (for testing) directly by sending the DNS messages to the upstream server.
type ClientCommunicator interface {
	io.Closer
	streams.Closed

	// Send a message to DNS and get a response
	SendAndReceive(m *dns.Msg, timeout *time.Duration) (r *dns.Msg, rtt time.Duration, err error)

	// LocalAddr returns the local network address.
	LocalAddr() net.Addr

	// RemoteAddr returns the remote network address.
	RemoteAddr() net.Addr

	// SetDeadline sets the read and write deadlines associated with the connection.
	SetDeadline(t time.Time) error

	// SetReadDeadline sets the deadline for future Read calls and any currently-blocked Read call.
	// A zero value for t means Read will not time out.
	SetReadDeadline(t time.Time) error

	// SetWriteDeadline sets the deadline for future Write calls and any currently-blocked Write call.
	// A zero value for t means Write will not time out.
	SetWriteDeadline(t time.Time) error
}

type NetConnectionClientCommunicator struct {
	Client *dns.Client
	Conn   *dns.Conn
	closed bool
}

type AddressList []net.Addr

func (l *AddressList) addAddress(network string, address string) {
	addr, err := ResolveNetworkAddress(network, address, "53")
	if err != nil {
		log.Warnf("Cannot resolve %v as a %v address: %v", address, network, err)
	} else {
		found := false
		for _, a := range *l {
			if a.String() == addr.String() && a.Network() == addr.Network() {
				found = true
				break
			}
		}
		if !found {
			log.Debugf("Adding %v to list.", address)
			*l = append(*l, addr)
		}
	}
	return
}

// ResolveAndAddAddress will try to resolve the provide string as a TCP an UDP address. And if any of these succeeed,
// it will add the address to the list
func (l *AddressList) ResolveAndAddAddress(address string) {
	switch {
	case strings.HasPrefix("tcp://", address):
		l.addAddress("tcp", address[6:])
	case strings.HasPrefix("udp://", address):
		l.addAddress("udp", address[6:])
	default:
		l.addAddress("tcp", address)
		l.addAddress("udp", address)
	}
}

// ClientConfig is the configuration for the ClientCommunicator
type ClientConfig struct {
	Servers AddressList
}

// ResolveNetworkAddress will generate an address, optionally adding the specified port if not in the initial string. The
// function will only work for TCP and UDP addresses.
func ResolveNetworkAddress(network, server, defaultPort string) (net.Addr, error) {
	switch network {
	case "udp", "udp4", "udp6":
		if addr, err := net.ResolveUDPAddr(network, server); err == nil {
			if addr.Port != 0 {
				return addr, nil
			}
		} else if addr, err := net.ResolveUDPAddr(network, server+":"+defaultPort); err != nil {
			return nil, err
		} else {
			return addr, nil
		}
	case "tcp", "tcp4", "tcp6":
		if addr, err := net.ResolveTCPAddr(network, server); err == nil {
			if addr.Port != 0 {
				return addr, nil
			}
		} else if addr, err := net.ResolveTCPAddr(network, server+":"+defaultPort); err != nil {
			return nil, err
		} else {
			return addr, nil
		}
	}

	return nil, errors.New("Could not generate an address")
}

func MustResolveNetworkAddress(network, server, defaultPort string) net.Addr {
	addr, err := ResolveNetworkAddress(network, server, defaultPort)
	if err != nil {
		panic(err)
	}
	return addr
}

func NewNetConnectionClientCommunicator(config *ClientConfig) (*NetConnectionClientCommunicator, error) {
	if config == nil {
		return nil, errors.New("Client configuration not provided")
	}

	if len(config.Servers) == 0 {
		return nil, errors.New("You need at least one upstream server!")
	}

	// Try addresses, in order
	var conn net.Conn
	var err error
	for _, addr := range config.Servers {
		// TODO: Add support for TCP DNS
		switch v := addr.(type) {
		case *net.UDPAddr:
			log.Tracef("Dialing UDP %v", addr)
			conn, err = net.DialUDP("udp", nil, v)
		case *net.TCPAddr:
			log.Tracef("Dialing TCP %v", addr)
			conn, err = net.DialTCP("tcp", nil, v)
		default:
			err = errors.Errorf("Don't know how to handle address %v", v)
		}
		if err != nil {
			err = errors.WithStack(err)
		} else {
			log.Infof("Connected to upstream server %v", addr)
			break
		}
	}

	if conn == nil {
		err = errors.Errorf("No connection can be established. None of the servers %v worked.", config.Servers)
	}

	if err != nil {
		return nil, errors.WithStack(err)
	}

	return &NetConnectionClientCommunicator{
		Client: &dns.Client{},
		Conn: &dns.Conn{
			Conn:    conn,
			UDPSize: 65535,
		},
	}, nil
}

func (sc *NetConnectionClientCommunicator) Close() error {
	if sc.closed {
		return nil
	}
	sc.closed = true
	return sc.Conn.Close()
}

func (sc *NetConnectionClientCommunicator) Closed() bool {
	return sc.closed
}

func (sc *NetConnectionClientCommunicator) SendAndReceive(m *dns.Msg, timeout *time.Duration) (r *dns.Msg, rtt time.Duration, err error) {
	if timeout != nil {
		sc.Client.Timeout = *timeout
	}
	r, rtt, err = sc.Client.ExchangeWithConn(m, sc.Conn)
	err = errors.Wrapf(err, "Could not send packet %v %q to server", dns.Type(m.Question[0].Qtype), m.Question[0].Name)
	return
}

func (sc *NetConnectionClientCommunicator) LocalAddr() net.Addr {
	return sc.Conn.LocalAddr()
}

func (sc *NetConnectionClientCommunicator) RemoteAddr() net.Addr {
	return sc.Conn.RemoteAddr()
}

func (sc *NetConnectionClientCommunicator) SetDeadline(t time.Time) error {
	return sc.Conn.SetDeadline(t)
}

func (sc *NetConnectionClientCommunicator) SetReadDeadline(t time.Time) error {
	return sc.Conn.SetReadDeadline(t)
}

func (sc *NetConnectionClientCommunicator) SetWriteDeadline(t time.Time) error {
	return sc.Conn.SetWriteDeadline(t)
}
