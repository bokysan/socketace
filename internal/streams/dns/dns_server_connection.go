package dns

import (
	"github.com/bokysan/socketace/v2/internal/streams/dns/commands"
	"github.com/bokysan/socketace/v2/internal/streams/dns/util"
	"github.com/bokysan/socketace/v2/internal/util/enc"
	"github.com/miekg/dns"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"golang.org/x/net/dns/dnsmessage"
	"golang.org/x/net/webdav"
	"io"
	"net"
	"os"
	"sync"
	"time"
)

var ConnectionTimeout = 5 * time.Minute          // ConnectionTimeout specifies that connections will timeout 2 minutes after we've seen the last contact from the user
var OldConnectionTimeout = 6 * ConnectionTimeout // Old connections will also timeout after a certain time

// ServerDnsListener will simulate connections over a DNS server request/response loop
type ServerDnsListener struct {
	Communicator      ServerCommunicator   // Communictor does IO. This allows us to abstract away the connection logic
	DefaultSerializer commands.Serializer  // The default serializer that's used when no user-specific serializer can be applied
	domain            string               // The server's top-level DNS domain
	connections       []*userConnection    // List of server connections
	oldConnections    []*userConnection    // List of closed connections
	usersLock         *sync.Mutex          // Mutex for adding and deleting users
	accept            chan *userConnection // Channel to notify on new user connection
}

type userConnection struct {
	UserId     uint16
	Serializer commands.Serializer

	lastConnection time.Time
	localAddress   net.Addr
	remoteAddress  net.Addr
	closer         func(u *userConnection) error
	closed         bool

	in  util.InQueue
	out util.OutQueue
}

func NewServerDnsListener(topDomain string, comm ServerCommunicator) *ServerDnsListener {
	// Users ID is exchanged as 2-char base-36 number between the server and the client. As such, it's simply
	// impossible to host more than 36*36. As this server type is not really meant for  high-scale / high-frequency
	// usage but as a last resort, this should be more than suficient.
	// Especially as SocketAce provides connection multiplexing.
	const MaxUserCount = 36 * 36
	srv := &ServerDnsListener{
		domain:       topDomain,
		Communicator: comm,
		DefaultSerializer: commands.Serializer{
			Domain: topDomain,
			Upstream: util.UpstreamConfig{
				FragmentSize: DefaultUpstreamMtuSize,
				QueryType:    &util.QueryTypeCname,
				Encoder:      enc.Base32Encoding,
			},
			Downstream: util.DownstreamConfig{
				FragmentSize: 1534,
				Encoder:      enc.Base32Encoding,
			},
			UseLazyMode: false,
		},
		connections:    make([]*userConnection, MaxUserCount),
		oldConnections: make([]*userConnection, MaxUserCount),
		usersLock:      &sync.Mutex{},
		accept:         make(chan *userConnection, MaxUserCount),
	}

	// This function will prune stale connections
	go func() {
		for !srv.Closed() {
			time.Sleep(1 * time.Minute)
			now := time.Now()

			srv.usersLock.Lock()

			for _, u := range srv.connections {
				if u == nil {
					continue
				}

				if u.lastConnection.Add(ConnectionTimeout).Before(now) {
					// Remove connection from our list
					log.Infof("Removing stale user connection for user %d (%s)", u.UserId, u.remoteAddress)
					srv.connections[u.UserId] = nil
					srv.oldConnections[u.UserId] = u
				}
			}

			for _, u := range srv.oldConnections {
				if u == nil {
					continue
				}

				if u.lastConnection.Add(OldConnectionTimeout).Before(now) {
					// Remove connection from our list
					log.Infof("Removing stale old connection for user %d (%s)", u.UserId, u.remoteAddress)
					srv.connections[u.UserId] = nil
					srv.oldConnections[u.UserId] = u
				}
			}

			srv.usersLock.Unlock()
		}
	}()

	comm.RegisterAccept(srv.onMessage)

	return srv
}

// newUser will register a new userConnection (or return an error if no more space)
func (s *ServerDnsListener) newUser(a net.Addr) (*userConnection, error) {
	s.usersLock.Lock()
	defer s.usersLock.Unlock()

	for i, u := range s.connections {
		if u == nil {
			u = &userConnection{
				lastConnection: time.Now(),
				localAddress:   s.Addr(),
				remoteAddress:  a,
				UserId:         uint16(i),
				Serializer:     s.DefaultSerializer,
				closer:         s.closeConnection,
			}
			s.connections[i] = u

			log.Infof("New server-side connection initiated for user #%d", i)

			s.accept <- u
			return u, nil
		}
	}

	return nil, commands.BadServerFull
}

func (s *ServerDnsListener) closeConnection(u *userConnection) error {
	s.usersLock.Lock()
	defer s.usersLock.Unlock()

	_, err := s.validateAndGetUser(u.UserId, u.remoteAddress)
	if err == commands.BadUser {
		// Connection already closed
		return nil
	} else if err == commands.BadIp {
		// Connection belongs to another user, ignore
		return nil
	} else if err == commands.BadConn {
		// Connection already closed
		return nil
	}

	log.Debugf("Closing server-side connection for user #%d", u.UserId)

	// Remove connection from our list
	s.connections[u.UserId] = nil
	s.oldConnections[u.UserId] = u
	u.closed = true

	return nil
}

