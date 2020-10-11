package listener

import (
	"github.com/bokysan/socketace/v2/internal/streams"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"net"
	"strings"
)

// StreamProtocolListener listens to stream data (e.g. TCP, unix socket, stdin...)
type StreamProtocolListener struct {
	netListener net.Listener
	listener    *Listener
	shutdown    chan bool
}

func (spl *StreamProtocolListener) Listen() (err error) {
	addr := spl.listener.Address.ProtoAddress
	spl.shutdown = make(chan bool, 1)
	spl.netListener, err = net.Listen(addr.Network, addr.Address)
	return
}

func (spl *StreamProtocolListener) String() string {
	if spl.netListener == nil {
		return "<unknown>"
	} else {
		return spl.netListener.Addr().String()
	}
}

func (spl *StreamProtocolListener) Accept() {
	for {
		select {
		case <-spl.shutdown:
			return
		default:
			conn, err := spl.netListener.Accept()
			if err != nil {
				if strings.Contains(err.Error(), "use of closed network connection") {
					return
				}
				err = errors.Wrap(err, "Trouble accepting connection!")
				log.WithError(err).Errorf("Could not accept connection: %+v", err)
				continue
			}
			// log.Debugf("Acceping connection on %p = %v -> %p %v", spl, spl, spl.listener, spl.listener)
			go spl.handleConnection(conn)
		}

	}
}

func (spl *StreamProtocolListener) Shutdown() (err error) {
	if spl.netListener != nil {
		spl.shutdown <- true
		err = errors.WithStack(streams.LogClose(spl.netListener))
		spl.netListener = nil
	}
	return
}

func (spl *StreamProtocolListener) handleConnection(conn net.Conn) {
	// Try connecting directly first
	if spl.connectDirectly(conn) {
		return
	}
	upstream, err := spl.listener.connector.Connect(spl.listener.config, spl.listener.Name)
	if err != nil {
		log.WithError(err).Warnf("Communication for %s with upstream failed: %v", spl.listener.Name, err)
	} else {
		upstream = streams.NewNamedStream(upstream, spl.listener.Name)
		stream := streams.NewNamedStream(conn, conn.RemoteAddr().String())
		err = streams.PipeData(stream, upstream)
		if err != nil {
			log.WithError(err).Warnf("Communication for %s with upstream failed: %v", spl.listener.Name, err)
		}
	}

	log.Trace("Closing upstream connection.")
	streams.TryClose(upstream)
	streams.TryClose(conn)
}

func (spl *StreamProtocolListener) connectDirectly(conn net.Conn) bool {
	forward := spl.listener.Forward
	proto := forward.Network
	addr := forward.Address
	if addr != "" && proto != "" {
		log.Debugf("Dialing direct connection to %s %s", proto, addr)
		var upstream net.Conn
		upstream, err := net.Dial(proto, addr)
		if err == nil {
			upstream = streams.NewNamedConnection(upstream, proto+"://"+addr)
			err = streams.PipeData(conn, upstream)
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
