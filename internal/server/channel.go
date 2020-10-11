package server

import (
	"fmt"
	"github.com/bokysan/socketace/v2/internal/util/addr"
	"github.com/davecgh/go-spew/spew"
	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"net"
	"regexp"
)

// Channel is a configuration of one of the server that are going to be multiplexed in the connection
type Channel struct {
	addr.ProtoAddress `yaml:",inline"`
	addr.ProtoName    `yaml:",inline"`
}

func (u *Channel) String() string {
	return fmt.Sprintf("%s:%s->%s", u.Name, u.Network, u.Address)
}

// OpenConnection will open a connection the the upstream server
func (u *Channel) OpenConnection() (net.Conn, error) {

	conn, err := net.Dial(u.Network, u.Address)
	if err != nil {
		err = errors.Wrapf(err, "Remote connection failed to %s->%s", u.Network, u.Address)
		log.WithError(err).Errorf("Could not connect to %s->%s: %+v", u.Network, u.Address, err)
	}
	return conn, err
}

// ------ // ------ // ------ // ------ // ------ // ------ // ------ //

var ChannelRegex = regexp.MustCompile("^(/[a-z0-9_^/]*)->(tcp|udp|unix|unixgram|unixpacket):(.*)$")

type ChannelList []*Channel

func (epl *ChannelList) String() string {
	return spew.Sdump(epl)
}

// Filter will return a list of channels if they are contained in the list of names. It will throw an error
// if no channels can be identified (either this list is empty or no match if found). If the list of names
// is empty or nil, it will return all the channels
func (epl *ChannelList) Filter(names []string) (ChannelList, error) {

	if names == nil || len(names) == 0 {
		return *epl, nil
	}

	var errs error

	upstreams := make(ChannelList, 0)
	for _, ch := range names {
		upstream, err := epl.Find(ch)
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
func (epl *ChannelList) Find(name string) (*Channel, error) {
	var available []string
	for _, e := range *epl {
		if e.Name == name {
			return e, nil
		}

		available = append(available, e.Name)
	}
	return nil, errors.Errorf("Could not find endpoint with name: '%s' among: %v", name, available)
}

func (epl *ChannelList) UnmarshalFlag(endpoint string) error {

	if !ChannelRegex.MatchString(endpoint) {
		return errors.Errorf("Channel '%s' does not match %s!", endpoint, ChannelRegex.String())
	}

	parts := ChannelRegex.FindAllStringSubmatch(endpoint, -1)[0]

	e := Channel{
		ProtoName: addr.ProtoName{
			Name: parts[0],
		},
		ProtoAddress: addr.ProtoAddress{
			Network: parts[1],
			Address: parts[2],
		},
	}

	*epl = append(*epl, &e)

	return nil
}
