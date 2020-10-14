package addr

import (
	"encoding/json"
	"github.com/pkg/errors"
	"net/url"
	"strings"
)

// ProtoAddress is a combination of network type and address.
type ProtoAddress struct {
	url.URL
	//	Network string `json:"network" short:"p" long:"network"   description:"Address network" choice:"tcp" choice:"unix" choice:"unixpacket"`
	//	Address string `json:"address"  short:"a" long:"address"  description:"Address IP and port, e.g. '192.168.8.0:22' or '/var/run/unix.sock'"`
}

// ProtoName defines the name so that we don't need to repeat it over and over again
type ProtoName struct {
	Name string `json:"name"     short:"n" long:"name"     description:"Unique endpoint name. Must match on the client and the server. E.g. 'ssh'."`
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
