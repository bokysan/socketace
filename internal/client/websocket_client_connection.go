package client

import (
	"crypto/tls"
	"github.com/bokysan/socketace/v2/internal/streams"
	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"io"
	"net/http"
	"net/url"
	"time"
)

type WebsocketClient struct {
	io.ReadWriteCloser
}

func NewWebsocketClientConnection(service *Service, addr *url.URL) (*WebsocketClient, error) {
	tlsConfig := &tls.Config{}

	u := *addr
	if u.Scheme == "http" {
		u.Scheme = "ws"
	} else if u.Scheme == "https" {
		u.Scheme = "wss"
	} else if u.Scheme == "wss" {
	}

	if crt, err := service.GetX509KeyPair(); err != nil {
		return nil, errors.Wrapf(err,"Could not read certificate pair")
	} else if crt != nil {
		tlsConfig.Certificates = make([]tls.Certificate, 1)
		tlsConfig.Certificates[0] = *crt
	}
	service.AddCaCertificates(tlsConfig)

	if service.Insecure {
		tlsConfig.InsecureSkipVerify = true
	}

	dialer := &websocket.Dialer{
		Proxy:            http.ProxyFromEnvironment,
		HandshakeTimeout: 45 * time.Second,
		TLSClientConfig: tlsConfig,
	}

	c, _, err := dialer.Dial(u.String(), nil)
	if err != nil {
		return nil, errors.Wrapf(err, "Could not connect to %v", addr)
	}

	log.Debugf("Connected to %+v", addr)

	closer := streams.NewWebsocketReadWriteCloser(c)
	client, err := streams.NewProxyWrapperClient(closer)
	if err != nil {
		return nil, errors.Wrapf(err, "Could not open connection")
	}

	return &WebsocketClient{
		ReadWriteCloser: client,
	}, nil
}
