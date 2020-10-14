package server

import (
	"github.com/bokysan/socketace/v2/internal/socketace"
	"github.com/bokysan/socketace/v2/internal/streams"
	"github.com/bokysan/socketace/v2/internal/util/buffers"
	"github.com/bokysan/socketace/v2/internal/util/cert"
	"github.com/multiformats/go-multistream"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/xtaci/smux"
	"io"
	"net"
	"strings"
)

func AcceptConnection(conn net.Conn, manager cert.TlsConfig, secure bool, channels Channels) error {
	log.Tracef("Establishing SocketAce connection...")
	server, err := socketace.NewServerConnection(conn, manager, secure)
	if err != nil {
		if !strings.Contains(err.Error(), "use of closed network connection") {
			log.WithError(err).Errorf("Could not negotiate connection: %v", err)
			if conn != nil {
				streams.TryClose(conn)
			}
		}
		return err
	}

	connectionHandler := &ConnectionHandler{
		channels: channels,
	}
	if err := connectionHandler.HandleConnection(server); err != nil {
		log.WithError(err).Errorf("Could not handle connection: %v", err)
		if conn != nil {
			streams.TryClose(conn)
		}
		return err
	}

	return nil
}

// ConnectionHandler will overlay a logical connection multiplexer over a pyhisical line
type ConnectionHandler struct {
	session  *smux.Session
	channels Channels
}

// Create a logical mutex session of a pyhisical link
func (ch *ConnectionHandler) HandleConnection(conn net.Conn) (err error) {
	config := smux.DefaultConfig()
	config.MaxFrameSize = buffers.BufferSize - 128
	ch.session, err = smux.Server(conn, config)

	if err != nil {
		if e := streams.LogClose(conn); e != nil {
			log.WithError(e).Errorf("Failed closing the connection: %+v", e)
		}
		return errors.WithStack(err)
	}

	go ch.acceptStream()

	return
}

// Accept a new logical stream and overlay endpoint selection on top of it
func (ch *ConnectionHandler) acceptStream() {
	for true {
		// Wait for next available stream
		var stream net.Conn

		stream, err := ch.session.AcceptStream()
		if err == io.ErrClosedPipe || err == io.EOF {
			log.Debugf("Stream closed, existing loop.")
			return
		} else if err != nil {
			log.WithError(err).Errorf("Error accepting stream: %v", err)
			continue
		}
		stream = streams.NewNamedConnection(stream, stream.RemoteAddr().String())
		log.Debugf("[Server] New logical connection accepted: %v", stream)

		if err = ch.multiplexToUpstream(stream); err != nil {
			log.WithError(err).Errorf("Error selecting multichannel stream: %v", err)
			streams.TryClose(stream)
		}
	}
}

func (ch *ConnectionHandler) muxHandler(protocol string, downstreamConnection io.ReadWriteCloser) error {
	for _, channel := range ch.channels {
		if protocol == "/"+channel.Name() {
			log.Debugf("[Upstream] Opening connection to upstream: %v", channel)
			upstreamConnection, err := channel.OpenConnection()
			if err != nil {
				return err
			}
			return streams.PipeData(downstreamConnection, upstreamConnection)
		}
	}
	return errors.Errorf("Uknown protocol %s", protocol)
}

// Create a multistream to let the client choose an appropriate solution
func (ch *ConnectionHandler) multiplexToUpstream(multiplexChannel net.Conn) error {
	mux := multistream.NewMultistreamMuxer()
	log.Tracef("[Server] Connection muxer created for %v", multiplexChannel)
	for _, u := range ch.channels {
		mux.AddHandler("/"+u.Name(), ch.muxHandler)
	}

	defer func() {
		if err := streams.LogClose(multiplexChannel); err != nil {
			log.WithError(err).Warnf("Failed closing the multiplex channel connection: %+v", err)
		}
	}()

	log.Tracef("[Server] Handle channel %v", multiplexChannel)
	if err := mux.Handle(multiplexChannel); err != nil {
		err = errors.Wrapf(err, "Could not handle multiplex channel: %+v", err)
		return err
	}
	return nil
}
