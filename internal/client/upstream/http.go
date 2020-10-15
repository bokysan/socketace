package upstream

import (
	"crypto/tls"
	"github.com/bokysan/socketace/v2/internal/socketace"
	"github.com/bokysan/socketace/v2/internal/streams"
	"github.com/bokysan/socketace/v2/internal/util/addr"
	"github.com/bokysan/socketace/v2/internal/util/cert"
	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"net/http"
	"time"
)

// Http will establish a connection with the server over HTTP
type Http struct {
	streams.Connection

	// Address is the parsed representation of the address and calculated automatically while unmarshalling
	Address addr.ProtoAddress
}

func (ups *Http) String() string {
	return ups.Address.String()
}

func (ups *Http) Connect(manager cert.TlsConfig, mustSecure bool) error {

	a := ups.Address

	var stream streams.Connection
	var tlsConfig *tls.Config
	var secure bool

	if addr.HasTls.MatchString(a.Scheme) {
		secure = true
		a.Scheme = addr.PlusEnd.ReplaceAllString(a.Scheme, "")
		a.Scheme = a.Scheme + "s"
		if a.Scheme == "http" {
			a.Scheme = "wss"
		}
	} else if a.Scheme == "http" {
		a.Scheme = "ws"
	} else if a.Scheme == "https" {
		a.Scheme = "wss"
		secure = true
	} else if a.Scheme == "wss" {
		secure = true
	}

	if secure {
		if conf, err := manager.GetTlsConfig(); err != nil {
			return errors.Wrapf(err, "Could not read certificate pair")
		} else {
			tlsConfig = conf
		}
	}

	dialer := &websocket.Dialer{
		Proxy:            http.ProxyFromEnvironment,
		HandshakeTimeout: 45 * time.Second,
		TLSClientConfig:  tlsConfig,
	}

	log.Debugf("Dialing %s", a.String())
	c, _, err := dialer.Dial(a.String(), nil)
	if err == websocket.ErrBadHandshake {
		// Gracefully handle errors, fallback to GET/POST handling if websocket connection cannot be established
		return errors.Wrapf(err, "Could not connect to %v", ups.Address)
	} else if err != nil {
		return errors.Wrapf(err, "Could not connect to %v", ups.Address)
	}
	log.Debugf("[Client] Http upstream connection established to %+v", ups.Address)
	cert.PrintPeerCertificates(c.UnderlyingConn())

	stream = streams.NewWebsocketTunnelConnection(c)
	cc, err := socketace.NewClientConnection(stream, manager, secure, ups.Address.Host)
	if err != nil {
		return errors.Wrapf(err, "Could not open connection")
	} else if mustSecure && !cc.Secure() {
		return errors.Errorf("Could not establish a secure connection to %v", ups.Address)
	} else {
		stream = cc
	}
	ups.Connection = streams.NewNamedConnection(streams.NewNamedConnection(stream, ups.Address.String()), "http")

	return nil
}
