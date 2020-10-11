package socketace

import (
	"bufio"
	"github.com/bokysan/socketace/v2/internal/version"
	"github.com/pkg/errors"
	"net/textproto"
)

const (
	RequestMethod          = "X-SOCKETACE"
	AcceptsProtocolVersion = "Accepts-Protocol-Version"
	UserAgent              = "User-Agent"
	Status                 = "Status"
	Capabilities           = "Capabilities"
	CapabilityStartTls     = "StartTLS"
	SecurityUnderlying     = "underlying"
	SecurityNone           = "none"
	SecurityTls            = "tls"
)

var SupportedProtocolVersions = []string{
	// Make sure this list is in descending order
	version.ProtocolVersion,
}

// readHeader will read the SocketAce header from the stream along with all mime headers and stop reading
// the stram at that point.
func readHeader(conn *bufio.Reader) (string, textproto.MIMEHeader, error) {
	headerReader := textproto.NewReader(conn)
	firstLine, err := headerReader.ReadLine()
	if err != nil {
		return "", nil, errors.WithStack(err)
	}

	header, err := headerReader.ReadMIMEHeader()
	if err != nil {
		return "", nil, errors.Wrapf(err, "Could not read MIME header!")
	}
	return firstLine, header, nil
}
