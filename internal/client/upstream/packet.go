package upstream

import (
	"crypto/sha256"
	"github.com/bokysan/socketace/v2/internal/socketace"
	"github.com/bokysan/socketace/v2/internal/streams"
	"github.com/bokysan/socketace/v2/internal/util/addr"
	"github.com/bokysan/socketace/v2/internal/util/cert"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/xtaci/kcp-go/v5"
	"golang.org/x/crypto/pbkdf2"
	"net"
)

type ConnectionFromPacketConn func(remote net.Addr, block kcp.BlockCrypt) (net.Conn, error)

// Socket connects to the server via a socket connection
type Packet struct {
	streams.Connection

	// Address is the parsed representation of the address and calculated automatically while unmarshalling
	Address addr.ProtoAddress
}

// DefaultCreateConnection will create a packet connection over UDP using KCP
func DefaultCreateConnection(remote net.Addr, block kcp.BlockCrypt) (net.Conn, error) {
	var listener net.PacketConn
	if conn, err := net.ListenPacket(remote.Network(), ""); err != nil {
		return nil, errors.WithStack(err)
	} else {
		listener = conn
	}

	return kcp.NewConn2(remote, block, 10, 3, listener)
}

func (ups *Packet) String() string {
	return ups.Address.String()
}

// Connect will create a stream over packet connection and use the DefaultCreateConnection to do so.
func (ups *Packet) Connect(manager cert.TlsConfig, mustSecure bool) error {
	return ups.ConnectPacket(manager, mustSecure, DefaultCreateConnection)
}

// ConnectPacket will create a stream over a packet connection. It will take the supplied
// connectFunc to actually "cast" the packet connection into a net.Conn. This is to allow pluggable
// mechanism of underlying packet translation service.
func (ups *Packet) ConnectPacket(manager cert.TlsConfig, mustSecure bool, connectFunc ConnectionFromPacketConn) error {

	var stream streams.Connection
	var secure bool
	var err error

	var block kcp.BlockCrypt
	var pass []byte
	var salt []byte

	if ups.Address.User != nil {
		if p, set := ups.Address.User.Password(); set && p != "" {
			secure = true
			pass = []byte(p)

			// Not the best way to calculate salt but still better than nothing
			h := sha256.New()
			h.Write(pass)
			salt = h.Sum(nil)
		}
	}
	ups.Address.User = nil

	n, err := ups.Address.Addr()
	if err != nil {
		return errors.WithStack(err)
	}

	if secure {
		log.Debugf("Starting AES-encrypted packet client to %s", ups.String())

		key := pbkdf2.Key(pass, salt, 1024, 64, sha256.New)
		if b, err := kcp.NewAESBlockCrypt(key); err != nil {
			return errors.WithStack(err)
		} else {
			block = b
		}
	} else {
		log.Debugf("Starting plain packet client to %s", ups.String())
	}

	c, err := connectFunc(n, block)
	if err != nil {
		return errors.Wrapf(err, "Could not connect to %v", ups.Address)
	}

	log.Debugf("[Client] Socket upstream connection established to %v", ups.Address.String())

	// Even if the packets are encrypted using AES symmetric cyper, let the server know we're open to StartTLS
	// communication. Why? Because:
	// - we can check certificates / hostnames
	// - we can execute mutual (client-server) authentication
	cc, err := socketace.NewClientConnection(c, manager, false, ups.Address.Host)
	if err != nil {
		return errors.Wrapf(err, "Could not open connection")
	} else if mustSecure && !cc.Secure() {
		return errors.Errorf("Could not establish a secure connection to %v", ups.Address)
	} else {
		stream = cc
	}

	ups.Connection = streams.NewNamedConnection(streams.NewNamedConnection(stream, ups.Address.String()), "socket")

	return nil
}
