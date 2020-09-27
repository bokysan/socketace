package server

import (
	"github.com/bokysan/socketace/v2/internal/streams"
	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"io"
	"os"
)

type StdInServer struct {
	Kind     string   `json:"kind"`
	Channels []string `json:"channels"`

	connection io.ReadWriteCloser
	service    *Service
}

func NewStdinServer() *StdInServer {
	return &StdInServer{
	}
}

func (st *StdInServer) String() string {
	return "stdin"
}

func (st *StdInServer) SetService(service *Service) {
	st.service = service
}

//goland:noinspection GoUnusedParameter
func (st *StdInServer) Execute(args []string) error {
	log.Infof("Starting stdin server...")

	var errs error
	upstreams := make(ChannelList, 0)
	if st.Channels == nil || len(st.Channels) == 0 {
		upstreams = st.service.Channels
	} else {
		for _, ch := range st.Channels {
			upstream, err := st.service.Channels.Find(ch)
			if err != nil {
				errs = multierror.Append(errs, errors.WithStack(err))
				continue
			}
			upstreams = append(upstreams, upstream)
		}
	}

	if len(upstreams) == 0 {
		errs = multierror.Append(errs, errors.Errorf("No upstreams defined for endpoint server"))
	}

	if errs != nil {
		return errors.WithStack(errs)
	}

	var conn io.ReadWriteCloser
	conn = streams.NewReadWriteCloser(os.Stdin, os.Stdout)
	conn = streams.NewNamedStream(conn, "stdin")

	client, err := streams.NewProxyWrapperServer(conn)
	if err != nil {
		return errors.WithStack(err)
	}
	st.connection = client

	if err := MultiplexToUpstream(st.connection, upstreams); err != nil {
		return errors.WithStack(err)
	}

	return nil
}

func (st *StdInServer) Shutdown() error {
	if st.connection != nil {
		err := st.connection.Close()
		return err
	} else {
		return nil
	}
}