func (s *ServerDnsListener) validateAndGetUser(userId uint16, remoteAddr net.Addr) (*userConnection, error) {
	user := s.connections[userId]
	if user == nil {
		if u := s.oldConnections[userId]; u != nil {
			if u.remoteAddress.String() == remoteAddr.String() {
				return u, commands.BadConn
			}
		}
		return nil, commands.BadUser
	}

	if user.remoteAddress.String() != remoteAddr.String() {
		return user, commands.BadIp
	}

	user.lastConnection = time.Now()
	return user, nil
}

func (s *ServerDnsListener) onMessage(m *dns.Msg, remoteAddr net.Addr) (*dns.Msg, error) {
	var user *userConnection
	var cmd *commands.Command
	var userErr error
	serializer := s.DefaultSerializer
	userId := uint16(0)

	request := commands.ComposeRequest(m, s.DefaultSerializer.Domain)
	for _, c := range commands.Commands {
		if c.IsOfType(request) {
			var err error
			_, userId, err = commands.DecodeRequestHeader(c, request)
			if err != nil {
				return nil, err
			}
			user, userErr = s.validateAndGetUser(userId, remoteAddr)
			if user != nil {
				serializer = user.Serializer
			}
			cmd = &c
			break
		}
	}

	if cmd == nil {
		return s.DefaultSerializer.EncodeDnsResponse(&commands.ErrorResponse{
			Err: commands.BadCommand,
		}, m)
	}

	if user == nil && cmd.NeedsUserId {
		return s.DefaultSerializer.EncodeDnsResponse(&commands.ErrorResponse{
			Err: commands.BadUser,
		}, m)
	}

	if user != nil && userErr == commands.BadConn {
		log.Warnf("User #%d attempted to use a closed connection.", user.UserId)
		return s.DefaultSerializer.EncodeDnsResponse(&commands.ErrorResponse{
			Err: commands.BadConn,
		}, m)
	}

	req, err := serializer.DecodeDnsRequest(request)
	if err != nil {
		err = errors.WithStack(err)
		log.WithError(err).Warnf("Failed to decode request: %v", err)
		return s.DefaultSerializer.EncodeDnsResponse(&commands.ErrorResponse{
			Err: commands.BadCodec,
		}, m)
	}
	switch v := req.(type) {
	case *commands.TestDownstreamEncoderRequest:
		return s.testDownstreamEncoder(v, m)
	case *commands.TestUpstreamEncoderRequest:
		return s.testUpstreamEncoder(v, m, remoteAddr)
	case *commands.TestDownstreamFragmentSizeRequest:
		return s.testDownstreamFragmentSize(v, m, remoteAddr)
	case *commands.SetOptionsRequest:
		return s.setOptionsRequest(v, m, remoteAddr)
	case *commands.VersionRequest:
		return s.version(v, m, remoteAddr)
	case *commands.PacketRequest:
		return s.packet(v, m, remoteAddr)
	}

	return nil, webdav.ErrNotImplemented
}

func (s *ServerDnsListener) packet(v *commands.PacketRequest, m *dns.Msg, remoteAddr net.Addr) (*dns.Msg, error) {
	resp := &commands.PacketResponse{}
	user, err := s.validateAndGetUser(v.UserId, remoteAddr)
	if err != nil {
		resp.Err = err
	} else {
		user.out.UpdateAcked(v.LastAckedSeqNo)

		err := user.in.Append(v.Packet)
		if err != nil {
			resp.Err = err
		} else {
			resp.LastAckedSeqNo = user.in.NextSeqNo - 1
			resp.Packet = user.out.NextChunk()
		}
	}
	if user != nil {
		return user.Serializer.EncodeDnsResponse(resp, m)
	} else {
		return s.DefaultSerializer.EncodeDnsResponse(resp, m)
	}
}

func (s *ServerDnsListener) version(v *commands.VersionRequest, m *dns.Msg, remoteAddr net.Addr) (*dns.Msg, error) {
	resp := &commands.VersionResponse{
		ServerVersion: ProtocolVersion,
	}
	if v.ClientVersion != ProtocolVersion {
		resp.Err = commands.BadVersion
	} else if u, err := s.newUser(remoteAddr); err == nil {
		resp.UserId = u.UserId
	} else {
		resp.Err = err
	}
	return s.DefaultSerializer.EncodeDnsResponse(resp, m)
}

