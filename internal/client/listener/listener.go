package listener

import (
	"encoding/json"
	"fmt"
	"github.com/bokysan/socketace/v2/internal/client/upstream"
	"github.com/bokysan/socketace/v2/internal/streams"
	"github.com/bokysan/socketace/v2/internal/util/addr"
	"github.com/bokysan/socketace/v2/internal/util/cert"
	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"net"
	"os"
	"strings"
)

// Listeners is a list of Listener objects.
type Listeners []Listener

func (ll *Listeners) Start(connector *upstream.Upstreams, config cert.ConfigGetter) error {
	var errs error
	log.Debugf("Start listening on %v listeners", len(*ll))
	for _, l := range *ll {
		if err := l.Start(connector, config); err != nil {
			errs = multierror.Append(errs, err)
		}
	}
	return errs
}

// MarshalFlag will serialize the whole Listeners for storage by flags
func (ll *Listeners) MarshalFlag() (string, error) {
	data, err := json.Marshal(ll)
	return string(data), errors.WithStack(err)
}

// UnmarshalFlag will deserialize the flag (e.g. from command line) into the Listeners slice
func (ll *Listeners) UnmarshalFlag(data string) error {
	data = strings.TrimSpace(data)
	var l Listener

	if strings.HasPrefix(data, "}") && strings.HasSuffix(data, "}") {
		stuff := make(map[string]interface{}, 0)
		err := json.Unmarshal([]byte(data), &stuff)
		if err != nil {
			return errors.Wrapf(err, "Failed unmarshalling: %s", data)
		}

		if a, ok := stuff["address"]; ok {
			address, err := addr.ParseAddress(a.(string))
			if err != nil {
				return errors.Wrapf(err, "Can't parse %q into an address", a)
			}

			switch address.Scheme {
			case "stdin", "stdio":
				l := &InputOutputListener{}
				err := json.Unmarshal([]byte(data), l)
				if err != nil {
					return errors.Wrapf(err, "Failed unmarshalling: %s", data)
				}
			case "tcp", "unix", "unixpacket":
				l := &SocketListener{}
				err := json.Unmarshal([]byte(data), l)
				if err != nil {
					return errors.Wrapf(err, "Failed unmarshalling: %s", data)
				}
			default:
				return errors.Errorf("Can't handle format: %s", address.Scheme)
			}
		} else {
			return errors.Errorf("Can't find address in: %s", data)

		}

	} else if strings.ContainsRune(data, '~') {
		parts := strings.Split(data, "~")

		var forward *addr.ProtoAddress

		channel := parts[0]
		address, err := addr.ParseAddress(parts[1])
		if err != nil {
			return errors.Wrapf(err, "Can't parse %q into an address", parts[1])
		}
		if len(parts) >= 3 {
			if a, err := addr.ParseAddress(parts[2]); err != nil {
				return errors.Wrapf(err, "Can't parse %q into an address", parts[2])
			} else {
				forward = a
			}
		}

		switch address.Scheme {
		case "stdin", "stdio":
			l = &InputOutputListener{
				AbstractListener: AbstractListener{
					ProtoName: addr.ProtoName{
						Name: channel,
					},
					Address: *address,
					Forward: forward,
				},
			}
		case "tcp", "unix", "unixpacket":
			l = &SocketListener{
				AbstractListener: AbstractListener{
					ProtoName: addr.ProtoName{
						Name: channel,
					},
					Address: *address,
					Forward: forward,
				},
			}
		default:
			return errors.Errorf("Can't handle format: %s", address.Scheme)
		}

	} else {
		return errors.Errorf("Unknown syntax for listener: %v", data)
	}

	log.Infof("Adding listener %p %v to the list", l, l)
	*ll = append(*ll, l)

	return nil
}

// ------ // ------ // ------ // ------ // ------ // ------ // ------ //

// Listener is a high-level implementation that listens to connections and tries to connect to backend upstreams(s).
type Listener interface {
	fmt.Stringer

	// Start will start listening on the given network
	Start(upstreams *upstream.Upstreams, config cert.ConfigGetter) error
	// Shutdown listening on a given port
	Shutdown() error
}

type AbstractListener struct {
	addr.ProtoName `yaml:",inline"`

	Address addr.ProtoAddress  `json:"address" description:"Connect a listening connection at this endpoint."`
	Forward *addr.ProtoAddress `json:"forward" description:"Try forwarding to this address first. Only valid for non-packet connections. (e.g. UDP not supported)"`

	Upstreams *upstream.Upstreams
	Config    cert.ConfigGetter
}

