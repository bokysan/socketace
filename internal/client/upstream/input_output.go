package upstream

import (
	"crypto/tls"
	"github.com/bokysan/socketace/v2/internal/socketace"
	"github.com/bokysan/socketace/v2/internal/streams"
	"github.com/bokysan/socketace/v2/internal/util/addr"
	"github.com/bokysan/socketace/v2/internal/util/cert"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"io"
	"os"
)

// InputOutput is a connection which connects via standard input / output to the server
type InputOutput struct {
	streams.Connection

	// Address is the parsed representation of the address and calculated automatically while unmarshalling
	Address addr.ProtoAddress

	Input  io.ReadCloser
	Output io.WriteCloser
}

func (ups *InputOutput) String() string {
	return ups.Address.String()
}

func (ups *InputOutput) Connect(manager cert.TlsConfig, mustSecure bool) error {
	var stream streams.Connection
	var secure bool
	var err error

	input := ups.Input
	output := ups.Output

	if input == nil {
		input = os.Stdin
	}
	if output == nil {
		output = os.Stdout
	}

	inputOuput := streams.NewReadWriteCloser(input, output)
	stream = streams.NewSimulatedConnection(inputOuput,
		&addr.StandardIOAddress{Address: "client-input"},
		&addr.StandardIOAddress{Address: "client-output"},
	)

	if addr.HasTls.MatchString(ups.Address.Scheme) {
		secure = true
		var tlsConfig *tls.Config
		if tlsConfig, err = manager.GetTlsConfig(); err != nil {
			return errors.Wrapf(err, "Could not configure TLS")
		}
		tlsConfig.InsecureSkipVerify = true

		log.Tracef("[Client] Executing TLS handshake %s", ups.Address.String())
		tlsConn := tls.Client(stream, tlsConfig)
		if err = tlsConn.Handshake(); err != nil {
			return errors.WithStack(err)
		}
		log.Debugf("[Client] Connection encrypted using TLS")
		cert.PrintPeerCertificates(tlsConn)

		stream = streams.NewNamedConnection(tlsConn, "tls")
	} else {
		log.Debugf("Dialing plain %s", ups.Address.String())

	}

	log.Debugf("[Client] Input/output upstream connection established to %+v", ups.Address)

	log.Debugf("[Client] mustSecure=%v", mustSecure)
	cc, err := socketace.NewClientConnection(stream, manager, secure, "")
	log.Debugf("[Client] cc=%v", cc)
	if cc != nil {
		log.Debugf("[Client] mustSecure=%v cc.Secure()=%v", mustSecure, cc.Secure())
	}

	if err != nil {
		return errors.Wrapf(err, "Could not open connection")
	} else if mustSecure && !cc.Secure() {
		return errors.Errorf("Could not establish a secure connection to %v", ups.Address)
	} else {
		stream = cc
	}

	ups.Connection = streams.NewNamedConnection(stream, "stdin")

	return nil
}
