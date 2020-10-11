package socketace

import (
	"bufio"
	"bytes"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"io"
	"net/http"
	"net/textproto"
	"strings"
)

type Request struct {
	Method  string // e.g. "X-SOCKETACE"
	URL     string // e.g. "/"
	Proto   string // e.g. "HTTP/1.0"
	Headers textproto.MIMEHeader
}

// prepareFirstLine will prepare the request line
func (sar *Request) prepareFirstLine() string {
	command := ""
	if sar.Method != "" {
		command += sar.Method
	} else {
		command += RequestMethod
	}
	if sar.URL != "" {
		command += " " + sar.URL
	} else {
		command += " /"
	}
	if sar.Proto != "" {
		command += " " + sar.Proto
	} else {
		command += " HTTP/1.1"
	}
	return command
}

// parseRequestLine parses "GET /foo HTTP/1.1" into its three parts.
func (sar *Request) parseRequestLine(line string) (method, requestURI, proto string, err error) {
	s1 := strings.Index(line, " ")
	s2 := strings.Index(line[s1+1:], " ")
	if s1 < 0 || s2 < 0 {
		return "", "", "", errors.Errorf("Invalid request line: %v", line)
	}
	s2 += s1 + 1
	return line[:s1], line[s1+1 : s2], line[s2+1:], nil
}

func (sar *Request) String() string {
	command := sar.prepareFirstLine()

	buf := bytes.NewBuffer([]byte{})
	if _, err := io.WriteString(buf, command+"\r\n"); err != nil {
		panic(errors.Wrapf(err, "Error writting command"))
	}

	headers := http.Header(sar.Headers)

	if err := headers.Write(buf); err != nil {
		panic(errors.Wrapf(err, "Error writting headers"))
	}
	if _, err := io.WriteString(buf, "\r\n"); err != nil {
		panic(errors.Wrapf(err, "Error ending headers"))
	}

	return string(buf.Bytes())
}

// Write will write a SocketAce command and MIME headers to the stream.
func (sar *Request) Write(c io.Writer) error {
	if _, err := c.Write([]byte(sar.String())); err != nil {
		return errors.Wrapf(err, "Could not write to stream")
	}

	log.Debugf("Request written: %v", sar)
	return nil
}

// Read will write a SocketAce command and MIME headers from the stream.
func (sar *Request) Read(c *bufio.Reader) error {
	line, headers, err := readHeader(c)
	if err != nil {
		return errors.WithStack(err)
	}

	method, requestUri, proto, err := sar.parseRequestLine(line)
	if err != nil {
		return errors.WithStack(err)
	}

	sar.Method = method
	sar.URL = requestUri
	sar.Proto = proto
	sar.Headers = headers

	log.Debugf("Request read: %v", sar)
	return nil
}
