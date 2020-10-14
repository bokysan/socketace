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
	"strconv"
	"strings"
)

// ServerConnection represents the server part of this app. It waits for the client announcement
// checks if versions match and disconnects it it can't establish a connection
type ServerConnection struct {
	streams.Connection
	ClientVersion     string
	negotiatedVersion string
	manager           cert.TlsConfig
	supportTls        bool
	secure            bool
	securityTech      string
}

// NewProxyWrapperServer will wait for client request and negotiate protocol version
func NewServerConnection(c net.Conn, manager cert.TlsConfig, secure bool) (*ServerConnection, error) {
	conn := streams.NewBufferedInputConnection(c)
	connection := &ServerConnection{
		manager: manager,
		secure:  secure,
	}
	if secure {
		connection.securityTech = SecurityUnderlying
	} else {
		connection.securityTech = SecurityNone
	}

	log.Debugf("[Server] SocketAce handshake...")
	if err := connection.handshake(conn); err != nil {
		return nil, errors.Wrapf(err, "Could not negotiate protocol version: %v", err)
	}

	log.Debugf("[Server] SocketAce upgrade...")
	if server, err := connection.upgrade(conn); err != nil {
		return nil, errors.Wrapf(err, "Could not upgrade connection: %v", err)
	} else {
		connection.Connection = server
	}

	return connection, nil
}

// Secure will return `true` is security has been enabled for this connection (either by encrypting the whole
// channel or by starting encryption using `StartTLS`
func (sc *ServerConnection) Secure() bool {
	return sc.secure
}

func (sc *ServerConnection) SecurityTech() string {
	return sc.securityTech
}

func (sc *ServerConnection) String() string {
	return fmt.Sprintf("%v[%v->%v], security=%v, proto=%v",
		sc.Connection,
		sc.Connection.LocalAddr(),
		sc.Connection.RemoteAddr(),
		sc.SecurityTech(),
		sc.negotiatedVersion,
	)
}

func (sc *ServerConnection) handshake(conn *streams.BufferedInputConnection) error {
	var request *Request
	var response *Response

	request = &Request{}
	if err := request.Read(conn.Reader); err != nil {
		response = &Response{
			Status:     strconv.Itoa(http.StatusBadRequest) + " Bad Request",
			StatusCode: http.StatusBadRequest,
			Headers:    make(textproto.MIMEHeader),
		}

		err := errors.Wrapf(err, "Failed parsing request: %v", err)

		response.Headers.Set("Server", "socketace/"+version.AppVersion())
		response.Headers.Set("Message", err.Error())

		if e := response.Write(conn); e != nil {
			log.WithError(e).Warnf("Could not write response: %v", e)
		}

		return err
	}
	log.Debugf("Client connected: %v", request.Headers.Get("User-Agent"))

	if request.Method != RequestMethod {
		response = &Response{
			Status:     strconv.Itoa(http.StatusMethodNotAllowed) + " Method Not Allowed",
			StatusCode: http.StatusMethodNotAllowed,
			Headers:    make(textproto.MIMEHeader),
		}
		response.Headers.Set("Server", "socketace/"+version.AppVersion())
		response.Headers.Set("Message", fmt.Sprintf("Invalid request method %v", request.Method))

		if err := response.Write(conn); err != nil {
			log.WithError(err).Warnf("Could not write response: %v", err)
		}

		return errors.Errorf("Invalid request method: %v - can't continue", request.Method)
	}

	response = &Response{
		Status:     "200 OK",
		StatusCode: 200,
		Headers:    make(textproto.MIMEHeader),
	}
	response.Headers.Set("Server", "socketace/"+version.AppVersion())

	capabilities := make([]string, 0)

	if !sc.secure {
		if sc.manager != nil {
			if c, err := sc.manager.GetTlsConfig(); err != nil {
				log.WithError(err).Warnf("Could not get X509 key pair, will not be able to advertise STARTTLS")
			} else if c != nil && c.Certificates != nil && len(c.Certificates) > 0 {
				sc.supportTls = true
				capabilities = append(capabilities, CapabilityStartTls)
			} else {
				log.Infof("No certificates, will not advrtise StartTLS.")
			}
		} else {
			log.Tracef("No certificate manager. Will not be able to secure the connection.")
		}
	} else {
		log.Debugf("Running over a secure connection, will not advertise StartTLS.")
	}

	acceptedProtocolVersions := request.Headers.Get(AcceptsProtocolVersion)
	sc.negotiatedVersion = sc.negotiateVersion(acceptedProtocolVersions)

	if sc.negotiatedVersion == "" {
		response.StatusCode = http.StatusConflict
		response.Status = strconv.Itoa(http.StatusConflict) + " Conflict"
		response.Headers.Set("Message",
			fmt.Sprintf("Could not negotiate protocol version. "+
				"Server supports: %v, client requires: %v", SupportedProtocolVersions, acceptedProtocolVersions),
		)

		if err := response.Write(conn); err != nil {
			log.WithError(err).Warnf("Could not write response: %v", err)
		}

		return errors.Errorf("Could not negoriate protocol version. "+
			"Server supports: %v, client requires: %v", SupportedProtocolVersions, acceptedProtocolVersions)
	}

	if len(capabilities) > 0 {
		response.Headers.Set(Capabilities, strings.Join(capabilities, ","))
	}

	response.Headers.Set("Protocol-Version", sc.negotiatedVersion)
	if err := response.Write(conn); err != nil {
		log.WithError(err).Warnf("Could not write response: %v", err)
	}

	return nil
}

