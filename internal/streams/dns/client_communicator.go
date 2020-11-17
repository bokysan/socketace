package dns

import (
	"github.com/bokysan/socketace/v2/internal/streams"
	"github.com/miekg/dns"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"io"
	"net"
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

// GenerateAddress will generate an address, optionally adding the specified port if not in the string
func GenerateAddress(server string, defaultPort string) (*net.UDPAddr, error) {
	if addr, err := net.ResolveUDPAddr("udp", server); err == nil {
		if addr.Port != 0 {
			return addr, nil
		}
	} else if addr, err := net.ResolveUDPAddr("udp", server+":"+defaultPort); err != nil {
		return nil, err
	} else {
		return addr, nil
	}

	return nil, errors.New("Could not generate an address")
}

func NewNetConnectionClientCommunicator(config *dns.ClientConfig) (*NetConnectionClientCommunicator, error) {
	if config == nil {
		var err error
		// TODO:: Make this cross platform
		config, err = dns.ClientConfigFromFile("/etc/resolv.conf")
		if err != nil {
			return nil, errors.WithStack(err)
		}
	}

	if len(config.Servers) == 0 {
		return nil, errors.New("You need at least one upstream server!")
	}

	var conn *net.UDPConn
	var err error
	for _, v := range config.Servers {
		var addr *net.UDPAddr
		addr, err = GenerateAddress(v, config.Port)
		if addr == nil {
			err = errors.Errorf("GenerateAddress(%v, %v) returned <nil>", v, config.Port)
		}

		if err != nil {
			err = errors.WithStack(err)
			continue
		}

		// TODO: Add support for TCP DNS
		log.Debugf("Dialing udp %v", addr)
		conn, err = net.DialUDP("udp", nil, addr)
		if err != nil {
			err = errors.WithStack(err)
			continue
		}
	}

	if err != nil {
		return nil, errors.WithStack(err)
	}

	return &NetConnectionClientCommunicator{
		Client: &dns.Client{},
		Conn: &dns.Conn{
			Conn: conn,
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
	err = errors.Wrapf(err, "Could not send packet to server: %v", m)
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
