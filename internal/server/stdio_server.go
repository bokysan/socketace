package server

import (
	"crypto/tls"
	"fmt"
	"github.com/bokysan/socketace/v2/internal/streams"
	"github.com/bokysan/socketace/v2/internal/util/addr"
	"github.com/bokysan/socketace/v2/internal/util/cert"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"io"
	"os"
)

type IoServer struct {
	cert.ServerConfig

	Address  addr.ProtoAddress `json:"address"`
	Channels []string          `json:"channels"`

	Input  io.ReadCloser
	Output io.WriteCloser

	secure     bool
	upstreams  Channels
	connection io.ReadWriteCloser
}

func NewIoServer() *IoServer {
	return &IoServer{
		Input:  os.Stdin,
		Output: os.Stdout,
	}
}

func (st *IoServer) String() string {
	return fmt.Sprintf("%v", st.Address)
}

//goland:noinspection GoUnusedParameter
func (st *IoServer) Startup(channels Channels) error {

	var errs error
	if upstreams, err := channels.Filter(st.Channels); err != nil {
		return errors.WithStack(errs)
	} else {
		st.upstreams = upstreams
	}

	var secure bool
	var err error
	var stream streams.Connection

	inputOuput := streams.NewReadWriteCloser(st.Input, st.Output)

	stream = streams.NewSimulatedConnection(inputOuput,
		&addr.StandardIOAddress{Address: "server-input"},
		&addr.StandardIOAddress{Address: "server-output"},
	)

	var tlsConfig *tls.Config
	if addr.HasTls.MatchString(st.Address.Scheme) {
		log.Infof("Starting TLS stdio server at %s", st.String())

		secure = true
		if tlsConfig, err = st.ServerConfig.GetTlsConfig(); err != nil {
			return errors.Wrapf(err, "Could not configure TLS")
		}
	} else {
		st.Address.Scheme = "stdio"
		log.Infof("Starting plain stdio server at %s", st.String())

	}

	go func() {
		if tlsConfig != nil {
			log.Tracef("[Server] Executing TLS handshake...")
			tlsConn := tls.Server(stream, tlsConfig)
			if err = tlsConn.Handshake(); err != nil {
				log.WithError(err).Errorf("Error executing TLS handshake: %v", err)
				return
			}
			log.Debugf("[Server] Connection encrypted using TLS")
			stream = streams.NewNamedConnection(tlsConn, "tls")
		}
		stream = streams.NewNamedConnection(stream, "stdin")

		if err := AcceptConnection(stream, &st.ServerConfig, secure, st.upstreams); err != nil {
			log.WithError(err).Errorf("Error accepting connection: %v", err)
		}
	}()

	return nil
}

func (st *IoServer) Shutdown() error {
	if st.connection != nil {
		err := streams.LogClose(st.connection)
		return err
	} else {
		return nil
	}
}
