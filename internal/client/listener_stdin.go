package client

import (
	"fmt"
	"github.com/bokysan/socketace/v2/internal/streams"
	"github.com/bokysan/socketace/v2/internal/util"
	ms "github.com/multiformats/go-multistream"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"io"
	"os"
)

// StdInProtocolListener listens on standard input and writes to standard output
type StdInProtocolListener struct {
	listener *Listener
	shutdown chan bool
	upstream io.ReadWriteCloser
}

func (spl *StdInProtocolListener) Listen() error {
	mutex, err := createUpstreamConnectionMutex(spl.listener)
	if err != nil {
		return errors.WithStack(err)
	}
	subProtocol := fmt.Sprintf("/%s", spl.listener.Name)
	err = ms.SelectProtoOrFail(subProtocol, mutex)
	if err != nil {
		return errors.Wrapf(err, "Could no select protocol %s", subProtocol)
	}

	spl.upstream = streams.NewNamedStream(mutex, "ssh->" + spl.listener.Address.String())

	return nil
}

func (spl *StdInProtocolListener) Accept() {
	var pipe io.ReadWriteCloser
	pipe = streams.NewReadWriteCloser(os.Stdin, os.Stdout)
	pipe = streams.NewNamedStream(pipe, "stdin")

	err := errors.WithStack(util.PipeData(pipe, spl.upstream))
	if err != nil {
		log.WithError(err).Errorf("Error streaming data: %v", err)
	}
	os.Exit(0)
}

func (spl *StdInProtocolListener) Shutdown() error {
	return spl.upstream.Close()
}
