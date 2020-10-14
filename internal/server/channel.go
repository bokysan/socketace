package server

import (
	"encoding/json"
	"fmt"
	"github.com/armon/go-socks5"
	"github.com/bokysan/socketace/v2/internal/streams"
	"github.com/bokysan/socketace/v2/internal/util/addr"
	"github.com/davecgh/go-spew/spew"
	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"io"
	"net"
	"regexp"
)

type Channel interface {
	fmt.Stringer
	Name() string
	OpenConnection() (net.Conn, error)
}

type AbstractChannel struct {
	addr.ProtoName `yaml:",inline"`
	Address        addr.ProtoAddress `json:"address"`
}

func (u *AbstractChannel) Name() string {
	return u.ProtoName.Name
}

type SocksChannel struct {
	AbstractChannel
}

func (u *SocksChannel) String() string {
	return fmt.Sprintf("%v:%v", u.Name(), "socks")
}

func (u *SocksChannel) OpenConnection() (net.Conn, error) {
	conf := &socks5.Config{}
	server, err := socks5.New(conf)
	if err != nil {
		return nil, errors.Wrapf(err, "Could not configure a socks proxy!")
	}

	p1Reader, p1Writer := io.Pipe()
	p2Reader, p2Writer := io.Pipe()
	p1 := streams.NewReadWriteCloser(p1Reader, p2Writer)
	p2 := streams.NewReadWriteCloser(p2Reader, p1Writer)

	var clientPipe streams.Connection
	var serverPipe streams.Connection

	clientPipe = streams.NewSimulatedConnection(p1, streams.Localhost, streams.Localhost)
	serverPipe = streams.NewSimulatedConnection(p2, streams.Localhost, streams.Localhost)

	clientPipe = streams.NewNamedConnection(clientPipe, u.String())

	go func() {
		defer streams.TryClose(serverPipe)
		if err := server.ServeConn(serverPipe); err != nil {
			log.WithError(err).Errorf("Error processing SOCKS connection: %v", err)
		}
	}()

	return clientPipe, nil
}

// Channel is a configuration of one of the server that are going to be multiplexed in the connection
type NetworkChannel struct {
	AbstractChannel
}

func (u *NetworkChannel) String() string {
	return fmt.Sprintf("%v->%v", u.Name(), u.Address)
}

// OpenConnection will open a connection the the upstream server
func (u *NetworkChannel) OpenConnection() (net.Conn, error) {
	scheme := u.Address.Scheme
	switch scheme {
	case "udp", "udp4", "udp6", "unixgram":
		return nil, errors.Errorf("Packet connections (%v) are not yet supported", scheme)
	}

	conn, err := net.Dial(u.Address.Scheme, u.Address.Host)
	if err != nil {
		err = errors.Wrapf(err, "Remote connection failed to %v", u.Address)
		log.WithError(err).Errorf("Could not connect to %v: %+v", u.Address, err)
	}
	conn = streams.NewNamedConnection(conn, u.String())

	log.Tracef("[Channel] Connected to %v", conn)
	return conn, err
}

// ------ // ------ // ------ // ------ // ------ // ------ // ------ //

var ChannelRegex = regexp.MustCompile("^(/[a-z0-9_^/]*)->((tcp|udp|unix|unixgram|unixpacket):(.*))$")

type Channels []Channel

func (chl *Channels) String() string {
	return spew.Sdump(chl)
}

// Filter will return a list of channels if they are contained in the list of names. It will throw an error
// if no channels can be identified (either this list is empty or no match if found). If the list of names
// is empty or nil, it will return all the channels
func (chl *Channels) Filter(names []string) (Channels, error) {

	if names == nil || len(names) == 0 {
		return *chl, nil
	}

	var errs error

	upstreams := make(Channels, 0)
	for _, ch := range names {
		upstream, err := chl.Find(ch)
		if err != nil {
			errs = multierror.Append(errs, errors.WithStack(err))
			continue
		}
		upstreams = append(upstreams, upstream)
	}

	if len(upstreams) == 0 {
		errs = multierror.Append(errs, errors.Errorf("No upstreams defined for endpoint server"))
	}

	return upstreams, errs
}

// Find finds an endpoint by name (case sensitive). If the
// endpoint does not exist, it returns an error
func (chl *Channels) Find(name string) (Channel, error) {
	var available []string
	for _, e := range *chl {
		if e.Name() == name {
			return e, nil
		}

		available = append(available, e.Name())
	}
	return nil, errors.Errorf("Could not find endpoint with name: '%s' among: %v", name, available)
}

func (chl *Channels) UnmarshalFlag(endpoint string) error {

	if !ChannelRegex.MatchString(endpoint) {
		return errors.Errorf("Channel '%s' does not match %s!", endpoint, ChannelRegex.String())
	}

	parts := ChannelRegex.FindAllStringSubmatch(endpoint, -1)[0]

	address, err := addr.ParseAddress(parts[1])
	if err != nil {
		return err
	}

	e := &NetworkChannel{
		AbstractChannel: AbstractChannel{
			ProtoName: addr.ProtoName{
				Name: parts[0],
			},
			Address: *address,
		},
	}

	*chl = append(*chl, e)

	return nil
}

func (chl *Channels) UnmarshalYAML(unmarshal func(interface{}) error) error {
	stuff := make([]interface{}, 0)
	if err := unmarshal(&stuff); err != nil {
		return errors.WithStack(err)
	}

	res := make(Channels, 0)
	for _, s := range stuff {
		if channel, err := unmarshalChannel(s); err != nil {
			return errors.WithStack(err)
		} else {
			res = append(res, channel)
		}
	}

	*chl = res
	return nil
}

func (chl *Channels) UnmarshalJSON(b []byte) error {
	stuff := make([]interface{}, 0)
	if err := json.Unmarshal(b, &stuff); err != nil {
		return errors.WithStack(err)
	}

	res := make(Channels, 0)
	for _, s := range stuff {
		if channel, err := unmarshalChannel(s); err != nil {
			return errors.WithStack(err)
		} else {
			res = append(res, channel)
		}
	}

	*chl = res
	return nil
}

func unmarshalChannel(s interface{}) (Channel, error) {
	stuff, ok := s.(map[string]interface{})
	if !ok {
		return nil, errors.Errorf("Invalid type. Expected map[string]interface{}, got: %+v", stuff)
	}

	var address *addr.ProtoAddress

	if val, ok := stuff["address"]; ok {
		if k, ok := val.(string); ok {
			var err error
			address, err = addr.ParseAddress(k)
			if err != nil {
				return nil, errors.Wrapf(err, "Could not parse '%s' as a valid address!", k)
			}
		}
	}

	var channel Channel
	switch address.Scheme {
	case "socks":
		channel = &SocksChannel{}
	case "tcp", "unix", "unixpacket":
		channel = &NetworkChannel{}
	default:
		return nil, errors.Errorf("Unknown channel type: %s", address.Scheme)
	}

	data, err := json.Marshal(s)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed marshalling data: %v", s)
	}

	err = json.Unmarshal(data, channel)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed unmarshalling data: %v", string(data))
	}

	return channel, nil

}
