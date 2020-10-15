package addr

import (
	"encoding/json"
	"github.com/pkg/errors"
	"net"
	"net/url"
	"strings"
)

// ProtoAddress is a combination of network type and address.
type ProtoAddress struct {
	url.URL
	//	Network string `json:"network" short:"p" long:"network"   description:"Address network" choice:"tcp" choice:"unix" choice:"unixpacket"`
	//	Address string `json:"address"  short:"a" long:"address"  description:"Address IP and port, e.g. '192.168.8.0:22' or '/var/run/unix.sock'"`
}

type netAddress struct {
	*ProtoAddress
}

// ProtoName defines the name so that we don't need to repeat it over and over again
type ProtoName struct {
	Name string `json:"name"     short:"n" long:"name"     description:"Unique endpoint name. Must match on the client and the server. E.g. 'ssh'."`
}

func (pa *ProtoAddress) Addr() (net.Addr, error) {
	switch pa.Scheme {
	case "udp", "udp4", "udp6":
		return net.ResolveUDPAddr(pa.Scheme, pa.Host)
	case "unix", "unixgram", "unixpacket":
		return net.ResolveUnixAddr(pa.Scheme, pa.Host)
	case "unix+tls", "unixpacket+tls":
		return net.ResolveUnixAddr(PlusEnd.ReplaceAllString(pa.Scheme, ""), pa.Host)
	case "tcp", "tpc4", "tcp6":
		return net.ResolveTCPAddr(pa.Scheme, pa.Host)
	case "tcp+tls", "tpc4+tls", "tcp6+tls":
		return net.ResolveTCPAddr(PlusEnd.ReplaceAllString(pa.Scheme, ""), pa.Host)
	default:
		return &netAddress{pa}, nil
	}
}

func (pa *ProtoAddress) UnmarshalFlag(s string) error {
	s = strings.TrimSpace(s)
	p, err := url.Parse(s)
	if err != nil {
		return errors.Wrapf(err, "Cannot parse %q", s)
	}
	pa.URL = *p
	return nil
}

func (pa *ProtoAddress) UnmarshalJSON(b []byte) error {
	var stuff string
	if err := json.Unmarshal(b, &stuff); err != nil {
		return errors.WithStack(err)
	}
	return pa.UnmarshalFlag(stuff)
}

func (na *netAddress) Network() string {
	// name of the network (for example, "tcp", "udp")
	return na.Scheme
}

func (na *netAddress) String() string {
	// string form of address (for example, "192.0.2.1:25", "[2001:db8::1]:80")
	if na.Host != "" {
		return na.Host
	} else {
		return na.Path
	}
}

// ParseAddress does the reserse of ProtoAddress.String -- it will take a string and convert it
// an address.
func ParseAddress(addr string) (*ProtoAddress, error) {
	pa := &ProtoAddress{}

	err := pa.UnmarshalFlag(addr)
	if err != nil {
		return nil, err
	}
	return pa, nil
}

// MustParseAddress will parse the address and panic if it can't
func MustParseAddress(addr string) ProtoAddress {
	pa, err := ParseAddress(addr)
	if err != nil {
		panic(err)
	}
	return *pa
}
