package dns

import (
	"bufio"
	"crypto/rand"
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

		// Simulate sending over the wire
		var source dns.Msg
		if data, err := m.Pack(); err != nil {
			return nil, 0, errors.WithStack(err)
		} else if err := source.Unpack(data); err != nil {
			return nil, 0, errors.WithStack(err)
		}

		msg, err := t.message(&source, t.LocalAddr())
		return msg, time.Millisecond, errors.WithStack(err)
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

func Test_ConnectionViaNetwork(t *testing.T) {
	var server *ServerDnsListener
	var client *ClientDnsConnection
	var err error

	serverComm, err := NewNetConnectionServerCommunicator(&dns.Server{
		Addr: "127.0.0.1:42000",
		Net:  "udp",
	})
	require.NoError(t, err)

	clientComm, err := NewNetConnectionClientCommunicator(&dns.ClientConfig{
		Servers:  []string{"127.0.0.1"},
		Search:   []string{},
		Port:     "42000",
		Ndots:    0,
		Timeout:  1000,
		Attempts: 2,
	})
	require.NoError(t, err)

	server = NewServerDnsListener(testDomain, serverComm)
	client, err = NewClientDnsConnection(testDomain, clientComm)
	require.NoError(t, err)

	defer client.Close()
	defer server.Close()

	echoTest(t, err, client, server)
}

func echoTest(t *testing.T, err error, client *ClientDnsConnection, server *ServerDnsListener) {
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

	log.Debugf("(client) Sending:  HELLO")
	_, err = client.Write([]byte("HELLO\r\n"))
	require.NoError(t, err)
	log.Debugf("(client) Waiting:  HELLO")
	require.True(t, scanner.Scan(), "Could not get first line from echo service")
	require.Equal(t, "HELLO", scanner.Text())

	log.Debugf("(client) Sending:  QUIT")
	_, err = client.Write([]byte("QUIT\r\n"))
	require.NoError(t, err)
	log.Debugf("(client) Waiting:  QUIT")
	require.True(t, scanner.Scan(), "Could not get the second line from the echo service")
	require.Equal(t, "QUIT", scanner.Text())

	wg.Wait()

	require.NoError(t, echoServiceError)
}

func Test_LargeUpload(t *testing.T) {
	var server *ServerDnsListener
	var client *ClientDnsConnection
	var err error

	serverComm, err := NewNetConnectionServerCommunicator(&dns.Server{
		Addr: "127.0.0.1:42001",
		Net:  "udp",
	})
	require.NoError(t, err)

	clientComm, err := NewNetConnectionClientCommunicator(&dns.ClientConfig{
		Servers:  []string{"127.0.0.1"},
		Search:   []string{},
		Port:     "42001",
		Ndots:    0,
		Timeout:  1000,
		Attempts: 2,
	})
	require.NoError(t, err)

	server = NewServerDnsListener(testDomain, serverComm)
	client, err = NewClientDnsConnection(testDomain, clientComm)
	require.NoError(t, err)
	defer client.Close()
	defer server.Close()

	source := make([]byte, 1024*512) // 512 kb
	dest := make([]byte, 0)
	n, err := rand.Read(source)
	require.NoError(t, err)
	l := len(source)
	require.Equal(t, l, n)

	var serverErr = make(chan error, 1)
	go func() {
		conn, err := server.Accept()
		if err != nil {
			serverErr <- err
			return
		}
		buf := make([]byte, 65536)
		for true {
			n, err := conn.Read(buf)
			if err == io.EOF || err == os.ErrClosed {
				dest = append(dest, buf[0:n]...)
				break
			} else if err != nil {
				serverErr <- err
				return
			}
			dest = append(dest, buf[0:n]...)
			if len(dest) == len(source) {
				break
			}
		}

		serverErr <- nil

	}()

	err = client.Handshake()
	require.NoError(t, err)

	size := 32768
	pos := 0
	i := 0
	for pos < l {
		i++
		log.Debugf("Sending %d/%d, %d%%", i, l/size, int(float64(pos)/float64(l)*100.0))
		if pos+size > l {
			_, err = client.Write(source[pos:l])
		} else {
			_, err = client.Write(source[pos : pos+size])
		}
		require.NoError(t, err)
		pos += size
	}
	require.NoError(t, client.Close())

	select {
	case err = <-serverErr:
		require.NoError(t, err)
	}

	require.Equal(t, source, dest)
}

func Test_LargeDownload(t *testing.T) {
	var server *ServerDnsListener
	var client *ClientDnsConnection
	var err error

	serverComm, err := NewNetConnectionServerCommunicator(&dns.Server{
		Addr: "127.0.0.1:42002",
		Net:  "udp",
	})
	require.NoError(t, err)

	clientComm, err := NewNetConnectionClientCommunicator(&dns.ClientConfig{
		Servers:  []string{"127.0.0.1"},
		Search:   []string{},
		Port:     "42002",
		Ndots:    0,
		Timeout:  1000,
		Attempts: 2,
	})
	require.NoError(t, err)

	server = NewServerDnsListener(testDomain, serverComm)
	client, err = NewClientDnsConnection(testDomain, clientComm)
	require.NoError(t, err)
	defer client.Close()
	defer server.Close()

	source := make([]byte, 1024*512) // 512 kb
	dest := make([]byte, 0)
	n, err := rand.Read(source)
	require.NoError(t, err)
	l := len(source)
	require.Equal(t, l, n)

	var serverErr = make(chan error, 1)
	go func() {
		conn, err := server.Accept()
		if err != nil {
			serverErr <- err
			return
		}

		size := 32768
		pos := 0
		i := 0
		for pos < l {
			i++
			log.Debugf("Sending %d/%d, %d%%", i, l/size, int(float64(pos)/float64(l)*100.0))
			if pos+size > l {
				_, err = conn.Write(source[pos:l])
			} else {
				_, err = conn.Write(source[pos : pos+size])
			}
			if err != nil {
				serverErr <- err
				return
			}
			pos += size
		}

		_ = conn.Close()

		serverErr <- nil

	}()

	err = client.Handshake()
	require.NoError(t, err)

	buf := make([]byte, 65536)
	for true {
		n, err := client.Read(buf)
		if err == io.EOF || err == os.ErrClosed {
			dest = append(dest, buf[0:n]...)
			break
		} else if err != nil {
			require.NoError(t, err)
			return
		}
		dest = append(dest, buf[0:n]...)
		//log.Debugf("%d of %d: %d", len(dest), len(source), int(float64(len(dest))/float64(len(source))*100))
		if len(dest) == len(source) {
			break
		}
	}

	require.NoError(t, client.Close())

	select {
	case err = <-serverErr:
		require.NoError(t, err)
	}

	require.Equal(t, source, dest)
}