func (sc *ServerConnection) upgrade(conn *streams.BufferedInputConnection) (streams.Connection, error) {
	request := &Request{}

	if err := request.Read(conn.Reader); err != nil {
		return nil, errors.Wrapf(err, "Failed parsing request!")
	}

	responseHeaders := make(textproto.MIMEHeader)
	responseHeaders.Set("Protocol-Version", sc.negotiatedVersion)
	responseHeaders.Set("Server", "socketace/"+version.AppVersion())

	response := &Response{
		Status:     "101 Switching Protocols",
		StatusCode: 101,
	}

	if request.Method != "GET" {
		response = &Response{
			Status:     strconv.Itoa(http.StatusMethodNotAllowed) + " Method Not Allowed",
			StatusCode: http.StatusMethodNotAllowed,
		}
		responseHeaders.Set("Message", fmt.Sprintf("Method %v not allowed in this context", request.Method))
	} else if strings.ToLower(request.Headers.Get("Connection")) != "upgrade" {
		response = &Response{
			Status:     strconv.Itoa(http.StatusNotAcceptable) + " Not acceptable",
			StatusCode: http.StatusNotAcceptable,
		}
		responseHeaders.Set("Message", "Only connection upgrade is allowed")
	} else if request.Headers.Get("Upgrade") != "socketace/"+sc.negotiatedVersion {
		response = &Response{
			Status:     strconv.Itoa(http.StatusNotAcceptable) + " Not acceptable",
			StatusCode: http.StatusNotAcceptable,
		}
		responseHeaders.Set("Message", fmt.Sprintf("Can't upgrade to protocol version %v", request.Headers.Get("Upgrade")))
	}

	if response.StatusCode != 101 {
		if err := response.Write(conn); err != nil {
			log.WithError(err).Warnf("Could not write response: %v", err)
		}
		return nil, errors.New(responseHeaders.Get("Message"))
	}

	responseHeaders.Set("Connection", "upgrade")
	responseHeaders.Set("Upgrade", "socketace/"+sc.negotiatedVersion)
	response.Headers = responseHeaders

	if strings.ToUpper(request.Headers.Get("Security")) == strings.ToUpper(CapabilityStartTls) {
		if sc.supportTls {
			config, err := sc.manager.GetTlsConfig()
			if err != nil {
				response = &Response{
					Status:     strconv.Itoa(http.StatusInternalServerError) + " Internal Server Error",
					StatusCode: http.StatusInternalServerError,
				}
				responseHeaders.Set("Message", err.Error())
				log.WithError(err).Errorf("Could not get TLS config: %v", err)

				if e := response.Write(conn); e != nil {
					log.WithError(e).Warnf("Could not write response: %v", e)
				}
				return nil, errors.WithStack(err)

			} else {
				if e := response.Write(conn); e != nil {
					streams.LogClose(conn)
					log.WithError(e).Warnf("Could not write response: %v", e)
				}

				log.Tracef("[Server] Executing TLS handshake")
				tlsConn := tls.Server(conn, config)
				if err := tlsConn.Handshake(); err != nil {
					log.WithError(err).Errorf("TLS handshake failed: %v", err)
					return nil, err
				}
				log.Debugf("[Server] Connection encrypted using TLS")

				sc.secure = true
				sc.securityTech = SecurityTls

				stream := streams.NewNamedConnection(tlsConn, "tls")
				return streams.NewNamedConnection(stream, "plain"), nil
			}
		} else {
			response = &Response{
				Status:     strconv.Itoa(http.StatusServiceUnavailable) + " Service Unavailable",
				StatusCode: http.StatusServiceUnavailable,
			}

			err := errors.Errorf("Client required StartTLS but not supported by the server!")
			responseHeaders.Set("Message", err.Error())
			log.WithError(err).Errorf(err.Error())

			if e := response.Write(conn); e != nil {
				log.WithError(e).Warnf("Could not write response: %v", e)
			}
			return nil, err

		}
	}

	if e := response.Write(conn); e != nil {
		streams.LogClose(conn)
		log.WithError(e).Warnf("Could not write response: %v", e)
	}

	return streams.NewNamedConnection(conn, "plain"), nil
}

// negotiateVersion will find the rpsion in the list of accepted client versions
func (sc *ServerConnection) negotiateVersion(acceptedVersions string) string {
	acceptedProtocolVersions := mime.SplitField(acceptedVersions)
	negotiatedVersion := ""

all:
	for _, v := range SupportedProtocolVersions {
		for _, k := range acceptedProtocolVersions {
			if v == k {
				negotiatedVersion = v
				break all
			}
		}
	}
	return negotiatedVersion
}
