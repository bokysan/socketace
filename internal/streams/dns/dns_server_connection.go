package dns

import (
	"github.com/bokysan/socketace/v2/internal/streams/dns/commands"
	"github.com/bokysan/socketace/v2/internal/streams/dns/util"
	"github.com/miekg/dns"
	"github.com/pkg/errors"
	"golang.org/x/net/webdav"
	"net"
)

// ServerDnsListener will simulate connections over a DNS server request/response loop
type ServerDnsListener struct {
	domain            string
	Communicator      ServerCommunicator
	DefaultSerializer commands.Serializer
}

func NewServerDnsListener(topDomain string, comm ServerCommunicator) (*ServerDnsListener, error) {
	srv := &ServerDnsListener{
		domain:       topDomain,
		Communicator: comm,
		DefaultSerializer: commands.Serializer{
			Domain: topDomain,
			Upstream: util.UpstreamConfig{
				MtuSize:   DefaultUpstreamMtuSize,
				QueryType: &util.QueryTypeCname,
			},
			Downstream: util.DownstreamConfig{},
		},
	}

	comm.RegisterAccept(srv.onMessage)

	return srv, nil
}

func (s *ServerDnsListener) onMessage(m *dns.Msg) (*dns.Msg, error) {
	req, err := s.DefaultSerializer.DecodeDnsRequest(m)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	switch v := req.(type) {
	case *commands.VersionRequest:
		resp := &commands.VersionResponse{
			ServerVersion: ProtocolVersion,
		}
		if v.ClientVersion != ProtocolVersion {
			resp.Err = &commands.BadVersion
		} else {
			resp.UserId = 5
		}
		return s.DefaultSerializer.EncodeDnsResponse(resp)
	}

	return nil, webdav.ErrNotImplemented
}

// Close will close the underlying stream. If the Close has already been called, it will do nothing
func (s *ServerDnsListener) Close() error {
	return s.Communicator.Close()
}

// Closed will return `true` if SafeStream.Close has been called at least once
func (s *ServerDnsListener) Closed() bool {
	return s.Communicator.Closed()
}

func (s *ServerDnsListener) Accept() (net.Conn, error) {
	return nil, webdav.ErrNotImplemented
}

func (s *ServerDnsListener) Addr() net.Addr {
	return s.Communicator.LocalAddr()
}
