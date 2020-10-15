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

// Socket connects to the server via a socket connection
type Packet struct {
	streams.Connection
	PacketConnection net.PacketConn

	// Address is the parsed representation of the address and calculated automatically while unmarshalling
	Address addr.ProtoAddress
}

func (ups *Packet) String() string {
	return ups.Address.String()
}

func (ups *Packet) Connect(manager cert.TlsConfig, mustSecure bool) error {

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

	if ups.PacketConnection == nil {
		if conn, err := net.ListenPacket(n.Network(), ""); err != nil {
			return errors.WithStack(err)
		} else {
			ups.PacketConnection = conn
		}
	}

	c, err := kcp.NewConn2(n, block, 10, 3, ups.PacketConnection)
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
