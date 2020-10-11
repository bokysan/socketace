package upstream

import (
	"fmt"
	"github.com/bokysan/socketace/v2/internal/streams"
	"github.com/bokysan/socketace/v2/internal/util/buffers"
	"github.com/bokysan/socketace/v2/internal/util/cert"
	ms "github.com/multiformats/go-multistream"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/xtaci/smux"
	"net/url"
	"sync"
)

type Server struct {
	// Address is the string representation of the address, as specified by the client
	Address string
	// addr is the parsed representation of the address and calculated automatically while unmarshalling
	addr *url.URL
}

// Addr return the address of the server as URL
func (srv *Server) Addr() *url.URL {
	return srv.addr
}

// NewSocketAceClient will choose a proper upstream connection based on UpstreamServer type, execute SocketAce
// handshake and return the "plain" pysical connection, which can then be wrapped in a logical mutex.
func (srv *Server) ConnectServer(manager cert.TlsConfig) (mutex streams.Connection, err error) {
	switch srv.Addr().Scheme {
	case "http", "https", "ws", "wss":
		mutex, err = NewWebsocketClientConnection(manager, srv.Addr())
	case "tcp", "tcp+tls", "unix", "unixpacket", "unix+tls", "unixpacket+tls":
		mutex, err = NewSocketClientConnection(manager, srv.Addr())
	case "stdin", "stdin+tls":
		mutex, err = NewStdInClientConnection(manager, srv.Addr())
	default:
		err = errors.Errorf("Unknown scheme: %s", srv.Addr().Scheme)
	}

	return
}

// ------ // ------ // ------ // ------ // ------ // ------ // ------ //

type Connector interface {
	// Connect will connect to the first available upstream server, optionally encrpyting the connection
	// with the cert manager. It will try to pick the provided subProtocol and will fail if not found.
	Connect(manager cert.ConfigGetter, subProtocol string) (streams.ReadWriteCloserClosed, error)
}

// ServerList is a list of upstream servers
type ServerList struct {
	data       []Server
	mutex      sync.Mutex
	connection streams.Connection
	session    *smux.Session
}

func (ul *ServerList) UnmarshalFlag(endpoint string) error {
	address, err := url.Parse(endpoint)
	err = errors.Wrapf(err, "Invalid URL: %s", endpoint)
	if err != nil {
		return err
	}

	ul.data = append(ul.data, Server{
		Address: endpoint,
		addr:    address,
	})

	return nil
}

// creteSession will create a logical connection muxer from a physical connection
func (ul *ServerList) creteSession() (err error) {

	config := smux.DefaultConfig()
	config.MaxFrameSize = buffers.BufferSize - 128
	ul.session, err = smux.Client(ul.connection, config)

	if err != nil {
		if e := streams.LogClose(ul.connection); e != nil {
			log.WithError(e).Errorf("Failed closing the connection: %+v", e)
		}
		ul.connection = nil
	}

	return err
}

func (ul *ServerList) open(manager cert.TlsConfig) (err error) {
	var upstream streams.Connection
	for _, a := range ul.data {
		upstream, err = a.ConnectServer(manager)
		if err != nil {
			log.WithError(err).Debugf("Could not connect to %s, will retry with the next endpoint.", a.Addr())
			continue
		}

		ul.connection = streams.NewSafeConnection(upstream)
		log.Tracef("Physical connection to %v opened", a.Address)

		return ul.creteSession()
	}

	return errors.Errorf("Could not connect to any upstream endpoints!")
}

// openStream will select a specific subprotocol stream within our session
func (ul *ServerList) openStream(subProtocol string) (streams.ReadWriteCloserClosed, error) {
	conn, err := ul.session.OpenStream()

	if err != nil {
		return nil, err
	}

	stream := streams.NewSafeConnection(conn)
	err = ms.SelectProtoOrFail(fmt.Sprintf("/%s", subProtocol), stream)
	if err != nil {
		if e := streams.LogClose(stream); e != nil {
			log.WithError(e).Errorf("Failed closing the connection: %+v", e)
		}
		return nil, errors.Wrapf(err, "Could no select protocol %s", subProtocol)
	}

	return streams.NewSafeStream(stream), err
}

// Connect will return a mutex stream to the first upstream available. If an upstream connection is already opened,
// it will be reused -- only one physical connection will be opened against the server, no matter how many logical
// connections you start.
func (ul *ServerList) Connect(config cert.ConfigGetter, subProtocol string) (streams.ReadWriteCloserClosed, error) {
	var err error

	ul.mutex.Lock()
	if ul.connection == nil || ul.connection.Closed() {
		ul.connection = nil
		ul.session = nil
		err = ul.open(config.CertManager())
	}
	ul.mutex.Unlock()

	if err != nil {
		return nil, err
	}

	return ul.openStream(subProtocol)
}

func (ul *ServerList) Shutdown() {
	go func() {
		ul.mutex.Lock()
		if ul.session != nil {
			streams.TryClose(ul.session)
		}
		if ul.connection != nil && !ul.connection.Closed() {
			streams.TryClose(ul.connection)
		}
		ul.connection = nil
		ul.session = nil
		ul.mutex.Unlock()
	}()
}
