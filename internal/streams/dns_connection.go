package streams

import "net"

type DnsPacketConnection struct {
	net.PacketConn
	name   string
	closed bool
}

type DnsServerPacketConnection struct {
	DnsPacketConnection
}

type DnsClientPacketConnection struct {
	DnsPacketConnection
}

func NewDnsServerPacketConnection(wrapped net.PacketConn, name string) *DnsServerPacketConnection {
	return &DnsServerPacketConnection{
		DnsPacketConnection: DnsPacketConnection{
			PacketConn: wrapped,
			name:       name,
		},
	}
}

func NewDnsClientPacketConnection(wrapped net.PacketConn, name string) *DnsClientPacketConnection {
	return &DnsClientPacketConnection{
		DnsPacketConnection: DnsPacketConnection{
			PacketConn: wrapped,
			name:       name,
		},
	}
}

// Close will close the underlying stream. If the Close has already been called, it will do nothing
func (ns *DnsPacketConnection) Close() error {
	if ns.closed {
		return nil
	}
	err := LogClose(ns.PacketConn)
	ns.closed = true

	return err
}

// Closed will return `true` if SafeStream.Close has been called at least once
func (ns *DnsPacketConnection) Closed() bool {
	return ns.closed
}

// Unwrap returns the embedded net.PacketConn
func (ns *DnsPacketConnection) Unwrap() net.PacketConn {
	return ns.PacketConn
}
