package upstream

import (
	"crypto/tls"
	"github.com/bokysan/socketace/v2/internal/socketace"
	"github.com/bokysan/socketace/v2/internal/streams"
	"github.com/bokysan/socketace/v2/internal/util/cert"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"net"
	"net/url"
)

// SocketClient connects to the server via a socket connection
type SocketClient struct {
	streams.Connection
}

func NewSocketClientConnection(manager cert.TlsConfig, addr *url.URL) (*SocketClient, error) {

	u := *addr

	var secure bool
	var err error
	var c net.Conn

	if streams.HasTls.MatchString(u.Scheme) {
		secure = true
		var tlsConfig *tls.Config
		if tlsConfig, err = manager.GetTlsConfig(); err != nil {
			return nil, errors.Wrapf(err, "Could not configure TLS")
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
		return nil, errors.Wrapf(err, "Could not connect to %v", addr)
	}

	log.Debugf("Connected to %+v", addr)

	var stream streams.Connection
	stream, err = socketace.NewClientConnection(c, manager, secure, addr.Host)
	if err != nil {
		return nil, errors.Wrapf(err, "Could not open connection")
	}
	stream = streams.NewNamedConnection(c, addr.String())

	return &SocketClient{
		Connection: stream,
	}, nil
}
