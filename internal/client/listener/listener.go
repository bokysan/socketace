package listener

import (
	"encoding/json"
	"fmt"
	"github.com/bokysan/socketace/v2/internal/client/upstream"
	"github.com/bokysan/socketace/v2/internal/util/addr"
	"github.com/bokysan/socketace/v2/internal/util/cert"
	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"strings"
)

// ListenList is a list of Listener objects.
type ListenList []*Listener

func (ll *ListenList) StartListening(connector *upstream.Upstreams, config cert.ConfigGetter) error {
	var errs error
	log.Debugf("Start listening on %v listeners", len(*ll))
	for _, l := range *ll {
		if err := l.Start(connector, config); err != nil {
			errs = multierror.Append(errs, err)
		}
	}
	return errs
}

// MarshalFlag will serialize the whole ListenList for storage by flags
func (ll *ListenList) MarshalFlag() (string, error) {
	data, err := json.Marshal(ll)
	return string(data), errors.WithStack(err)
}

// UnmarshalFlag will deserialize the flag (e.g. from command line) into the ListenList slice
func (ll *ListenList) UnmarshalFlag(data string) error {

	l := &Listener{}

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
		if addr, err := addr.ParseAddress(parts[1]); err != nil {
			return err
		} else {
			l.Address = addr
		}

		if len(parts) >= 3 {
			if addr, err := addr.ParseAddress(parts[2]); err != nil {
				return err
			} else {
				l.Forward = addr
			}
		}

	} else {
		return errors.Errorf("Unknown syntax for listener: %v", data)
	}

	log.Infof("Adding listener %p %v to the list", l, l)
	*ll = append(*ll, l)

	return nil
}

// ------ // ------ // ------ // ------ // ------ // ------ // ------ //

// ProtocolListener is a generic implementation of the specific listener. Required because UPD connections act
// differently than any other connections
type ProtocolListener interface {
	Listen() error
	Accept()
	Shutdown() error
}

func NewProtocolListener(listener *Listener) (ProtocolListener, error) {
	var l ProtocolListener
	addr := listener.Address
	switch addr.Scheme {
	case "stdin":
		l = &StdInProtocolListener{
			listener: listener,
		}

	case "udp", "udp4", "udp6", "unixgram":
		return l, errors.Errorf("Packet connections (%v) are not yet supported", addr.Scheme)
	default:
		l = &StreamProtocolListener{
			listener: listener,
		}
	}

	return l, l.Listen()
}

// ------ // ------ // ------ // ------ // ------ // ------ // ------ //

// Listener is a high-level client-side implementation that listens to connections and tries to connect
// to backend endpoint(s).
type Listener struct {
	addr.ProtoName `yaml:",inline"`

	Address *addr.ProtoAddress `json:"address" description:"Connect a listening connection at this endpoint."`
	Forward *addr.ProtoAddress `json:"forward" description:"Try forwarding to this address first. Only valid for non-packet connections. (e.g. UDP not supported)"`

	upstreams        *upstream.Upstreams
	config           cert.ConfigGetter
	protocolListener ProtocolListener
}

func (l *Listener) String() string {
	return fmt.Sprintf("%s:%s", l.Name, l.Address.String())
}

// Starts opens the listening connection on specific network (e.g. TCP, STDIN, UNIX...)
func (l *Listener) Start(upstreams *upstream.Upstreams, config cert.ConfigGetter) (err error) {
	l.upstreams = upstreams
	l.config = config

	log.Infof("Listener on %p %v", l, l.String())
	l.protocolListener, err = NewProtocolListener(l)
	log.Debugf("... %p = %v -> %v", l.protocolListener, l.protocolListener, l)

	if err != nil {
		return errors.Wrapf(err, "Could not listen on %s", l.Address)
	}

	go l.protocolListener.Accept()

	err = errors.Wrapf(err, "Could not listen on %s", l.Address)
	return
}

// Shutdown stops listening on specific network connection
func (l *Listener) Shutdown() (err error) {
	if l.protocolListener != nil {
		err = l.protocolListener.Shutdown()
		err = errors.Wrapf(err, "Failed closing down listener for %s", l.Address)
		l.protocolListener = nil
	}
	return
}
