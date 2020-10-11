package upstream

import (
	"crypto/tls"
	"github.com/bokysan/socketace/v2/internal/socketace"
	"github.com/bokysan/socketace/v2/internal/streams"
	"github.com/bokysan/socketace/v2/internal/util/cert"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"net/url"
	"os"
)

// StdinClient is a connection which connects via standard input / output to the server
type StdinClient struct {
	streams.Connection
}

func NewStdInClientConnection(manager cert.TlsConfig, addr *url.URL) (*StdinClient, error) {
	u := *addr

	var stream streams.Connection
	var secure bool
	var err error

	inputOuput := streams.NewReadWriteCloser(os.Stdin, os.Stdout)

	stream = streams.NewSimulatedConnection(inputOuput,
		&streams.StandardIOAddress{Address: "input"},
		&streams.StandardIOAddress{Address: "output"},
	)

	if streams.HasTls.MatchString(u.Scheme) {
		secure = true
		var tlsConfig *tls.Config
		if tlsConfig, err = manager.GetTlsConfig(); err != nil {
			return nil, errors.Wrapf(err, "Could not configure TLS")
		}
		tlsConfig.InsecureSkipVerify = true

		log.Debugf("Dialing TLS %s", u.String())
		tlsConn := tls.Client(stream, tlsConfig)
		if err = tlsConn.Handshake(); err != nil {
			return nil, errors.WithStack(err)
		}
		stream = streams.NewSafeConnection(tlsConn)
	} else {
		log.Debugf("Dialing plain %s", u.String())

	}

	stream, err = socketace.NewClientConnection(stream, manager, secure, "")
	stream = streams.NewNamedConnection(stream, "stdin")

	if err != nil {
		return nil, errors.Wrapf(err, "Could not open connection")
	}
	return &StdinClient{
		Connection: stream,
	}, nil
}
