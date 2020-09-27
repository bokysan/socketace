package client

import (
	"encoding/json"
	"fmt"
	"github.com/bokysan/socketace/v2/internal/util"
	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"io"
	"strings"
)

// ListenList is a list of Listener objects.
type ListenList []Listener

func (ll *ListenList) StartListening(s *Service) error {
	var errs error
	for _, l := range *ll {
		if err := l.Start(s); err != nil {
			errs = multierror.Append(errs, err)
		}
	}
	return errs
}

func (ll *ListenList) MarshalFlag() (string, error) {
	data, err := json.Marshal(ll)
	return string(data), errors.WithStack(err)
}

func (ll *ListenList) UnmarshalFlag(data string) error {

	l := Listener{}

	data = strings.TrimSpace(data)
	if strings.HasPrefix(data, "}") && strings.HasSuffix(data, "}") {
		err := json.Unmarshal([]byte(data), &l)
		err = errors.Wrapf(err, "Failed unmarshalling: %s", data)

		if err != nil {
			return err
		}
	} else if strings.ContainsRune(data, '~') {
		parts := strings.Split(data, "~")

		l.ProtoName.Name = parts[0]
		if addr, err := parseAddress(parts[1]); err != nil {
			return err
		} else {
			l.Address.ProtoAddress = addr
		}

		if len(parts) >= 3 {
			if addr, err := parseAddress(parts[2]); err != nil {
				return err
			} else {
				l.Forward.ProtoAddress = addr
			}
		}

	} else {
		return errors.Errorf("Unknown syntax for listener: %v", data)
	}

	*ll = append(*ll, l)

	return nil
}

func parseAddress(addr string) (util.ProtoAddress, error) {
	addr = strings.TrimSpace(addr)

	if addr == "stdin" || addr == "stdin:" {
		return util.ProtoAddress{
			Network: "stdin",
		}, nil
	}

	parts := strings.SplitN(addr, "://", 2)
	if len(parts) != 2 {
		return util.ProtoAddress{}, errors.Errorf("Invalid address format: %v", addr)
	}

	return util.ProtoAddress{
		Network: parts[0], Address: parts[1],
	}, nil
}

// ------ // ------ // ------ // ------ // ------ // ------ // ------ //

// ProtocolListener is a generic implementation of the specific listener. Required because UPD connections act
// differently than any other connections
type ProtocolListener interface {
	Listen() error
	Accept()
	Shutdown() error
}

func NewProtocolListener(listener *Listener) (l ProtocolListener, err error) {
	addr := listener.Address.ProtoAddress
	switch addr.Network {
	case "stdin":
		l = &StdInProtocolListener{
			listener: listener,
		}

	case "udp", "udp4", "udp6", "unixgram":
		l = &PacketProtocolListener{
			listener: listener,
		}
		return l, errors.Errorf("Packet connections (%v) are not yet supported", addr.Network)
	default:
		l = &StreamProtocolListener{
			listener: listener,
		}
	}

	return l, l.Listen()
}

// ------ // ------ // ------ // ------ // ------ // ------ // ------ //

// Listener is a high-level client-side implementation that listens to connections and trues to connect to backend
// endpoints.
type Listener struct {
	util.ProtoName `yaml:",inline"`

	Address struct {
		util.ProtoAddress `yaml:",inline"`
	} `json:"address" description:"Open a listening connection at this endpoint."`
	Forward struct {
		util.ProtoAddress `yaml:",inline"`
	} `json:"forward" description:"Try forwarding to this address first. Only valid for non-packet connections. (e.g. UDP not supported)"`

	service          *Service
	protocolListener ProtocolListener
}

func (l *Listener) String() string {
	return fmt.Sprintf("%s:%s->%s", l.Name, l.Address.Network, l.Address.Address)
}

func (l *Listener) Start(s *Service) (err error) {
	log.Infof("Opening listener for '%s' on %s://%s", l.Name, l.Address.Network, l.Address.Address)
	l.service = s
	l.protocolListener, err = NewProtocolListener(l)

	if err != nil {
		return errors.Wrapf(err, "Could not listen on %s://%s", l.Address.Network, l.Address.Address)
	}

	go l.protocolListener.Accept()

	err = errors.Wrapf(err, "Could not listen on %s://%s", l.Address.Network, l.Address.Address)
	return
}

func (l *Listener) Shutdown() (err error) {
	if l.protocolListener != nil {
		err = l.protocolListener.Shutdown()
		err = errors.Wrapf(err, "Failed closing down listener for %s %s", l.Address.Network, l.Address.Address)
		l.protocolListener = nil
	}
	return
}

// ------ // ------ // ------ // ------ // ------ // ------ // ------ //

// createUpstreamConnectionMutex will return a mutex stream to the first upstream available
func createUpstreamConnectionMutex(listener *Listener) (mutex io.ReadWriteCloser, err error) {
	for _, a := range listener.service.Upstream {
		mutex, err = getUpstreamMutex(listener.service, a)

		if err != nil {
			log.WithError(err).Debugf("Could not connect to %s, will retry with the next endpoint.", a.Addr())
			continue
		}

		return mutex, nil
	}

	return nil, errors.Errorf("Could not connect to any upstream endpoints!")
}

// getUpstreamMutex will choose a proper upstream connection based on UpstreamServer type.
func getUpstreamMutex(service *Service, a UpstreamServer) (io.ReadWriteCloser, error) {
	log.Debugf("Dialing indirect connection to %s", a.Address)
	var mutex io.ReadWriteCloser
	var err error

	switch a.Addr().Scheme {
	case "http", "https", "ws", "wss":
		mutex, err = NewWebsocketClientConnection(service, a.Addr())
	case "tcp", "tcp+tls":
		mutex, err = NewSocketClientConnection(service, a.Addr())
	case "unix", "unixpacket", "unix+tls", "unixpacket+tls":
		mutex, err = NewSocketClientConnection(service, a.Addr())
	case "stdin":
		mutex, err = NewStdInClientConnection()
	default:
		err = errors.Errorf("Unknown scheme: %s", a.Addr().Scheme)
	}
	return mutex, err
}