func (l *AbstractListener) String() string {
	return fmt.Sprintf("%s->%s", l.Address.String(), l.Name)
}

func (l *AbstractListener) ConnectDirectly(conn net.Conn) bool {
	forward := l.Forward
	if forward == nil {
		return false
	}
	if forward.Host == "" || forward.Scheme == "" {
		return false
	}
	log.Debugf("Dialing direct connection to %s %s", forward.Scheme, forward.Host)
	var direct net.Conn
	var err error
	direct, err = net.Dial(forward.Scheme, forward.Host)
	if err == nil {
		direct = streams.NewNamedConnection(direct, fmt.Sprintf("%v", forward))
		err = streams.PipeData(conn, direct)
		if err != nil {
			err = errors.WithStack(err)
			log.WithError(err).Warnf("Error while communicating %s with %s %s: %+v",
				l.Name, forward.Scheme, forward.Host, err,
			)
		}
		return true
	}

	return false
}

func (l *AbstractListener) HandleConnection(conn net.Conn) {
	// Try connecting directly first
	if l.ConnectDirectly(conn) {
		return
	}

	log.Tracef("Connecting to upstream for channel %s...", l.Name)
	up, err := l.Upstreams.Connect(l.Config, l.Name)
	if err != nil {
		log.WithError(err).Warnf("Communication for %s with upstream failed: %v", l.Name, err)
	} else {
		log.Tracef("...connected to %s via %s", l.Name, up)
		stream := streams.NewNamedStream(conn, "->"+conn.RemoteAddr().String())
		err = streams.PipeData(stream, up)
		if err != nil {
			log.WithError(err).Warnf("Communication for %s with upstream failed: %v", l.Name, err)
		}
	}

	log.Trace("Closing upstream connection.")
	streams.TryClose(up)
	streams.TryClose(conn)
}

type InputOutputListener struct {
	AbstractListener
	InputOutput streams.Connection

	shutdown chan bool
	upstream streams.ReadWriteCloserClosed
}

func (l *InputOutputListener) Start(upstreams *upstream.Upstreams, config cert.ConfigGetter) (err error) {
	l.Upstreams = upstreams
	l.Config = config

	if l.InputOutput == nil {
		var stream streams.Connection

		inputOuput := streams.NewReadWriteCloser(os.Stdin, os.Stdout)
		stream = streams.NewSimulatedConnection(inputOuput,
			&addr.StandardIOAddress{Address: "client-input"},
			&addr.StandardIOAddress{Address: "client-output"},
		)
		stream = streams.NewNamedConnection(stream, "stdio")
		l.InputOutput = stream
	}

	if _, ok := l.InputOutput.(streams.Connection); !ok {
		l.InputOutput = streams.NewSafeConnection(l.InputOutput)
	}

	log.Infof("Staring InputOutputListener %v", l.String())
	go l.HandleConnection(l.InputOutput)

	return nil
}

func (l *InputOutputListener) Shutdown() (err error) {
	if l.InputOutput != nil {
		return streams.LogClose(l.InputOutput)
	} else {
		return nil
	}
}

type SocketListener struct {
	AbstractListener
	netListener net.Listener
	shutdown    chan bool
}

func (l *SocketListener) Start(upstreams *upstream.Upstreams, config cert.ConfigGetter) (err error) {
	l.Upstreams = upstreams
	l.Config = config

	log.Infof("Starting SocketListener %v", l.String())
	l.shutdown = make(chan bool, 1)
	l.netListener, err = net.Listen(l.Address.Scheme, l.Address.Host)

	if err == nil {
		go l.accept()
	}

	return errors.WithStack(err)
}

// Shutdown stops listening on specific network connection
func (l *SocketListener) Shutdown() (err error) {
	if l.netListener != nil {
		l.shutdown <- true
		err = errors.WithStack(streams.LogClose(l.netListener))
		l.netListener = nil
	}
	return
}

func (l *SocketListener) accept() {
	for {
		select {
		case <-l.shutdown:
			return
		default:
			conn, err := l.netListener.Accept()
			select {
			case <-l.shutdown:
				return
			default:
				if err != nil {
					if strings.Contains(err.Error(), "use of closed network connection") {
						return
					}
					err = errors.Wrap(err, "Trouble accepting connection!")
					log.WithError(err).Errorf("Could not accept connection: %+v", err)
					continue
				}
				log.Debugf("Acceping connection on %p = %v, %s->%s", l, l, conn.RemoteAddr(), conn.LocalAddr())
				go l.HandleConnection(conn)
			}
		}

	}
}
