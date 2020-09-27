package client

import (
	"fmt"
	"github.com/bokysan/socketace/v2/internal/util/packet"
	ms "github.com/multiformats/go-multistream"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"net"
)

// PacketProtocolListener listens to packet data (e.g. UDP, unixgram)
type PacketProtocolListener struct {
	netListener        net.PacketConn
	listener           *Listener
	shutdown           chan bool
	isShutdown         bool
	upstreamConnection net.PacketConn
}

func (ppl *PacketProtocolListener) Listen() (err error) {
	addr := ppl.listener.Address.ProtoAddress
	ppl.netListener, err = net.ListenPacket(addr.Network, addr.Address)

	if err != nil {
		mutex, err := createUpstreamConnectionMutex(ppl.listener)
		if err != nil {
			return errors.WithStack(err)
		}
		subProtocol := fmt.Sprintf("/%s", ppl.listener.Name)
		err = ms.SelectProtoOrFail(subProtocol, mutex)
		if err != nil {
			return errors.Wrapf(err, "Could no select protocol %s", subProtocol)
		}
		ppl.upstreamConnection = packet.NewUpstreamConnection(mutex)
	}

	return
}

func (ppl *PacketProtocolListener) Accept() {
	for {
		select {
		case <-ppl.shutdown:
			return
		default:
			go ppl.handleRequest()
			go ppl.handleResponse()
		}
	}
}

// handleRequest will listen for UDP packets and forward them to upstream
func (ppl *PacketProtocolListener) handleRequest() {
	for !ppl.isShutdown {
		select {
		case <-ppl.shutdown:
			ppl.isShutdown = true
			return
		default:
			data := make([]byte, packet.BufferSize)
			n, addr, err := ppl.netListener.ReadFrom(data)
			if err != nil {
				log.WithError(err).Errorf("Error reading input packet: %v", err)
				continue
			}
			_, err = ppl.upstreamConnection.WriteTo(data[:n], addr)
			if err != nil {
				log.WithError(err).Errorf("Error writing to upstream: %v", err)
			}
		}
	}
}

// handleResponse will listen for packets from upstream and resend them downwards
func (ppl *PacketProtocolListener) handleResponse() {
	for !ppl.isShutdown {
		select {
		case <-ppl.shutdown:
			ppl.isShutdown = true
			return
		default:
			data := make([]byte, packet.BufferSize)
			n, addr, err := ppl.netListener.ReadFrom(data)
			if err != nil {
				log.WithError(err).Errorf("Error reading upstream packet: %v", err)
				continue
			}
			network := ppl.listener.Address.ProtoAddress.Network
			conn, err := net.Dial(network, addr.String())
			if err != nil {
				log.WithError(err).Errorf("Error connecting to downstream %v: %v", addr, err)
				continue
			}
			_, err = conn.Write(data[:n])
			if err != nil {
				log.WithError(err).Errorf("Error writting to downstream %v: %v", addr, err)
				continue
			}
		}
	}
}

func (ppl *PacketProtocolListener) Shutdown() (err error) {
	if ppl.netListener != nil {
		ppl.shutdown <- true
		err = errors.WithStack(ppl.netListener.Close())
		ppl.netListener = nil
	}
	return
}
