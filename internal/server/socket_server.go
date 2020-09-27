package server

import (
	"crypto/tls"
	"fmt"
	"github.com/bokysan/socketace/v2/internal/cert"
	"github.com/bokysan/socketace/v2/internal/streams"
	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"io"
	"net"
	"net/http"
	"strings"
)

type SocketServer struct {
	cert.Manager
	cert.ClientAuthentication

	Kind     string   `json:"kind"`
	Listen   string   `json:"listen"`
	Network  string   `json:"network"`
	Channels []string `json:"channels"`

	listener  net.Listener
	service   *Service
	tlsConfig *tls.Config
	done      bool
}

func NewSocketServer() *SocketServer {
	return &SocketServer{
		Network: "tcp",
	}
}

func (st *SocketServer) String() string {
	protocol := "tcp"
	if st.tlsConfig != nil {
		protocol += "+tls"
	}

	return fmt.Sprintf("%s://%s", protocol, st.Listen)
}

func (st *SocketServer) SetService(service *Service) {
	st.service = service
}

//goland:noinspection GoUnusedParameter
func (st *SocketServer) Execute(args []string) error {
	log.Infof("Starting socket server...")

	var errs error
	upstreams := make(ChannelList, 0)
	if st.Channels == nil || len(st.Channels) == 0 {
		upstreams = st.service.Channels
	} else {
		for _, ch := range st.Channels {
			upstream, err := st.service.Channels.Find(ch)
			if err != nil {
				errs = multierror.Append(errs, errors.WithStack(err))
				continue
			}
			upstreams = append(upstreams, upstream)
		}
	}

	if len(upstreams) == 0 {
		errs = multierror.Append(errs, errors.Errorf("No upstreams defined for endpoint server"))
	}

	if errs != nil {
		return errors.WithStack(errs)
	}

	if crt, err := st.GetX509KeyPair(); err != nil {
		return errors.WithStack(err)
	} else if crt != nil {
		st.tlsConfig = st.MakeTlsConfig(crt)
		st.AddCaCertificates(st.tlsConfig)
		if st.RequireClientCert {
			st.tlsConfig.ClientAuth = tls.RequireAndVerifyClientCert
		}
		log.Infof("Starting TLS socket server at %s", st)
		if st.listener, err = tls.Listen(st.Network, st.Listen, st.tlsConfig); err != nil && err != http.ErrServerClosed {
			return errors.WithStack(err)
		}
	} else {
		log.Infof("Starting socket at %s", st)
		if st.listener, err = net.Listen(st.Network, st.Listen); err != nil && err != http.ErrServerClosed {
			return errors.WithStack(err)
		}
	}

	for !st.done {
		conn, err := st.listener.Accept()
		if conn != nil {
			log.Debugf("New connection detected: %+v", conn)
		}
		if err != nil {
			if ! strings.Contains(err.Error(), "use of closed network connection") {
				log.WithError(err).Errorf("Error accepting the connection: %v", err)
				if conn != nil {
					conn.Close()
				}
			}
			continue
		}
		go func(conn io.ReadWriteCloser) {
			log.Tracef("Creating new proxy wrapper server...")
			client, err := streams.NewProxyWrapperServer(conn)
			if err != nil {
				if ! strings.Contains(err.Error(), "use of closed network connection") {
					log.WithError(err).Errorf("Could not negotiate connection: %v", err)
					if conn != nil {
						conn.Close()
					}
				}
				return
			}

			if err := MultiplexToUpstream(client, upstreams); err != nil {
				log.WithError(err).Errorf("Error communicating: %v", err)
				if err = conn.Close(); err != nil {
					log.WithError(err).Warnf("Error closing upstream connection: %v", err)
				}
			}
		}(conn)
	}


	return nil
}

func (st *SocketServer) Shutdown() error {
	st.done = true
	return st.listener.Close()
}
