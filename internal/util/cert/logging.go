package cert

import (
	"crypto/tls"
	"github.com/bokysan/socketace/v2/internal/streams"
	log "github.com/sirupsen/logrus"
	"net"
)

func PrintPeerCertificates(conn net.Conn) {
	for {
		if tlsConn, ok := conn.(*tls.Conn); ok {
			state := tlsConn.ConnectionState()
			certChain := state.PeerCertificates
			cert := certChain[len(certChain)-1]

			log.Infof(
				"Peer certificate: ver=%v, serial=%v, subject=%v",
				cert.Version,
				cert.SerialNumber,
				cert.Subject,
			)
			return
		} else if wrapper, ok := conn.(streams.UnwrappedConnection); ok {
			conn = wrapper.Unwrap()
		} else {
			log.Tracef("%v not a valid TLS connection.", conn)
			return
		}
	}
}
