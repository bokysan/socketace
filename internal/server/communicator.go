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

func AcceptConnection(conn net.Conn, manager cert.TlsConfig, secure bool, upstreams ChannelList) error {
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
		upstreams: upstreams,
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
	session   *smux.Session
	upstreams ChannelList
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
		stream, err := ch.session.AcceptStream()

		if err == io.ErrClosedPipe || err == io.EOF {
			log.Debugf("Stream closed, existing loop.")
			return
		} else if err != nil {
			log.WithError(err).Errorf("Error accepting stream: %v", err)
			continue
		}

		if err = ch.multiplexToUpstream(stream, ch.upstreams); err != nil {
			log.WithError(err).Errorf("Error selecting multichannel stream: %v", err)
			streams.TryClose(stream)
		}
	}
}

func (ch *ConnectionHandler) addMuxHandler(mux *multistream.MultistreamMuxer, upstream *Channel) {
	handler := func(protocol string, downstreamConnection io.ReadWriteCloser) error {
		switch upstream.Network {
		case "udp", "udp4", "udp6", "unixgram":
			return errors.Errorf("Packet connections (%v) are not yet supported", upstream.Network)
		}

		log.Debugf("Opening connection to upstream: %v", upstream)
		upstreamConnection, err := upstream.OpenConnection()
		if err != nil {
			return err
		}

		return streams.PipeData(downstreamConnection, upstreamConnection)
	}
	mux.AddHandler("/"+upstream.Name, handler)
}

// Create a multistream to let the client choose an appropriate solution
func (ch *ConnectionHandler) multiplexToUpstream(multiplexChannel io.ReadWriteCloser, upstreams ChannelList) error {
	mux := multistream.NewMultistreamMuxer()
	for _, u := range upstreams {
		ch.addMuxHandler(mux, u)
	}

	defer func() {
		if err := streams.LogClose(multiplexChannel); err != nil {
			log.WithError(err).Warnf("Failed closing the multiplex channel connection: %+v", err)
		}
	}()

	if err := mux.Handle(multiplexChannel); err != nil {
		err = errors.Wrapf(err, "Could not handle multiplex channel: %+v", err)
		return err
	}
	return nil
}
