package addr

import (
	"github.com/pkg/errors"
	"net"
)

// ResolveHostAddress will take an address as a string and try to parse it into a net.TCPAddr using
// `net.ResolveTCPAddr`. If unsuccessful, it will wrap an error and return it.
func ResolveHostAddress(addr string) (*net.TCPAddr, error) {
	address, err := net.ResolveTCPAddr("tcp", addr)
	err = errors.Wrapf(err, "Could not deduct host and port from %v: %+v", addr, err)
	return address, err
}
