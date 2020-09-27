package client

import (
	"github.com/bokysan/socketace/v2/internal/streams"
	"github.com/pkg/errors"
	"io"
	"os"
)

type StdinClient struct {
	io.ReadWriteCloser
}

func NewStdInClientConnection() (*StdinClient, error) {
	var closer io.ReadWriteCloser
	closer = streams.NewReadWriteCloser(os.Stdin, os.Stdout)
	closer = streams.NewNamedStream(closer, "stdin")

	client, err := streams.NewProxyWrapperClient(closer)
	if err != nil {
		return nil, errors.Wrapf(err, "Could not open connection")
	}
	return &StdinClient{
		ReadWriteCloser: *client,
	}, nil
}
