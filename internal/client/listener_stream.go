package client

import (
	"github.com/bokysan/socketace/v2/internal/util"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"net"
)

// StreamProtocolListener listens to stream data (e.g. TCP, unix socket, stdin...)
type StreamProtocolListener struct {
	netListener net.Listener
	listener    *Listener
	shutdown    chan bool
}

func (spl *StreamProtocolListener) Listen() (err error) {
	addr := spl.listener.Address.ProtoAddress
	spl.netListener, err = net.Listen(addr.Network, addr.Address)
	return
}

func (spl *StreamProtocolListener) Accept() {
	for {
		select {
		case <-spl.shutdown:
			return
		default:
			conn, err := spl.netListener.Accept()
			if err != nil {
				err = errors.Wrap(err, "Trouble accepting connection!")
				log.WithError(err).Errorf("Could not accept connection: %+v", err)
				continue
			}

			go spl.handleConnection(conn)
		}

	}
}

func (spl *StreamProtocolListener) Shutdown() (err error) {
	if spl.netListener != nil {
		spl.shutdown <- true
		err = errors.WithStack(spl.netListener.Close())
		spl.netListener = nil
	}
	return
}

func (spl *StreamProtocolListener) handleConnection(conn net.Conn) {
	// Try connecting directly first
	if spl.connectDirectly(conn) {
		return
	}

	mutex, err := createUpstreamConnectionMutex(spl.listener)
	if err = util.SourceToMultiplex(spl.listener.Name, conn, mutex); err != nil {
		log.WithError(err).Warnf("Communication for %s with upstream failed: %v", spl.listener.Name, err)
	}
	log.Trace("Closing upstream connection.")
	if err := mutex.Close(); err != nil {
		log.WithError(err).Tracef("Could not close upstream mutex: %v", err)
	}

	log.Trace("Closing client connection")
	if err := conn.Close(); err != nil {
		log.WithError(err).Tracef("Error closing client connection: %v", err)
	}
}

func (spl *StreamProtocolListener) connectDirectly(conn net.Conn) bool {
	forward := spl.listener.Forward
	proto := forward.Network
	addr := forward.Address
	if addr != "" && proto != "" {
		log.Debugf("Dialing direct connection to %s %s", proto, addr)
		upstream, err := net.Dial(proto, addr)
		if err == nil {
			err = util.PipeData(conn, upstream)
			if err != nil {
				err = errors.WithStack(err)
				log.WithError(err).Warnf("Error while communicating %s with %s %s: %+v",
					spl.listener.Name, proto, addr, err,
				)
			}
			return true
		}
	}
	return false
}
