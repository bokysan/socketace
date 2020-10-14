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
	Address *addr.ProtoAddress
}

func (ups *Http) String() string {
	return ups.Address.String()
}

func (ups *Http) Connect(manager cert.TlsConfig) error {

	u := *ups.Address

	var stream streams.Connection
	var tlsConfig *tls.Config
	var secure bool

	if streams.HasTls.MatchString(u.Scheme) {
		secure = true
		u.Scheme = streams.PlusEnd.ReplaceAllString(u.Scheme, "")
		u.Scheme = u.Scheme + "s"
		if u.Scheme == "http" {
			u.Scheme = "wss"
		}
	} else if u.Scheme == "http" {
		u.Scheme = "ws"
	} else if u.Scheme == "https" {
		u.Scheme = "wss"
		secure = true
	} else if u.Scheme == "wss" {
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

	log.Debugf("Dialing %s", u.String())
	c, _, err := dialer.Dial(u.String(), nil)
	if err == websocket.ErrBadHandshake {
		// Gracefully handle errors, fallback to GET/POST handling if websocket connection cannot be established
		return errors.Wrapf(err, "Could not connect to %v", ups.Address)
	} else if err != nil {
		return errors.Wrapf(err, "Could not connect to %v", ups.Address)
	}

	log.Debugf("[Client] Http upstream connection established to %+v", ups.Address)

	stream = streams.NewWebsocketTunnelConnection(c)
	stream, err = socketace.NewClientConnection(stream, manager, secure, ups.Address.Host)
	if err != nil {
		return errors.Wrapf(err, "Could not open connection")
	}
	ups.Connection = streams.NewNamedConnection(stream, ups.Address.String())

	return nil
}
