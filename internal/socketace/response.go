package socketace

import (
	"bufio"
	"bytes"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"io"
	"net/http"
	"net/textproto"
	"strconv"
	"strings"
)

type Response struct {
	Status     string // e.g. "200 OK"
	StatusCode int    // e.g. 200
	Proto      string // e.g. "HTTP/1.1"
	Headers    textproto.MIMEHeader
}

// prepareFirstLine will prepare the request line
func (sar *Response) prepareFirstLine() string {
	command := ""
	if sar.Proto != "" {
		command += sar.Proto
	} else {
		command += "HTTP/1.1"
	}

	command += " " + sar.Status

	return command
}

// parseResponse line parses "HTTP/1.1 101 Switching Protocols" into its three parts
func (sar *Response) parseResponseLine(line string) (proto string, statusCode int, message string, err error) {
	s1 := strings.Index(line, " ")
	s2 := strings.Index(line[s1+1:], " ")
	if s1 < 0 || s2 < 0 {
		return "", 0, "", errors.Errorf("Invalid response line: %v", line)
	}
	s2 += s1 + 1
	status := line[s1+1 : s2]
	if sc, err := strconv.ParseInt(status, 10, 32); err != nil {
		return "", 0, "", err
	} else {
		return line[:s1], int(sc), line[s2+1:], nil
	}
}

func (sar *Response) String() string {
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
func (sar *Response) Write(c io.Writer) error {
	if _, err := c.Write([]byte(sar.String())); err != nil {
		return errors.Wrapf(err, "Could not write to stream")
	}
	log.Debugf("Response writen:\n%v", sar)
	return nil
}

// Read will write a SocketAce command and MIME headers from the stream.
func (sar *Response) Read(c *bufio.Reader) error {
	line, headers, err := readHeader(c)
	if err != nil {
		return errors.WithStack(err)
	}

	proto, statusCode, message, err := sar.parseResponseLine(line)
	if err != nil {
		return errors.WithStack(err)
	}

	sar.Proto = proto
	sar.StatusCode = statusCode
	sar.Status = strconv.Itoa(statusCode) + " " + message
	sar.Headers = headers

	log.Debugf("Response read:\n%v", sar)
	return nil
}
