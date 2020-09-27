package server

import (
	"fmt"
	"github.com/bokysan/socketace/v2/internal/util"
	"github.com/davecgh/go-spew/spew"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"net"
	"regexp"
)

// Channel is a configuration of one of the server that are going to be multiplexed in the connection
type Channel struct {
	util.ProtoAddress `yaml:",inline"`
	util.ProtoName `yaml:",inline"`
}

func (u *Channel) String() string {
	return fmt.Sprintf("%s:%s->%s", u.Name, u.Network, u.Address)
}

// OpenConnection will open a connection the the upstream server
func (u *Channel) OpenConnection() (net.Conn, error) {


	conn, err := net.Dial(u.Network, u.Address)
	if err != nil {
		err = errors.Wrapf(err,"Remote connection failed to %s->%s", u.Network, u.Address)
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
		return errors.Errorf("Channel %s does not match %s!", endpoint, ChannelRegex.String())
	}

	parts := ChannelRegex.FindAllStringSubmatch(endpoint, -1)[0]

	e := Channel{
		ProtoName: util.ProtoName{
			Name:     parts[0],
		},
		ProtoAddress: util.ProtoAddress{
			Network: parts[1],
			Address: parts[2],
		},
	}

	*epl = append(*epl, &e)

	return nil
}
