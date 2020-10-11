package streams

import "regexp"

var PlusEnd = regexp.MustCompile("\\+.+$")
var HasTls = regexp.MustCompile("\\+tls")

type StandardIOAddress struct {
	Address string
}

func (StandardIOAddress) Network() string {
	return "std"
}

func (s *StandardIOAddress) String() string {
	return s.Address
}
