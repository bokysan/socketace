package util

import (
	"fmt"
	ms "github.com/multiformats/go-multistream"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"io"
)

const BufferSize = 16384

// pipeData reads up to `BufferSize` data from the input stream and writes it dow to the output stream.
// It uses `io.CopyBuffer` method to do this. In case of an error (or end of stream) it will notify the
// provided channel and exit.
func pipeData(errs chan<- error, r io.Reader, w io.Writer) {
	var err error
	if writerTo, ok := r.(io.WriterTo); ok {
		_, err = writerTo.WriteTo(w)
	} else if readerFrom, ok := w.(io.ReaderFrom); ok {
		_, err = readerFrom.ReadFrom(r)
	} else {
		_, err = io.CopyBuffer(w, r, make([]byte, BufferSize))
	}

	if err != nil {
		errs <- err
	} else {
		errs <- io.EOF
	}
	return
}

// SourceToMultiplex will take a connection, and a multiplex channel. It will pick the single channel (based on
// the provided `name`) and pipe data between them.
func SourceToMultiplex(name string, connection io.ReadWriteCloser, multiplexChannel io.ReadWriteCloser) error {
	subProtocol := fmt.Sprintf("/%s", name)
	err := ms.SelectProtoOrFail(subProtocol, multiplexChannel)
	if err != nil {
		return errors.Wrapf(err,"Could no select protocol %s", subProtocol)
	}
	log.Debugf("Selected channel %v.", subProtocol)
	return PipeData(connection, multiplexChannel)
}

// PipeData does exactly what the name suggests, it pipes the data both ways -- from one reade to another writer
// and back. It closes the channel(s) on EOF or on errors.
func PipeData(down io.ReadWriteCloser, up io.ReadWriteCloser) error {
	pipe1 := make(chan error, 0)
	pipe2 := make(chan error, 0)
	go pipeData(pipe1, down, up)
	go pipeData(pipe2, up, down)

	select {
	case err := <-pipe1:
		log.Debugf("Closing upstream connection: %+v", up)
		up.Close()
		if err != io.EOF {
			log.Debugf("Closing downstream connection: %+v", down)
			down.Close()
			return err
		}
	case err := <-pipe2:
		log.Debugf("Closing downstream connection: %+v", down)
		down.Close()
		if err != io.EOF {
			log.Debugf("Closing upstream connection: %+v", up)
			up.Close()
			return err
		}
	}
	return nil
}