package dns

import (
	"github.com/bokysan/socketace/v2/internal/streams"
	"github.com/miekg/dns"
	"io"
	"net"
)

// OnMessage is a funnction that's called when a message is received
type OnMessage func(m *dns.Msg) (*dns.Msg, error)

// ServerCommunicator is an an interface to accept DNS requests from a multiconnection interface implementation.
type ServerCommunicator interface {
	io.Closer
	streams.Closed

	// Register a callback function that will be executed when a packet is received
	RegisterAccept(messageFunc OnMessage)

	// LocalAddr returns the local network address.
	LocalAddr() net.Addr
}
