package upstream

import (
	"crypto/tls"
	"github.com/bokysan/socketace/v2/internal/socketace"
	"github.com/bokysan/socketace/v2/internal/streams"
	"github.com/bokysan/socketace/v2/internal/util/addr"
	"github.com/bokysan/socketace/v2/internal/util/cert"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"net"
)

// Socket connects to the server via a socket connection
type Socket struct {
	streams.Connection

	// Address is the parsed representation of the address and calculated automatically while unmarshalling
	Address addr.ProtoAddress
}

func (ups *Socket) String() string {
	return ups.Address.String()
}

func (ups *Socket) Connect(manager cert.TlsConfig, mustSecure bool) error {

	u := ups.Address

	var stream streams.Connection
	var secure bool
	var err error
	var c net.Conn

	if streams.HasTls.MatchString(u.Scheme) {
		secure = true
		var tlsConfig *tls.Config
		if tlsConfig, err = manager.GetTlsConfig(); err != nil {
			return errors.Wrapf(err, "Could not configure TLS")
		}
		u.Scheme = streams.PlusEnd.ReplaceAllString(u.Scheme, "")
		log.Debugf("Dialing TLS %s", u.String())
		c, err = tls.Dial(u.Scheme, u.Host, tlsConfig)
	} else {
		u.Scheme = streams.PlusEnd.ReplaceAllString(u.Scheme, "")
		log.Debugf("Dialing plain %s", u.String())
		c, err = net.Dial(u.Scheme, u.Host)
	}

	if err != nil {
		return errors.Wrapf(err, "Could not connect to %v", ups.Address)
	}

	log.Debugf("[Client] Socket upstream connection established to %+v", ups.Address)
	cert.PrintPeerCertificates(c)

	cc, err := socketace.NewClientConnection(c, manager, secure, ups.Address.Host)
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
