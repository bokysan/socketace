package upstream

import (
	"fmt"
	"github.com/bokysan/socketace/v2/internal/streams"
	"github.com/bokysan/socketace/v2/internal/util/addr"
	"github.com/bokysan/socketace/v2/internal/util/buffers"
	"github.com/bokysan/socketace/v2/internal/util/cert"
	ms "github.com/multiformats/go-multistream"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/xtaci/smux"
	"sync"
)

// Upstream adds the Connect method to connect to the upstream
type Upstream interface {
	streams.Connection
	Connect(manager cert.TlsConfig, mustSecure bool) error
}

// ------ // ------ // ------ // ------ // ------ // ------ // ------ //

// Upstreams is a list of upstream servers
type Upstreams struct {
	Data       []Upstream
	MustSecure bool // If MustSecure is true, non-secured sessions are not tolerated
	mutex      sync.Mutex
	connection Upstream
	session    *smux.Session
}

func (ul *Upstreams) UnmarshalFlag(endpoint string) error {
	conn, err := unmarshalUpstream(endpoint)
	if err != nil {
		return err
	}
	ul.Data = append(ul.Data, conn)

	return nil
}

func unmarshalUpstream(endpoint string) (Upstream, error) {
	address, err := addr.ParseAddress(endpoint)
	err = errors.Wrapf(err, "Invalid URL: %s", endpoint)
	if err != nil {
		return nil, err
	}

	switch address.Scheme {
	case "http", "https", "ws", "wss":
		return &Http{Address: *address}, nil
	case "tcp", "tcp+tls", "unix", "unixpacket", "unix+tls", "unixpacket+tls":
		return &Socket{Address: *address}, nil
	case "stdin", "stdin+tls":
		return &InputOutput{Address: *address}, nil
	case "udp", "udp4", "udp6", "unixgram":
		return &Packet{Address: *address}, nil
	case "dns", "dns+udp", "dns+unixgram":
		return &Dns{Address: *address}, nil
	default:
		return nil, errors.Errorf("Unknown scheme: %s", address.Scheme)
	}
}

// creteSession will create a logical connection muxer from a physical connection
func (ul *Upstreams) creteSession() (err error) {

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

func (ul *Upstreams) open(manager cert.TlsConfig) (err error) {
	for _, a := range ul.Data {
		err = a.Connect(manager, ul.MustSecure)
		if err != nil {
			log.WithError(err).Debugf("Could not connect to %v, will retry with the next endpoint.", a)
			continue
		}

		ul.connection = a
		log.Tracef("[Upstream] Physical connection to %v opened", a)

		return ul.creteSession()
	}

	return errors.Errorf("Could not connect to any upstream endpoints!")
}

// openStream will select a specific subprotocol stream within our session
func (ul *Upstreams) openStream(subProtocol string) (streams.ReadWriteCloserClosed, error) {
	conn, err := ul.session.OpenStream()

	if err != nil {
		return nil, err
	}

	stream := streams.NewNamedStream(conn, ul.session.RemoteAddr().String())
	err = ms.SelectProtoOrFail(fmt.Sprintf("/%s", subProtocol), stream)
	if err != nil {
		if e := streams.LogClose(stream); e != nil {
			log.WithError(e).Errorf("Failed closing the connection: %+v", e)
		}
		return nil, errors.Wrapf(err, "Could no select protocol %s", subProtocol)
	}

	return streams.NewNamedStream(stream, subProtocol), err
}

// Connect will return a mutex stream to the first upstream available. If an upstream connection is already opened,
// it will be reused -- only one physical connection will be opened against the server, no matter how many logical
// connections you start.
func (ul *Upstreams) Connect(config cert.ConfigGetter, subProtocol string) (streams.ReadWriteCloserClosed, error) {
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

// Shutdown will close the connection to the connected upstream server
func (ul *Upstreams) Shutdown() {
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
