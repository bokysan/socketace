package server

import (
	"crypto/sha256"
	"fmt"
	"github.com/bokysan/socketace/v2/internal/streams"
	"github.com/bokysan/socketace/v2/internal/util/addr"
	"github.com/bokysan/socketace/v2/internal/util/cert"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/xtaci/kcp-go/v5"
	"golang.org/x/crypto/pbkdf2"
	"net"
	"strings"
)

type PacketServer struct {
	cert.ServerConfig

	Address  addr.ProtoAddress `json:"address"`
	Channels []string          `json:"channels"`

	upstreams Channels
	listener  net.Listener
	done      bool
}

func NewPacketServer() *PacketServer {
	return &PacketServer{}
}

func (st *PacketServer) String() string {
	return fmt.Sprintf("%s", st.Address.String())
}

func (st *PacketServer) Startup(channels Channels) error {
	if upstreams, err := channels.Filter(st.Channels); err != nil {
		return errors.WithStack(err)
	} else {
		st.upstreams = upstreams
	}

	var a = st.Address
	var secure bool
	var block kcp.BlockCrypt
	var pass []byte
	var salt []byte
	var packetListener net.PacketConn

	if st.Address.User != nil {
		if p, set := st.Address.User.Password(); set && p != "" {
			secure = true
			pass = []byte(p)

			// Not the best way to calculate salt but still better than nothing
			h := sha256.New()
			h.Write(pass)
			salt = h.Sum(nil)
		}
	}
	st.Address.User = nil

	n, err := a.Addr()
	if err != nil {
		return errors.WithStack(err)
	}

	if conn, err := net.ListenPacket(n.Network(), n.String()); err != nil {
		return errors.WithStack(err)
	} else {
		packetListener = conn
	}

	if secure {
		log.Infof("Starting AES-encrypted packet server at %s", st.String())
		key := pbkdf2.Key(pass, salt, 1024, 64, sha256.New)
		if b, err := kcp.NewAESBlockCrypt(key); err != nil {
			return errors.WithStack(err)
		} else {
			block = b
		}
	} else {
		log.Infof("Starting plain packet server at %s", st.String())
	}

	listener, err := kcp.ServeConn(block, 10, 3, packetListener)
	if err != nil {
		return errors.WithStack(err)
	} else {
		st.listener = listener
	}

	go func() {
		st.acceptConnection()
	}()

	return nil
}

func (st *PacketServer) acceptConnection() {
	for !st.done {
		conn, err := st.listener.Accept()
		if conn != nil {
			conn = streams.NewNamedConnection(conn, "packet")
			log.Debugf("New connection detected: %+v", conn)
		}
		if st.done {
			if conn != nil && err == nil {
				streams.TryClose(conn)
			}
			break
		}
		if err != nil {
			if !strings.Contains(err.Error(), "use of closed network connection") {
				log.WithError(err).Errorf("Error accepting the connection: %v", err)
				if conn != nil {
					streams.TryClose(conn)
				}
			}
			continue
		}

		// Even though the connection might be secured by an AES-encrypted symmetric ciper, we
		// state here "secure=false" to enable the client to provide StartTLS and do a potential
		// host check and/or identify itself with a client certificate
		if err = AcceptConnection(conn, &st.ServerConfig, false, st.upstreams); err != nil {
			log.WithError(err).Errorf("Error accepting connection: %v", err)
		}
	}
}

func (st *PacketServer) Shutdown() error {
	st.done = true
	return streams.LogClose(st.listener)
}
