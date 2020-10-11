package server

import (
	"crypto/tls"
	"fmt"
	"github.com/bokysan/socketace/v2/internal/streams"
	"github.com/bokysan/socketace/v2/internal/util/cert"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"net"
	"net/http"
	"strings"
)

type SocketServer struct {
	AbstractServer
	cert.ServerConfig

	Listen   string   `json:"listen"`
	Channels []string `json:"channels"`

	upstreams ChannelList
	network   string
	listener  net.Listener
	done      bool
}

func NewSocketServer() *SocketServer {
	return &SocketServer{}
}

func (st *SocketServer) String() string {
	return fmt.Sprintf("%s://%s", st.Kind, st.Listen)
}

//goland:noinspection GoUnusedParameter
func (st *SocketServer) Startup(channels ChannelList) error {
	var errs error
	if upstreams, err := channels.Filter(st.Channels); err != nil {
		return errors.WithStack(errs)
	} else {
		st.upstreams = upstreams
	}

	if strings.HasSuffix(st.Kind, "+tls") {
		st.network = st.Kind[:len(st.Kind)-4]
		st.secure = true
	} else {
		st.network = st.Kind
		st.secure = false
	}

	var err error
	if st.secure {
		var tlsConfig *tls.Config
		if tlsConfig, err = st.ServerConfig.GetTlsConfig(); err != nil {
			return errors.Wrapf(err, "Could not configure TLS")
		}

		log.Infof("Starting TLS socket server at %s", st)
		if st.listener, err = tls.Listen(st.network, st.Listen, tlsConfig); err != nil && err != http.ErrServerClosed {
			return errors.WithStack(err)
		}
	} else {
		log.Infof("Starting plain socket server at %s", st)
		if st.listener, err = net.Listen(st.network, st.Listen); err != nil && err != http.ErrServerClosed {
			return errors.WithStack(err)
		}
	}

	st.acceptConnection()

	return nil
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
