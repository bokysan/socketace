package it

import (
	"bufio"
	"github.com/bokysan/socketace/v2/internal/client/listener"
	"github.com/bokysan/socketace/v2/internal/client/upstream"
	clientCmd "github.com/bokysan/socketace/v2/internal/commands/client"
	serverCmd "github.com/bokysan/socketace/v2/internal/commands/server"
	"github.com/bokysan/socketace/v2/internal/server"
	"github.com/bokysan/socketace/v2/internal/streams"
	"github.com/bokysan/socketace/v2/internal/util/addr"
	"github.com/bokysan/socketace/v2/internal/util/cert"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"io"
	"net"
	"net/url"
	"os"
	"strconv"
	"testing"
)

const echoServicePort int = 41000

var echoServiceAddress = "127.0.0.1:" + strconv.Itoa(echoServicePort)

type closer func()

func mustParseUrl(endpoint string) *url.URL {
	u, err := url.Parse(endpoint)
	if err != nil {
		panic(err)
	}
	return u
}

func echoService(r io.ReadCloser, w io.WriteCloser) error {
	defer streams.TryClose(r)
	defer streams.TryClose(w)

	scanner := bufio.NewReader(r)

	var line []byte
	for true {
		l, prefix, err := scanner.ReadLine()
		if err == io.EOF {
			return nil
		} else if err != nil {
			return err
		} else if prefix {
			line = append(line, l...)
			continue
		} else {
			line = append(line, l...)
		}

		log.Tracef("Got: %v", string(line))
		response := append(line, '\r', '\n')
		if _, err := w.Write(response); err != nil {
			return err
		}
		log.Tracef("Wrote: %v", string(response))

		if string(line) == "QUIT" {
			break
		}

		line = make([]byte, 0)
	}

	return nil
}

func TestMain(m *testing.M) {
	log.SetLevel(log.TraceLevel)

	closer, err := setup()
	if err != nil {
		panic(err)
	}

	code := m.Run()

	log.Infof("Tests complete")

	closer()

	log.Debugf("Existing tests...")

	os.Exit(code)
}

func setup() (closer, error) {

	shutdown := make(chan bool, 1)
	l, err := net.Listen("tcp", echoServiceAddress)
	if err != nil {
		return nil, err
	}
	log.Infof("Echo service %v started at %v", l, echoServiceAddress)

	go func() {
		for {
			var conn net.Conn

			select {
			case <-shutdown:
				return
			default:
				conn, err = l.Accept()
			}

			select {
			case <-shutdown:
				return
			default:
				if err != nil {
					panic(err)
				}
			}

			conn = streams.NewNamedConnection(conn, conn.RemoteAddr().String())
			log.Infof("New connection to echo service detected from %v", conn)

			go func(c net.Conn) {
				err := echoService(c, c)
				if err != nil {
					panic(err)
				}
			}(conn)
		}
	}()

	return func() {
		shutdown <- true
		streams.TryClose(l)
	}, nil
}

func helloEchoTest(t *testing.T, conn net.Conn) {
	var err error

	scanner := bufio.NewScanner(conn)

	log.Debugf("Sending HELO...")
	_, err = conn.Write([]byte("HELLO\r\n"))
	require.NoError(t, err)
	log.Debugf("Reading HELO...")
	require.True(t, scanner.Scan(), "Could not get first line from echo service")
	require.Equal(t, "HELLO", scanner.Text())

	log.Debugf("Sending QUIT...")
	_, err = conn.Write([]byte("QUIT\r\n"))
	require.NoError(t, err)
	log.Debugf("Reading QUIT...")
	require.True(t, scanner.Scan(), "Could not gt the second line from the echo service")
	require.Equal(t, "QUIT", scanner.Text())

	log.Debugf("Making sure the stream is finished...")
	require.False(t, scanner.Scan())

}

func Test_SimpleConnection(t *testing.T) {

	p1Reader, p1Writer := io.Pipe()
	p2Reader, p2Writer := io.Pipe()

	localServiceAddress := "localhost:" + strconv.Itoa(echoServicePort+1)

	s := serverCmd.Command{
		Channels: server.Channels{
			&server.NetworkChannel{
				AbstractChannel: server.AbstractChannel{
					Kind: "network",
					ProtoName: addr.ProtoName{
						Name: "echo",
					},
				},
				ProtoAddress: addr.MustParseAddress("tcp://" + echoServiceAddress),
			},
		},
		Servers: server.Servers{
			&server.IoServer{
				Address: addr.MustParseAddress("stdio://"),
				Input:   p1Reader,
				Output:  p2Writer,
			},
		},
	}

	c := clientCmd.Command{
		Upstream: upstream.Upstreams{
			Data: []upstream.Upstream{
				&upstream.InputOutput{
					Address: addr.MustParseAddress("stdio://"),
					Input:   p2Reader,
					Output:  p1Writer,
				},
			},
		},
		ListenList: listener.ListenList{
			&listener.Listener{
				ProtoName: addr.ProtoName{
					Name: "echo",
				},
				Address: addr.MustParseAddress("tcp://" + localServiceAddress),
			},
		},
	}

	interrupted := make(chan os.Signal, 1)
	require.NoError(t, s.Startup(interrupted))
	require.NoError(t, c.Startup(interrupted))

	defer func() {
		interrupted <- os.Interrupt
		require.NoError(t, c.Shutdown())
		require.NoError(t, s.Shutdown())
	}()

	conn, err := net.Dial("tcp", localServiceAddress)
	require.NoError(t, err)

	conn = streams.NewSafeConnection(conn)
	defer streams.TryClose(conn)

	helloEchoTest(t, conn)

	log.Infof("Test completed.")

}

