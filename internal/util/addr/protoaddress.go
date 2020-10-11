package addr

import (
	"github.com/pkg/errors"
	"strings"
)

// ProtoAddress is a combination of network type and address.
type ProtoAddress struct {
	Network string `json:"network" short:"p" long:"network"   description:"Address network" choice:"tcp" choice:"unix" choice:"unixpacket"`
	Address string `json:"address"  short:"a" long:"address"  description:"Address IP and port, e.g. '192.168.8.0:22' or '/var/run/unix.sock'"`
}

// ProtoName defines the name so that we don't need to repeat it over and over again
type ProtoName struct {
	Name string `json:"name"     short:"n" long:"name"     description:"Unique endpoint name. Must match on the client and the server. E.g. 'ssh'."`
}

// String will combine the network with address in format <network>://<address>
func (p *ProtoAddress) String() string {
	if p.Network == "stdin" {
		return p.Network
	}

	return p.Network + "://" + p.Address
}

// ParseAddress does the reserse of ProtoAddress.String -- it will take a string and convert it
// an address.
func ParseAddress(a string) (ProtoAddress, error) {
	a = strings.TrimSpace(a)

	if a == "stdin" || a == "stdin:" {
		return ProtoAddress{
			Network: "stdin",
		}, nil
	}

	parts := strings.SplitN(a, "://", 2)
	if len(parts) != 2 {
		return ProtoAddress{}, errors.Errorf("Invalid address format: %v", a)
	}

	return ProtoAddress{
		Network: parts[0], Address: parts[1],
	}, nil
}
