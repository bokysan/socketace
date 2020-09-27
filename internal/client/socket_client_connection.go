package client

import (
	"crypto/tls"
	"github.com/bokysan/socketace/v2/internal/streams"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"io"
	"net"
	"net/url"
	"regexp"
)

var PlusEnd = regexp.MustCompile("\\+.+$")
var HasTls = regexp.MustCompile("\\+tls")

type SocketClient struct {
	io.ReadWriteCloser
}

func NewSocketClientConnection(service *Service, addr *url.URL) (*SocketClient, error) {
	tlsConfig := &tls.Config{}

	u := *addr

	var err error
	var c net.Conn

	if HasTls.MatchString(u.Scheme) {
		if crt, err := service.GetX509KeyPair(); err != nil {
			return nil, errors.Wrapf(err, "Could not read certificate pair")
		} else if crt != nil {
			tlsConfig.Certificates = make([]tls.Certificate, 1)
			tlsConfig.Certificates[0] = *crt
		}
		service.AddCaCertificates(tlsConfig)

		if service.Insecure {
			tlsConfig.InsecureSkipVerify = true
		}
		u.Scheme = PlusEnd.ReplaceAllString(u.Scheme, "")
		c, err = tls.Dial(u.Scheme, u.Host, tlsConfig)
	} else {
		u.Scheme = PlusEnd.ReplaceAllString(u.Scheme, "")
		c, err = net.Dial(u.Scheme, u.Host)
	}

	if err != nil {
		return nil, errors.Wrapf(err, "Could not connect to %v", addr)
	}

	log.Debugf("Connected to %+v", addr)

	client, err := streams.NewProxyWrapperClient(c)
	if err != nil {
		return nil, errors.Wrapf(err, "Could not open connection")
	}

	return &SocketClient{
		ReadWriteCloser: client,
	}, nil
}

