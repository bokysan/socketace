package socketace

import (
	"crypto/tls"
	"fmt"
	"github.com/bokysan/socketace/v2/internal/streams"
	"github.com/bokysan/socketace/v2/internal/util/cert"
	"github.com/bokysan/socketace/v2/internal/util/mime"
	"github.com/bokysan/socketace/v2/internal/version"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"net"
	"net/http"
	"net/textproto"
	"strings"
)

// ClientConnection represents a client to the socketace server. It announces the client to the server,
// checks the server and establishes the connection.
type ClientConnection struct {
	streams.Connection
	ServerVersion     string
	negotiatedVersion string
	capabilities      []string
	manager           cert.TlsConfig
	host              string
	secure            bool
	securityTech      string
}

// NewClientConnection will create a connection and negotiate the protocol and TLS security
func NewClientConnection(c net.Conn, manager cert.TlsConfig, secure bool, host string) (*ClientConnection, error) {
	conn := streams.NewBufferedInputConnection(c)
	connection := &ClientConnection{
		manager: manager,
		host:    host,
		secure:  secure,
	}
	if secure {
		connection.securityTech = SecurityUnderlying
	} else {
		connection.securityTech = SecurityNone
	}

	log.Debugf("[Client] SocketAce handshake...")
	if err := connection.handshake(conn); err != nil {
		return nil, errors.Wrapf(err, "Could not negotiate protocol version: %v", err)
	}

	shouldStartTls := !secure && connection.containsCapability(connection.capabilities, CapabilityStartTls)

	log.Debugf("[Client] SocketAce upgrade...")
	if client, err := connection.upgrade(conn, shouldStartTls, secure); err != nil {
		return nil, errors.Wrapf(err, "Could not upgrade connection: %v", err)
	} else {
		connection.Connection = client
	}

	return connection, nil
}

func (cc *ClientConnection) String() string {
	return fmt.Sprintf("%v[%v->%v], security=%v, proto=%v",
		cc.Connection,
		cc.Connection.LocalAddr(),
		cc.Connection.RemoteAddr(),
		cc.SecurityTech(),
		cc.negotiatedVersion,
	)
}

// Secure will return `true` is security has been enabled for this connection (either by encrypting the whole
// channel or by starting encryption using `StartTLS`
func (cc *ClientConnection) Secure() bool {
	return cc.secure
}

func (cc *ClientConnection) SecurityTech() string {
	return cc.securityTech
}

// handshake will call the server and execute the handshake with the server and try to negotiate the capabilities
// and the protocol version.
func (cc *ClientConnection) handshake(conn *streams.BufferedInputConnection) (err error) {
	var request *Request
	var response *Response

	// QueryDns the intital requests
	request = &Request{
		Method:  RequestMethod,
		URL:     "/",
		Headers: make(textproto.MIMEHeader),
	}
	request.Headers.Set(AcceptsProtocolVersion, version.ProtocolVersion)
	request.Headers.Set(UserAgent, "socketace/"+version.AppVersion())
	if err := request.Write(conn); err != nil {
		return errors.Wrapf(err, "Coud not send intial request")
	}

	// Parse the response
	response = &Response{}
	if err := response.Read(conn.Reader); err != nil {
		return errors.Wrapf(err, "Coud not get response to initial request")
	}

	if response.StatusCode != http.StatusOK {
		return errors.Errorf("Server refused our request with error: %v", response.StatusCode)
	}

	cc.negotiatedVersion = response.Headers.Get("Protocol-Version")
	cc.capabilities = mime.SplitField(strings.ToUpper(response.Headers.Get(Capabilities)))

	return
}

// upgrade will upgrade the connection and negotiate the security protocol
func (cc *ClientConnection) upgrade(conn *streams.BufferedInputConnection, shouldStartTls bool, secure bool) (streams.Connection, error) {

	// Upgrade the connection now
	request := &Request{
		Method:  "GET",
		URL:     "/",
		Headers: make(textproto.MIMEHeader),
	}
	request.Headers.Set(UserAgent, "socketace/"+version.AppVersion())
	request.Headers.Set("Upgrade", "socketace/"+cc.negotiatedVersion)
	request.Headers.Set("Connection", "upgrade")

	if shouldStartTls {
		request.Headers.Set("Security", CapabilityStartTls)
	}

	if err := request.Write(conn); err != nil {
		return nil, errors.Wrapf(err, "Coud not send upgrade request")
	}

	// Expect the 101 Switching protocols
	response := &Response{}
	if err := response.Read(conn.Reader); err != nil {
		return nil, errors.Wrapf(err, "Could not get response to upgrade request")
	}

	if response.StatusCode != http.StatusSwitchingProtocols {
		return nil, errors.Errorf("Server refused our request with error: %v", response.StatusCode)
	}

	if shouldStartTls {
		client, err := cc.startTls(conn)
		if err != nil {
			streams.TryClose(conn)
			err = errors.Errorf("Could not establish a secure connection: %v", err)
			return nil, err
		}
		log.Infof("[Client] (Secure) Connected StartTLS-secured to server %v at %v", response.Headers.Get("Server"), client.RemoteAddr().String())

		cc.secure = true
		cc.securityTech = SecurityTls
		return streams.NewNamedConnection(client, "tls"), err
	}

	if secure {
		log.Infof("[Client] (Secure) Connected (non-StartTLS) to server %v at %v.", response.Headers.Get("Server"), conn.RemoteAddr().String())
	} else {
		log.Warnf("[Client] (Insecure) Connected (non-StartTLS) to server %v at %v.", response.Headers.Get("Server"), conn.RemoteAddr().String())
	}
	return streams.NewNamedConnection(conn, "plain"), nil
}

// startTls will start a TLS over the given connection
func (cc *ClientConnection) startTls(conn streams.Connection) (streams.Connection, error) {
	var tlsConfig *tls.Config

	if cc.manager != nil {
		if conf, err := cc.manager.GetTlsConfig(); err != nil {
			return nil, errors.Wrapf(err, "Could not get TLS config: %v", err)
		} else {
			tlsConfig = conf
		}
	} else {
		tlsConfig = &tls.Config{}
	}
	tlsConfig.ServerName = cc.host

	log.Tracef("[Client] Executing TLS handshake")
	tlsConn := tls.Client(conn, tlsConfig)
	if err := tlsConn.Handshake(); err != nil {
		return nil, errors.Wrapf(err, "StartTLS handshake failed: %v", err)
	}
	log.Debugf("[Client] Connection encrypted using TLS")
	cert.PrintPeerCertificates(tlsConn)
	return streams.NewNamedConnection(tlsConn, "tls"), nil
}

// containsCapability checks if the provided slice contains the selected capability (case-insensitive)
func (cc *ClientConnection) containsCapability(caps []string, cap string) bool {
	cap = strings.ToUpper(cap)
	for _, c := range caps {
		if strings.ToUpper(c) == cap {
			return true
		}
	}
	return false
}