func (s *ServerDnsListener) setOptionsRequest(v *commands.SetOptionsRequest, m *dns.Msg, remoteAddr net.Addr) (*dns.Msg, error) {
	resp := &commands.SetOptionsResponse{}
	user, err := s.validateAndGetUser(v.UserId, remoteAddr)
	if err != nil {
		resp.Err = err
	} else if v.Closed != nil && *v.Closed == true {
		log.Debugf("Client-initiated closing of the connection.")
		_ = s.closeConnection(user)
	} else {
		logString := "SetOptions(user=#%d"
		logData := make([]interface{}, 0)
		logData = append(logData, user.UserId)

		if v.UpstreamEncoder != nil {
			user.Serializer.Upstream.Encoder = v.UpstreamEncoder
			logString += ", upenc=%v"
			logData = append(logData, v.UpstreamEncoder)
		}
		if v.DownstreamEncoder != nil {
			user.Serializer.Downstream.Encoder = v.DownstreamEncoder
			logString += ", downenc=%v"
			logData = append(logData, v.DownstreamEncoder)
		}
		if v.DownstreamFragmentSize != nil {
			user.Serializer.Downstream.FragmentSize = *v.DownstreamFragmentSize
			logString += ", downfrag=%v"
			logData = append(logData, *v.DownstreamFragmentSize)
		}
		if v.LazyMode != nil {
			user.Serializer.UseLazyMode = *v.LazyMode
			logString += ", lazy=%v"
			logData = append(logData, *v.LazyMode)
		}
		if v.MultiQuery != nil {
			user.Serializer.UseMultiQuery = *v.MultiQuery
			logString += ", multi=%v"
			logData = append(logData, *v.MultiQuery)
		}
		log.Infof(logString+")", logData...)
	}
	return s.DefaultSerializer.EncodeDnsResponse(resp, m)
}

func (s *ServerDnsListener) testDownstreamFragmentSize(v *commands.TestDownstreamFragmentSizeRequest, m *dns.Msg, remoteAddr net.Addr) (*dns.Msg, error) {
	resp := &commands.TestDownstreamFragmentSizeResponse{}
	u, err := s.validateAndGetUser(v.UserId, remoteAddr)
	if err != nil {
		resp.Err = err
	} else {
		resp.Data = make([]byte, v.FragmentSize)
		v := byte(107)
		for i := 0; i < len(resp.Data); i++ {
			resp.Data[i] = v
			v = (v + 107) & 0xff
		}
		resp.FragmentSize = uint32(len(resp.Data))
	}
	if u != nil {
		return u.Serializer.EncodeDnsResponse(resp, m)
	} else {
		return s.DefaultSerializer.EncodeDnsResponse(resp, m)
	}
}

func (s *ServerDnsListener) testUpstreamEncoder(v *commands.TestUpstreamEncoderRequest, m *dns.Msg, remoteAddr net.Addr) (*dns.Msg, error) {
	resp := &commands.TestUpstreamEncoderResponse{
		Data: v.Pattern,
	}
	_, err := s.validateAndGetUser(v.UserId, remoteAddr)
	if err != nil {
		resp.Err = err
	}
	return s.DefaultSerializer.EncodeDnsResponse(resp, m)
}

func (s *ServerDnsListener) testDownstreamEncoder(v *commands.TestDownstreamEncoderRequest, m *dns.Msg) (*dns.Msg, error) {
	resp := &commands.TestDownstreamEncoderResponse{
		Data: util.DownloadCodecCheck,
	}
	return s.DefaultSerializer.EncodeDnsResponseWithParams(resp, m, dnsmessage.Type(m.Question[0].Qtype), v.DownstreamEncoder)
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
	for !s.Closed() {
		select {
		case u := <-s.accept:
			return u, nil
		case <-time.After(time.Second):
			// continue, recheck if the DNS server is shutdown
		}
	}
	return nil, os.ErrClosed
}

func (s *ServerDnsListener) Addr() net.Addr {
	return s.Communicator.LocalAddr()
}

func (u *userConnection) Read(b []byte) (n int, err error) {
	if u.closed && !u.in.HasData() {
		return 0, io.EOF
	}
	return u.in.Read(b)
}

func (u *userConnection) Write(b []byte) (n int, err error) {
	if u.closed {
		return 0, os.ErrClosed
	}
	return u.out.Write(b, u.Serializer.Downstream.FragmentSize)
}

func (u *userConnection) Close() error {
	return u.closer(u)
}

func (u *userConnection) LocalAddr() net.Addr {
	return u.localAddress
}

func (u *userConnection) RemoteAddr() net.Addr {
	return u.remoteAddress
}

func (u *userConnection) SetDeadline(t time.Time) (err error) {
	err = u.out.SetWriteDeadline(t)
	if err != nil {
		err = u.in.SetReadDeadline(t)
	}
	return err
}

// SetReadDeadline sets the deadline for future Read calls
// and any currently-blocked Read call.
// A zero value for t means Read will not time out.
func (u *userConnection) SetReadDeadline(t time.Time) error {
	return u.in.SetReadDeadline(t)
}

// SetWriteDeadline sets the deadline for future Write calls
// and any currently-blocked Write call.
// Even if write times out, it may return n > 0, indicating that
// some of the data was successfully written.
// A zero value for t means Write will not time out.
func (u *userConnection) SetWriteDeadline(t time.Time) error {
	return u.out.SetWriteDeadline(t)
}
