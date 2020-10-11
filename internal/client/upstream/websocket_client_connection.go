package upstream

import (
	"crypto/tls"
	"github.com/bokysan/socketace/v2/internal/socketace"
	"github.com/bokysan/socketace/v2/internal/streams"
	"github.com/bokysan/socketace/v2/internal/util/cert"
	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"net/http"
	"net/url"
	"time"
)

// WebsocketClient will establish a connection with the server over HTTP
type WebsocketClient struct {
	streams.Connection
}

func NewWebsocketClientConnection(manager cert.TlsConfig, addr *url.URL) (*WebsocketClient, error) {

	var tlsConfig *tls.Config
	var secure bool

	u := *addr
	if u.Scheme == "http" {
		u.Scheme = "ws"
	} else if u.Scheme == "https" {
		u.Scheme = "wss"
		secure = true
	} else if u.Scheme == "wss" {
		secure = true
	}

	if secure {
		if conf, err := manager.GetTlsConfig(); err != nil {
			return nil, errors.Wrapf(err, "Could not read certificate pair")
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
		return nil, errors.Wrapf(err, "Could not connect to %v", addr)
	} else if err != nil {
		return nil, errors.Wrapf(err, "Could not connect to %v", addr)
	}

	log.Debugf("Connected to %+v", addr)

	var stream streams.Connection
	stream = streams.NewWebsocketTunnelConnection(c)
	stream, err = socketace.NewClientConnection(stream, manager, secure, addr.Host)
	if err != nil {
		return nil, errors.Wrapf(err, "Could not open connection")
	}
	stream = streams.NewNamedConnection(stream, addr.String())

	return &WebsocketClient{
		Connection: stream,
	}, nil
}
