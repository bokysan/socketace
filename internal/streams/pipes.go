package streams

import (
	"fmt"
	"github.com/bokysan/socketace/v2/internal/util/buffers"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"io"
	"os"
	"strings"
)

type logWriter struct {
	Name func() string
}

func (l *logWriter) Write(p []byte) (n int, err error) {
	log.Debugf("%v: %v", l.Name(), string(p))
	return len(p), nil
}

// pipeDebugData will pipe data and output what's transferred to the log
func pipeDebugData(errs chan<- error, r io.Reader, w io.Writer) {
	reader := io.TeeReader(r, &logWriter{Name: func() string {
		return fmt.Sprintf("Read [%v]->%v", r, w)
	}})
	writer := io.MultiWriter(w, &logWriter{Name: func() string {
		return fmt.Sprintf("Wrote %v->[%v]", r, w)
	}})
	pipeData(errs, reader, writer)
}

// pipeData reads up to `BufferSize` data from the input stream and writes it dow to the output stream.
// It uses `io.CopyBuffer` method to do this. In case of an error (or end of stream) it will notify the
// provided channel and exit.
func pipeData(errs chan<- error, r io.Reader, w io.Writer) {
	var err error
	_, err = io.CopyBuffer(w, r, make([]byte, buffers.BufferSize))
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

	if err := closer.Close(); err != nil && !strings.Contains(err.Error(), " use of closed network connection") {
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
			return nil
		}
	}

	if err := closer.Close(); err != nil {
		err = errors.WithStack(err)
		log.WithError(err).Errorf("Could not close: %v", err)
		return err
	} else {
		// log.Tracef("%v successfully closed", closer)
		return nil
	}
}

// PipeData does exactly what the name suggests, it pipes the data both ways -- from one reade to another writer
// and back. It closes the channel(s) on EOF or on errors.
func PipeData(down io.ReadWriteCloser, up io.ReadWriteCloser) error {
	log.Debugf("Piping data %v <-> %v", down, up)

	downPipe := make(chan error, 0)
	upPipe := make(chan error, 0)

	if os.Getenv("SOCKETACE_PIPE_DEBUG") == "1" {
		go pipeDebugData(downPipe, down, up)
		go pipeDebugData(upPipe, up, down)

	} else {
		go pipeData(downPipe, down, up)
		go pipeData(upPipe, up, down)
	}

	select {
	case err := <-downPipe:
		log.Debugf("Closing piped upstream connection due to '%v': %+v", err, up)
		TryClose(up)
		if err != io.EOF {
			log.Debugf("Closing piped downstream connection: %+v", down)
			TryClose(down)
			return err
		}
	case err := <-upPipe:
		log.Debugf("Closing piped downstream connection due to '%v': %+v", err, down)
		TryClose(down)
		if err != io.EOF {
			log.Debugf("Closing piped upstream connection: %+v", up)
			TryClose(up)
			return err
		}
	}
	return nil
}
