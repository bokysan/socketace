package dns

import (
	"github.com/bokysan/socketace/v2/internal/streams"
	"github.com/miekg/dns"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"io"
	"net"
	"time"
)

// OnMessage is a funnction that's called when a message is received
type OnMessage func(m *dns.Msg, remoteAddr net.Addr) (*dns.Msg, error)

// ServerCommunicator is an an interface to accept DNS requests from a multiconnection interface implementation.
type ServerCommunicator interface {
	io.Closer
	streams.Closed

	// Register a callback function that will be executed when a packet is received
	RegisterAccept(messageFunc OnMessage)

	// LocalAddr returns the local network address.
	LocalAddr() net.Addr
}

type NetConnectionServerCommunicator struct {
	closed    bool
	server    *dns.Server
	onMessage OnMessage
}

func NewNetConnectionServerCommunicator(server *dns.Server) (*NetConnectionServerCommunicator, error) {
	c := &NetConnectionServerCommunicator{
		server: server,
	}
	err := make(chan error, 0)

	go func() {
		err <- server.ListenAndServe()
	}()

	select {
	case e := <-err:
		return nil, e
	case <-time.After(1 * time.Second):
		// continue
	}

	dns.HandleFunc(".", c.handleRequest)
	return c, nil

}

func (n *NetConnectionServerCommunicator) handleRequest(w dns.ResponseWriter, r *dns.Msg) {
	var resp *dns.Msg
	var err error
	if n.onMessage != nil {
		resp, err = n.onMessage(r, w.RemoteAddr())
	}

	if err != nil {
		err = errors.WithStack(err)
		log.WithError(err).Errorf("Failed preparing response -- will not send anything back: %v", err)
		return
	}

	if r.IsTsig() != nil {
		if w.TsigStatus() == nil {
			// *Msg r has an TSIG record and it was validated
			resp.SetTsig("axfr.", dns.HmacMD5, 300, time.Now().Unix())
		} else {
			// *Msg r has an TSIG records and it was not validated
		}
	}

	if err = w.WriteMsg(resp); err != nil {
		err = errors.WithStack(err)
		log.WithError(err).Errorf("Failed writing response: %v", resp)
	}
}

func (n *NetConnectionServerCommunicator) Close() (err error) {
	if n.closed {
		return nil
	}
	if n.server.Listener != nil {
		err = n.server.Listener.Close()
	} else if n.server.PacketConn != nil {
		err = n.server.PacketConn.Close()
	}
	n.closed = true
	return err
}

func (n *NetConnectionServerCommunicator) Closed() bool {
	return n.closed
}

func (n *NetConnectionServerCommunicator) RegisterAccept(messageFunc OnMessage) {
	n.onMessage = messageFunc
}

func (n *NetConnectionServerCommunicator) LocalAddr() net.Addr {
	if n.server.PacketConn != nil {
		return n.server.PacketConn.LocalAddr()
	}
	return n.server.Listener.Addr()
}
