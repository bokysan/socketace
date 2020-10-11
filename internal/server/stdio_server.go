package server

import (
	"crypto/tls"
	"fmt"
	"github.com/bokysan/socketace/v2/internal/streams"
	"github.com/bokysan/socketace/v2/internal/util/cert"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"io"
	"os"
)

type StdioServer struct {
	AbstractServer
	cert.ServerConfig

	Channels []string `json:"channels"`

	upstreams ChannelList

	connection io.ReadWriteCloser
}

func NewStdioServer() *StdioServer {
	return &StdioServer{}
}

func (st *StdioServer) String() string {
	return fmt.Sprintf("%s://%s", st.Kind)
}

//goland:noinspection GoUnusedParameter
func (st *StdioServer) Startup(channels ChannelList) error {

	var errs error
	if upstreams, err := channels.Filter(st.Channels); err != nil {
		return errors.WithStack(errs)
	} else {
		st.upstreams = upstreams
	}

	var secure bool
	var err error
	var stream streams.Connection

	inputOuput := streams.NewReadWriteCloser(os.Stdin, os.Stdout)

	stream = streams.NewSimulatedConnection(inputOuput,
		&streams.StandardIOAddress{Address: "input"},
		&streams.StandardIOAddress{Address: "output"},
	)

	if streams.HasTls.MatchString(st.Kind) {
		log.Infof("Starting TLS stdio server at %s", st)

		secure = true
		var tlsConfig *tls.Config
		if tlsConfig, err = st.ServerConfig.GetTlsConfig(); err != nil {
			return errors.Wrapf(err, "Could not configure TLS")
		}
		tlsConn := tls.Server(stream, tlsConfig)

		if err = tlsConn.Handshake(); err != nil {
			return errors.WithStack(err)
		}
		stream = streams.NewSafeConnection(tlsConn)
	} else {
		log.Infof("Starting plain stdio server at %s", st)

	}

	stream = streams.NewNamedConnection(stream, "stdin")
	return AcceptConnection(stream, &st.ServerConfig, secure, st.upstreams)
}

func (st *StdioServer) Shutdown() error {
	if st.connection != nil {
		err := streams.LogClose(st.connection)
		return err
	} else {
		return nil
	}
}
