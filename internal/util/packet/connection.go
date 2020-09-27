package packet

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"io"
	"net"
	"strings"
	"time"
)

type Connection struct {
	stream  io.ReadWriteCloser
	scanner *bufio.Scanner
}

func NewUpstreamConnection(stream io.ReadWriteCloser) *Connection {
	conn := &Connection{
		stream:  stream,
		scanner: bufio.NewScanner(stream),
	}
	conn.scanner.Split(packetSplit)

	return conn
}

// ReadFrom reads a packet from the connection,
// copying the payload into p. It returns the number of
// bytes copied into p and the return address that
// was on the packet.
// It returns the number of bytes read (0 <= n <= len(p))
// and any error encountered. Callers should always process
// the n > 0 bytes returned before considering the error err.
// ReadFrom can be made to time out and return
// an Error with Timeout() == true after a fixed time limit;
// see SetDeadline and SetReadDeadline.
func (c *Connection) ReadFrom(p []byte) (n int, addr net.Addr, err error) {
	if c.scanner.Scan() {
		data := c.scanner.Bytes()
		buf := bytes.NewBuffer(data)

		var addressLen uint8
		mustRead(buf, &addressLen)
		addressBytes := make([]byte, addressLen)
		mustRead(buf, &addressBytes)
		addressString := strings.SplitN(string(addressBytes), ":", 2)

		network := addressString[0]
		switch network {
		case "udp", "udp4", "udp6":
			addr, err = net.ResolveUDPAddr(network, addressString[1])
		case "tcp", "tcp4", "tcp6":
			addr, err = net.ResolveTCPAddr(network, addressString[1])
		case "unix", "unixgram", "unixpacket":
			addr, err = net.ResolveUnixAddr(network, addressString[1])
		default:
			err = errors.Errorf("Unknown network name: %v", network)
		}

		if err != nil {
			return
		}

		data = data[1 + addressLen:]

		n = len(data)
		if n > len(p) {
			return 0, nil, errors.Errorf("Data array to short. Expected at least %v bytes, but got %v", n, len(p))
		}

		copy(p, data)

		return

	}

	return 0, nil, c.scanner.Err()
}

// WriteTo writes a packet with payload p to addr.
// WriteTo can be made to time out and return
// an Error with Timeout() == true after a fixed time limit;
// see SetDeadline and SetWriteDeadline.
// On packet-oriented connections, write timeouts are rare.
func (c *Connection) WriteTo(p []byte, addr net.Addr) (n int, err error) {
	buffer := &bytes.Buffer{}
	addrString := addr.Network() + ":" + addr.String()

	addressLen := len(addrString)
	dataLen := len(p)

	mustWrite(buffer, Magic)
	mustWrite(buffer, uint32(dataLen+addressLen))
	mustWrite(buffer, uint8(addressLen))
	mustWrite(buffer, []byte(addrString))
	mustWrite(buffer, p)

	return c.stream.Write(buffer.Bytes())
}

// packetSplit is a split function used to tokenize the
// input. The arguments are an initial substring of the remaining unprocessed
// data and a flag, atEOF, that reports whether the Reader has no more data
// to give. The return values are:
// - the number of bytes to advance the input
// - the next token to return to the user, if any, plus
// - an error, if any.
//
// Scanning stops if the function returns an error, in which case some of
// the input may be discarded.
//
// Otherwise, the Scanner advances the input. If the token is not nil,
// the Scanner returns it to the user. If the token is nil, the
// Scanner reads more data and continues scanning; if there is no more
// data--if atEOF was true--the Scanner returns. If the data does not
// yet hold a complete token, for instance if it has no newline while
// scanning lines, a SplitFunc can return (0, nil, nil) to signal the
// Scanner to read more data into the slice and try again with a
// longer slice starting at the same point in the input.
//
// The function is never called with an empty data slice unless atEOF
// is true. If atEOF is true, however, data may be non-empty and,
// as always, holds unprocessed text.
func packetSplit(data []byte, atEOF bool) (advance int, token []byte, err error) {
	dataLen := len(data)
	if atEOF && dataLen == 0 {
		return 0, nil, nil
	}
	if i := bytes.Index(data, Magic); i == 0 {
		// Needs at least header + len of data (uint32) + len of address (uint8)
		if dataLen < MagicLength+4+1 {
			return 0, nil, nil
		}
		buf := bytes.NewBuffer(data[MagicLength:])

		var packetLen uint32
		mustRead(buf, &packetLen)

		if dataLen < MagicLength+4+1+int(packetLen) {
			return 0, nil, nil
		}

		data = data[MagicLength+4:]

		return MagicLength + 4 + 1 + int(packetLen), data[:packetLen], nil

	} else if i > 0 {
		log.Warnf("Garbled data, ignoring intermediate packet.")
		return i, nil, nil
	}

	// If we're at EOF, we have a final, non-terminated line. Return it.
	if atEOF {
		return dataLen, data, nil
	}
	// Request more data.
	return 0, nil, nil

}

func mustWrite(w io.Writer, data interface{}) {
	if err := binary.Write(w, binary.LittleEndian, data); err != nil {
		panic(errors.Wrapf(err, "Something really wrong with memory handling: %v", err))
	}
}

func mustRead(r io.Reader, data interface{}) {
	if err := binary.Read(r, binary.LittleEndian, data); err != nil {
		panic(errors.Wrapf(err, "Something really wrong with memory handling: %v", err))
	}
}

// Close closes the connection.
// Any blocked ReadFrom or WriteTo operations will be unblocked and return errors.
func (c *Connection) Close() error {
	if c.stream != nil {
		return c.stream.Close()
	} else {
		return nil
	}
}

// LocalAddr returns the local network address.
func (c *Connection) LocalAddr() net.Addr {
	return nil
}

// SetDeadline sets the read and write deadlines associated
// with the connection. It is equivalent to calling both
// SetReadDeadline and SetWriteDeadline.
//
// A deadline is an absolute time after which I/O operations
// fail with a timeout (see type Error) instead of
// blocking. The deadline applies to all future and pending
// I/O, not just the immediately following call to ReadFrom or
// WriteTo. After a deadline has been exceeded, the connection
// can be refreshed by setting a deadline in the future.
//
// An idle timeout can be implemented by repeatedly extending
// the deadline after successful ReadFrom or WriteTo calls.
//
// A zero value for t means I/O operations will not time out.
func (c *Connection) SetDeadline(t time.Time) error {
	return nil
}

// SetReadDeadline sets the deadline for future ReadFrom calls
// and any currently-blocked ReadFrom call.
// A zero value for t means ReadFrom will not time out.
func (c *Connection) SetReadDeadline(t time.Time) error {
	return nil
}

// SetWriteDeadline sets the deadline for future WriteTo calls
// and any currently-blocked WriteTo call.
// Even if write times out, it may return n > 0, indicating that
// some of the data was successfully written.
// A zero value for t means WriteTo will not time out.
func (c *Connection) SetWriteDeadline(t time.Time) error {
	return nil
}