func Test_SimpleSslConnection(t *testing.T) {

	p1Reader, p1Writer := io.Pipe()
	p2Reader, p2Writer := io.Pipe()

	localServiceAddress := "localhost:" + strconv.Itoa(echoServicePort+1)

	s := serverCmd.Command{
		Channels: server.Channels{
			&server.NetworkChannel{
				AbstractChannel: server.AbstractChannel{
					Kind: "network",
					ProtoName: addr.ProtoName{
						Name: "echo",
					},
				},
				ProtoAddress: addr.MustParseAddress("tcp://" + echoServiceAddress),
			},
		},
		Servers: server.Servers{
			&server.IoServer{
				ServerConfig: cert.ServerConfig{
					Config: cert.Config{
						Certificate:        testCertificate,
						PrivateKey:         testPrivatekey,
						PrivateKeyPassword: &testPassword,
					},
					RequireClientCert: false,
				},
				Address: addr.MustParseAddress("stdio+tls://"),
				Input:   p1Reader,
				Output:  p2Writer,
			},
		},
	}

	c := clientCmd.Command{
		ClientConfig: cert.ClientConfig{
			InsecureSkipVerify: true,
		},
		Upstream: upstream.Upstreams{
			Data: []upstream.Upstream{
				&upstream.InputOutput{
					Address: addr.MustParseAddress("stdio+tls://"),
					Input:   p2Reader,
					Output:  p1Writer,
				},
			},
		},
		ListenList: listener.ListenList{
			&listener.Listener{
				ProtoName: addr.ProtoName{
					Name: "echo",
				},
				Address: addr.MustParseAddress("tcp://" + localServiceAddress),
			},
		},
	}

	interrupted := make(chan os.Signal, 1)
	require.NoError(t, s.Startup(interrupted))
	require.NoError(t, c.Startup(interrupted))

	defer func() {
		interrupted <- os.Interrupt
		require.NoError(t, c.Shutdown())
		require.NoError(t, s.Shutdown())
	}()

	conn, err := net.Dial("tcp", localServiceAddress)
	require.NoError(t, err)

	conn = streams.NewSafeConnection(conn)

	defer streams.TryClose(conn)

	helloEchoTest(t, conn)

	log.Infof("Test completed.")

}

func Test_SimpleStartTlsConnection(t *testing.T) {

	p1Reader, p1Writer := io.Pipe()
	p2Reader, p2Writer := io.Pipe()

	localServiceAddress := "localhost:" + strconv.Itoa(echoServicePort+1)

	s := serverCmd.Command{
		Channels: server.Channels{
			&server.NetworkChannel{
				AbstractChannel: server.AbstractChannel{
					Kind: "network",
					ProtoName: addr.ProtoName{
						Name: "echo",
					},
				},
				ProtoAddress: addr.MustParseAddress("tcp://" + echoServiceAddress),
			},
		},
		Servers: server.Servers{
			&server.IoServer{
				ServerConfig: cert.ServerConfig{
					Config: cert.Config{
						Certificate:        testCertificate,
						PrivateKey:         testPrivatekey,
						PrivateKeyPassword: &testPassword,
					},
					RequireClientCert: false,
				},
				Address: addr.MustParseAddress("stdio://"),
				Input:   p1Reader,
				Output:  p2Writer,
			},
		},
	}

	c := clientCmd.Command{
		ClientConfig: cert.ClientConfig{
			InsecureSkipVerify: true,
		},
		Upstream: upstream.Upstreams{
			Data: []upstream.Upstream{
				&upstream.InputOutput{
					Address: addr.MustParseAddress("stdio://"),
					Input:   p2Reader,
					Output:  p1Writer,
				},
			},
		},
		ListenList: listener.ListenList{
			&listener.Listener{
				ProtoName: addr.ProtoName{
					Name: "echo",
				},
				Address: addr.MustParseAddress("tcp://" + localServiceAddress),
			},
		},
	}

	interrupted := make(chan os.Signal, 1)
	require.NoError(t, s.Startup(interrupted))
	require.NoError(t, c.Startup(interrupted))

	defer func() {
		interrupted <- os.Interrupt
		require.NoError(t, c.Shutdown())
		require.NoError(t, s.Shutdown())
	}()

	conn, err := net.Dial("tcp", localServiceAddress)
	require.NoError(t, err)

	conn = streams.NewSafeConnection(conn)

	defer streams.TryClose(conn)

	helloEchoTest(t, conn)

	log.Infof("Test completed.")

}