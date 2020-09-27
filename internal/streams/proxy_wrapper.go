package streams

import (
	"bufio"
	"fmt"
	"github.com/bokysan/socketace/v2/internal/util"
	"github.com/bokysan/socketace/v2/internal/version"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"io"
	"net/http"
	"net/textproto"
	"regexp"
	"strings"
)

const (
	MaxHeaderSize          = 4096
	HelloMessage           = "SOCKETACE"
	AcceptsProtocolVersion = "Accepts-Protocol-Version"
	UserAgent              = "User-Agent"
	Status                 = "Status"
)

var Splitter = regexp.MustCompile("\\s*,\\s*")
var SupportedProtocolVersions = []string{
	// Make sure this list is in descending order
	version.ProtocolVersion,
}

// ProxyWrapperClient represents a client to the socketace server. It announces the client
// checks the server and establishes the connection.
type ProxyWrapperClient struct {
	io.ReadWriteCloser
	ServerVersion string
}

// ProxyWrapperServer represents the server part of this app. It waits for the client announcement
// checks if versions match and disconnects it it can't establish a connection
type ProxyWrapperServer struct {
	io.ReadWriteCloser
	ClientVersion string
}

// NewProxyWrapperClient will create a connection an negotiate protocol support
func NewProxyWrapperClient(c io.ReadWriteCloser) (*ProxyWrapperClient, error) {
	conn := NewBufferedReadWriteCloser(c)

	headers := http.Header(make(textproto.MIMEHeader))
	headers.Set(AcceptsProtocolVersion, version.ProtocolVersion)
	headers.Set(UserAgent, "socketace/"+version.AppVersion())

	bufferedWriter := bufio.NewWriterSize(conn, util.BufferSize)
	log.Trace("Sending PING")
	if _, err := io.WriteString(conn, HelloMessage+" PING\r\n"); err != nil {
		return nil, errors.Wrapf(err, "Error sending HELLO")
	}

	if err := headers.Write(bufferedWriter); err != nil {
		return nil, errors.Wrapf(err, "Error sending headers")
	}
	if _, err := io.WriteString(bufferedWriter, "\r\n"); err != nil {
		return nil, errors.Wrapf(err, "Error ending headers")
	}
	if err := bufferedWriter.Flush(); err != nil {
		return nil, errors.WithStack(err)
	}

	log.Trace("Reading response")
	mime, err := parseHeader(conn.Reader)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed parsing response!")
	}
	log.Debugf("Connected to server %v: %v", mime.Get("Server"), mime)

	status := mime.Get(Status)
	if status == "200" {
		return &ProxyWrapperClient{
			ReadWriteCloser: conn,
		}, nil
	} else {
		return nil, errors.Errorf("Error from the server: %v", mime.Get("Message"))
	}
}

// NewProxyWrapperServer will wait for client request and negotiate protocol version
func NewProxyWrapperServer(c io.ReadWriteCloser) (*ProxyWrapperServer, error) {
	conn := NewBufferedReadWriteCloser(c)

	mime, err := parseHeader(conn.Reader)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed parsing request!")
	}
	log.Debugf("Client connected: %v", mime.Get("User-Agent"))

	headers := http.Header(make(textproto.MIMEHeader))
	headers.Set("Server", "socketace/"+version.AppVersion())

	acceptedProtocolVersions := Splitter.Split(mime.Get(AcceptsProtocolVersion), -1)
	// sort.SliceStable(acceptedProtocolVersions, func (i, j int) bool {
	// 	return semver.Compare(acceptedProtocolVersions[i], acceptedProtocolVersions[j]) > 0
	// })

	negotiatedVersion := ""
	status := "200"

all:
	for _, v := range SupportedProtocolVersions {
		for _, k := range acceptedProtocolVersions {
			if v == k {
				negotiatedVersion = v
				headers.Set("Protocol-Version", negotiatedVersion)
				break all
			}
		}
	}

	if negotiatedVersion == "" {
		status = "400"
		headers.Set("Message",
			fmt.Sprintf("Could not negotiate protocol version. "+
				"Server supports: %v, client requires: %v", SupportedProtocolVersions, acceptedProtocolVersions),
		)
	}

	headers.Set("Status", status)

	bufferedWriter := bufio.NewWriterSize(conn, util.BufferSize)
	log.Trace("Sending PONG")
	if _, err := io.WriteString(bufferedWriter, HelloMessage+" PONG\r\n"); err != nil {
		return nil, errors.Wrapf(err, "Error sending HELLO")
	}

	if err := headers.Write(bufferedWriter); err != nil {
		return nil, errors.Wrapf(err, "Error sending headers")
	}
	if _, err := io.WriteString(bufferedWriter, "\r\n"); err != nil {
		return nil, errors.Wrapf(err, "Error ending headers")
	}
	if err := bufferedWriter.Flush(); err != nil {
		return nil, errors.WithStack(err)
	}

	if status != "200" {
		return nil, errors.Wrapf(err, "Could not negotiate client connection")
	}

	return &ProxyWrapperServer{
		ReadWriteCloser: conn,
	}, nil
}

// parseHeader will read the SocketAce header from the stream and then pass over the stream to specific
// implementation.
func parseHeader(conn *bufio.Reader) (textproto.MIMEHeader, error) {
	headerReader := textproto.NewReader(conn)
	firstLine, err := headerReader.ReadLine()
	if err != nil {
		return nil, errors.WithStack(err)
	}
	if !strings.HasPrefix(firstLine, HelloMessage) {
		return nil, errors.Errorf("Expected helo to start with '%v', but got '%v'", HelloMessage, firstLine)
	}

	mime, err := headerReader.ReadMIMEHeader()
	if err != nil {
		return nil, errors.Wrapf(err, "Could not read MIME header!")
	}
	return mime, nil
}
