package client

import (
	"github.com/pkg/errors"
	"net/url"
)

type UpstreamServer struct {
	// Address is the string representation of the address, as specified by the client
	Address string
	// addr is the parsed representation of the address and calculated automatically while unmarshalling
	addr *url.URL
}

func (us *UpstreamServer) Addr() *url.URL {
	return us.addr
}

// ------ // ------ // ------ // ------ // ------ // ------ // ------ //

type UpstreamList []UpstreamServer

func (ul *UpstreamList) UnmarshalFlag(endpoint string) error {
	address, err := url.Parse(endpoint)
	err = errors.Wrapf(err, "Invalid URL: %s", endpoint)
	if err != nil {
		return err
	}

	*ul = append(*ul, UpstreamServer{
		Address: endpoint,
		addr: address,
	})

	return nil
}

