package listener

import (
	"github.com/bokysan/socketace/v2/internal/streams"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"io"
	"os"
)

// StdInProtocolListener listens on standard input and writes to standard output
type StdInProtocolListener struct {
	listener *Listener
	shutdown chan bool
	upstream streams.ReadWriteCloserClosed
}

// Listen will directly open up standard input/output and connect to the remote server directly
func (spl *StdInProtocolListener) Listen() error {
	// Connect to the upstream server
	upstream, err := spl.listener.connector.Connect(spl.listener.config, spl.listener.Name)
	if err != nil {
		return errors.WithStack(err)
	}
	spl.upstream = streams.NewNamedStream(upstream, spl.listener.Address.String())

	return nil
}

func (spl *StdInProtocolListener) Accept() {
	var pipe io.ReadWriteCloser
	pipe = streams.NewReadWriteCloser(os.Stdin, os.Stdout)
	pipe = streams.NewNamedStream(pipe, "stdin")

	err := errors.WithStack(streams.PipeData(pipe, spl.upstream))
	if err != nil {
		log.WithError(err).Errorf("Error streaming data: %v", err)
	}
	os.Exit(0)
}

func (spl *StdInProtocolListener) Shutdown() error {
	return streams.LogClose(spl.upstream)
}
