package dns

import (
	"github.com/bokysan/socketace/v2/internal/streams/dns/commands"
	"github.com/bokysan/socketace/v2/internal/streams/dns/util"
	"github.com/miekg/dns"
	"github.com/pkg/errors"
	"golang.org/x/net/dns/dnsmessage"
	"golang.org/x/net/webdav"
	"net"
	"sync"
)

// ServerDnsListener will simulate connections over a DNS server request/response loop
type ServerDnsListener struct {
	domain            string
	allowedQueryTypes []dnsmessage.Type // used for testing
	Communicator      ServerCommunicator
	DefaultSerializer commands.Serializer
	users             []*user
	usersLock         *sync.Mutex
}

type user struct {
	Address    net.Addr
	UserId     byte
	Serializer commands.Serializer
}

func NewServerDnsListener(topDomain string, comm ServerCommunicator) *ServerDnsListener {
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
		users:     make([]*user, 256),
		usersLock: &sync.Mutex{},
	}
	comm.RegisterAccept(srv.onMessage)

	return srv
}

// newUser will register a new user (or return an error if no more space
func (s *ServerDnsListener) newUser(a net.Addr) (*user, error) {
	s.usersLock.Lock()
	for i, u := range s.users {
		if u == nil {
			u = &user{
				Address:    a,
				UserId:     byte(i),
				Serializer: s.DefaultSerializer,
			}
			s.users[i] = u
			return u, nil
		}
	}
	s.usersLock.Unlock()
	return nil, &commands.BadServerFull
}

func (s *ServerDnsListener) onMessage(m *dns.Msg, remoteAddr net.Addr) (*dns.Msg, error) {
	req, err := s.DefaultSerializer.DecodeDnsRequest(m)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	switch v := req.(type) {
	case *commands.TestDownstreamEncoderRequest:
		if s.allowedQueryTypes != nil && len(s.allowedQueryTypes) > 0 {
			// For testing, make sure query type is supported
			ok := false
			for _, qt := range s.allowedQueryTypes {
				if uint16(qt) == m.Question[0].Qtype {
					ok = true
					break
				}
			}

			if !ok {
				return nil, errors.Errorf("Query type %v not supported", dnsmessage.Type(m.Question[0].Qtype))
			}
		}

		ser := s.DefaultSerializer
		ser.Downstream.Encoder = v.DownstreamEncoder
		resp := &commands.TestDownstreamEncoderResponse{
			Data: []byte(util.DownloadCodecCheck),
		}
		return ser.EncodeDnsResponse(resp)
	case *commands.VersionRequest:
		resp := &commands.VersionResponse{
			ServerVersion: ProtocolVersion,
		}
		if v.ClientVersion != ProtocolVersion {
			resp.Err = &commands.BadVersion
		} else if u, err := s.newUser(remoteAddr); err == nil {
			resp.UserId = u.UserId
		} else {
			resp.Err = err
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
