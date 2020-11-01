package dns

import (
	"github.com/miekg/dns"
	"golang.org/x/net/webdav"
	"net"
)

// ServerDnsListener will simulate connections over a DNS server request/response loop
type ServerDnsListener struct {
	domain string
	comm   ServerCommunicator
}

func NewServerDnsListener(topDomain string, comm ServerCommunicator) (*ServerDnsListener, error) {
	srv := &ServerDnsListener{
		domain: topDomain,
		comm:   comm,
	}

	comm.RegisterAccept(srv.onMessage)

	return srv, nil
}

func (s *ServerDnsListener) onMessage(m *dns.Msg) (*dns.Msg, error) {
	return nil, webdav.ErrNotImplemented
}

// Close will close the underlying stream. If the Close has already been called, it will do nothing
func (s *ServerDnsListener) Close() error {
	return s.comm.Close()
}

// Closed will return `true` if SafeStream.Close has been called at least once
func (s *ServerDnsListener) Closed() bool {
	return s.comm.Closed()
}

func (s *ServerDnsListener) Accept() (net.Conn, error) {
	return nil, webdav.ErrNotImplemented
}

func (s *ServerDnsListener) Addr() net.Addr {
	return s.comm.LocalAddr()
}
