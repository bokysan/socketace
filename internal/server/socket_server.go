package server

import (
	"crypto/tls"
	"fmt"
	"github.com/bokysan/socketace/v2/internal/streams"
	"github.com/bokysan/socketace/v2/internal/util/addr"
	"github.com/bokysan/socketace/v2/internal/util/cert"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"net"
	"net/http"
	"strings"
)

type SocketServer struct {
	cert.ServerConfig

	Address  addr.ProtoAddress `json:"address"`
	Channels []string          `json:"channels"`

	secure    bool
	upstreams Channels
	listener  net.Listener
	done      bool
}

func NewSocketServer() *SocketServer {
	return &SocketServer{}
}

func (st *SocketServer) String() string {
	var a addr.ProtoAddress
	a = st.Address
	if st.secure {
		a.Scheme = a.Scheme + "+tls"
	}

	return fmt.Sprintf("%s", a.String())
}

func (st *SocketServer) Startup(channels Channels) error {
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

	var err error

	n, err := st.Address.Addr()
	if err != nil {
		return errors.WithStack(err)
	}

	if st.secure {
		var tlsConfig *tls.Config
		if tlsConfig, err = st.ServerConfig.GetTlsConfig(); err != nil {
			return errors.Wrapf(err, "Could not configure TLS")
		}
		log.Infof("Starting TLS socket server at %s", st.String())
		if st.listener, err = tls.Listen(n.Network(), n.String(), tlsConfig); err != nil && err != http.ErrServerClosed {
			return errors.WithStack(err)
		}
	} else {
		log.Infof("Starting plain socket server at %s", st.String())
		if st.listener, err = net.Listen(n.Network(), n.String()); err != nil && err != http.ErrServerClosed {
			return errors.WithStack(err)
		}
	}

	if err != nil {
		return err
	}

	go func() {
		st.acceptConnection()
	}()

	return nil
}

func (st *SocketServer) acceptConnection() {
	for !st.done {
		conn, err := st.listener.Accept()
		if conn != nil {
			conn = streams.NewNamedConnection(conn, "socket")
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
		if err = AcceptConnection(conn, &st.ServerConfig, st.secure, st.upstreams); err != nil {
			log.WithError(err).Errorf("Error accepting connection: %v", err)
		}
	}
}

func (st *SocketServer) Shutdown() error {
	st.done = true
	return streams.LogClose(st.listener)
}
