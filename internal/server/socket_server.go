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
	"time"
)

type SocketServer struct {
	cert.ServerConfig

	Address  *addr.ProtoAddress `json:"address"`
	Channels []string           `json:"channels"`

	secure    bool
	upstreams Channels
	listener  net.Listener
	done      bool
}

func NewSocketServer() *SocketServer {
	return &SocketServer{}
}

func (st *SocketServer) String() string {
	var addr addr.ProtoAddress
	addr = *st.Address
	if st.secure {
		addr.Scheme = addr.Scheme + "+tls"
	}

	return fmt.Sprintf("%s", addr.String())
}

//goland:noinspection GoUnusedParameter
func (st *SocketServer) Startup(channels Channels) error {
	var errs error
	if upstreams, err := channels.Filter(st.Channels); err != nil {
		return errors.WithStack(errs)
	} else {
		st.upstreams = upstreams
	}

	if strings.HasSuffix(st.Address.Scheme, "+tls") {
		st.Address.Scheme = st.Address.Scheme[:len(st.Address.Scheme)-4]
		st.secure = true
	} else {
		st.secure = false
	}

	startupErrors := make(chan error, 1)
	go func() {
		var err error
		if st.secure {
			var tlsConfig *tls.Config
			if tlsConfig, err = st.ServerConfig.GetTlsConfig(); err != nil {
				startupErrors <- errors.Wrapf(err, "Could not configure TLS")
				return
			}

			log.Infof("Starting TLS socket server at %s", st)
			if st.listener, err = tls.Listen(st.Address.Scheme, st.Address.Host, tlsConfig); err != nil && err != http.ErrServerClosed {
				startupErrors <- errors.WithStack(err)
				return
			}
		} else {
			log.Infof("Starting plain socket server at %s", st)
			if st.listener, err = net.Listen(st.Address.Scheme, st.Address.Host); err != nil && err != http.ErrServerClosed {
				startupErrors <- errors.WithStack(err)
				return
			}
		}

		st.acceptConnection()
	}()

	select {
	case <-time.After(3 * time.Second):
		return nil
	case err := <-startupErrors:
		return err
	}
}

func (st *SocketServer) acceptConnection() {
	for !st.done {
		conn, err := st.listener.Accept()
		if conn != nil {
			conn = streams.NewNamedConnection(conn, "socket")
			log.Debugf("New connection detected: %+v", conn)
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
		streams.TryClose(conn)
	}
}

func (st *SocketServer) Shutdown() error {
	st.done = true
	return streams.LogClose(st.listener)
}
