package streams

import (
	"github.com/bokysan/socketace/v2/internal/util/buffers"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"io"
)

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
		_, err = io.CopyBuffer(w, r, make([]byte, buffers.BufferSize))
	}

	if err != nil {
		errs <- err
	} else {
		errs <- io.EOF
	}
	return
}

// Try closing a connection and just report to log if it fails
func TryClose(closer io.Closer) {
	if closer == nil {
		return
	}

	if c, ok := closer.(Closed); ok {
		if c.Closed() {
			return
		}
	}

	if err := closer.Close(); err != nil {
		err = errors.WithStack(err)
		log.WithError(err).Errorf("Could not close stream: %v", err)
	}
}

// LogClose will log when shutting wodn a connection
func LogClose(closer io.Closer) error {
	if closer == nil {
		return nil
	}

	if c, ok := closer.(Closed); ok {
		if c.Closed() {
			log.Tracef("%v already closed", closer)
			return nil
		}
	}

	if err := closer.Close(); err != nil {
		err = errors.WithStack(err)
		log.WithError(err).Errorf("Could not close: %v", err)
		return err
	} else {
		// log.Tracef("%v succesfully closed", closer)
		return nil
	}
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
		log.Debugf("Closing piped upstream connection: %+v", up)
		TryClose(up)
		if err != io.EOF {
			log.Debugf("Closing piped downstream connection: %+v", down)
			TryClose(down)
			return err
		}
	case err := <-pipe2:
		log.Debugf("Closing piped downstream connection: %+v", down)
		TryClose(down)
		if err != io.EOF {
			log.Debugf("Closing piped upstream connection: %+v", up)
			TryClose(up)
			return err
		}
	}
	return nil
}
