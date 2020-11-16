package dns

import (
	"bufio"
	"github.com/miekg/dns"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"io"
	"net"
	"os"
	"sync"
	"testing"
	"time"
)

const testDomain = "example.org"

type testCommunicator struct {
	closed  bool
	message OnMessage
}

func echoService(r io.ReadCloser, w io.WriteCloser) error {
	defer func() {
		log.Debugf("(echo)   Closing streams...")
		r.Close()
		w.Close()
	}()

	scanner := bufio.NewReader(r)

	var line []byte
	for true {
		l, prefix, err := scanner.ReadLine()
		if err == io.EOF {
			if len(line) == 0 {
				log.Debugf("(echo)   EOF")
				return nil
			}
		} else if err != nil {
			return err
		} else if prefix {
			line = append(line, l...)
			continue
		} else {
			line = append(line, l...)
		}

		log.Tracef("(echo)   Received: %v", string(line))
		response := append(line, '\r', '\n')
		if _, err := w.Write(response); err != nil {
			return err
		}
		log.Tracef("(echo)   Wrote:    %v", string(response[0:len(response)-2]))
		if string(line) == "QUIT" {
			break
		}

		line = make([]byte, 0)

		if err == io.EOF {
			return nil
		}

	}

	return nil
}

func (t *testCommunicator) Close() error {
	t.closed = true
	return nil
}

func (t *testCommunicator) Closed() bool {
	return t.closed
}

func (t *testCommunicator) SendAndReceive(m *dns.Msg, timeout *time.Duration) (r *dns.Msg, rtt time.Duration, err error) {
	if t.message != nil {
		msg, err := t.message(m, t.LocalAddr())
		return msg, time.Millisecond, err
	}
	return nil, time.Duration(0), errors.New("responding method not defined")
}

func (t *testCommunicator) LocalAddr() net.Addr {
	return &net.UDPAddr{IP: net.IPv4(127, 0, 0, 0), Port: 1234}
}

func (t *testCommunicator) RemoteAddr() net.Addr {
	return &net.UDPAddr{IP: net.IPv4(127, 0, 0, 0), Port: 53}
}

func (t *testCommunicator) SetDeadline(time time.Time) error {
	// ignore
	return nil
}

func (t *testCommunicator) SetReadDeadline(time time.Time) error {
	// ignore
	return nil
}

func (t *testCommunicator) SetWriteDeadline(time time.Time) error {
	// ignore
	return nil
}

func (t *testCommunicator) RegisterAccept(message OnMessage) {
	t.message = message
}

func TestMain(m *testing.M) {
	log.SetLevel(log.TraceLevel)

	code := m.Run()

	log.Infof("Tests complete")
	os.Exit(code)
}

func Test_Handshake(t *testing.T) {

	comm := &testCommunicator{}

	NewServerDnsListener(testDomain, comm)
	client, err := NewClientDnsConnection(testDomain, comm)
	require.NoError(t, err)

	err = client.AutoDetectQueryType()
	require.NoError(t, err)

	err = client.VersionHandshake()
	require.NoError(t, err)

	client.AutodetectEdns0Extension()
	client.AutodetectEncodingUpstream()

}

func Test_Connection(t *testing.T) {
	var server *ServerDnsListener
	var client *ClientDnsConnection
	var err error

	comm := &testCommunicator{}

	server = NewServerDnsListener(testDomain, comm)
	client, err = NewClientDnsConnection(testDomain, comm)
	require.NoError(t, err)

	err = client.Handshake()
	require.NoError(t, err)

	var echoServiceError error

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		conn, err := server.Accept()
		require.NoError(t, err)

		echoServiceError = echoService(conn, conn)
		if echoServiceError != nil {
			echoServiceError = errors.WithStack(err)
			log.WithError(err).Warnf("(echo) Error in echo service: %v", err)
		} else {
			log.Tracef("(echo) Echo service completed successfully.")
		}
		wg.Done()
	}()

	scanner := bufio.NewScanner(client)

	log.Debugf("(client) Sending HELLO...")
	_, err = client.Write([]byte("HELLO\r\n"))
	require.NoError(t, err)
	log.Debugf("(client) Waiting for HELO...")
	require.True(t, scanner.Scan(), "Could not get first line from echo service")
	require.Equal(t, "HELLO", scanner.Text())

	log.Debugf("(client) Sending QUIT...")
	_, err = client.Write([]byte("QUIT\r\n"))
	require.NoError(t, err)
	log.Debugf("(client) Waiting for QUIT...")
	require.True(t, scanner.Scan(), "Could not get the second line from the echo service")
	require.Equal(t, "QUIT", scanner.Text())

	wg.Wait()

	require.NoError(t, echoServiceError)

}
