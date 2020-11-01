package socketace

import (
	"bufio"
	"github.com/bokysan/socketace/v2/internal/streams"
	"github.com/bokysan/socketace/v2/internal/util/cert"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"io"
	"sync"
	"testing"
)

var testPassword = "test1234"

const testCertificate = `
-----BEGIN CERTIFICATE-----
MIIFkTCCA3mgAwIBAgIUJQGiIyrFI52mPHueeMHgzB6yFXEwDQYJKoZIhvcNAQEL
BQAwWDELMAkGA1UEBhMCRVUxDTALBgNVBAoMBFRlc3QxGTAXBgNVBAMMEHRlc3Qu
ZXhhbXBsZS5jb20xHzAdBgkqhkiG9w0BCQEWEHRlc3RAZXhhbXBsZS5jb20wHhcN
MjAwOTAxMTcwNjM1WhcNMzAwODMwMTcwNjM1WjBYMQswCQYDVQQGEwJFVTENMAsG
A1UECgwEVGVzdDEZMBcGA1UEAwwQdGVzdC5leGFtcGxlLmNvbTEfMB0GCSqGSIb3
DQEJARYQdGVzdEBleGFtcGxlLmNvbTCCAiIwDQYJKoZIhvcNAQEBBQADggIPADCC
AgoCggIBANfRK4TEZXTBKu33S9ARkSpd38EWc9wMDYsKNBDoIvZMjPAh96gEQc6B
ol5v5ykouiApmeq8ytzw/wu8YRqwl+Yut+fsxBoVhVuwUSeTMfRl5Rnv+w0FCGFP
sXhDhvZu+mWle4OXyvUHl3HxLmIwfi4vZfiqnka3bhfzyRZzoHlHpk+aRmE3Smpv
lynJILAaF33Yqh22CoWtAhd43SgN/Y2Ri22b1d2pJneLriVjxi63A+NSc1HGAvQE
rvDrqfUfVMdEUVe254+f6z5vf3iEhozpY8VFeaSP9Nex5b62LDWba6xvr1ZwkI8U
Pv/4+1PUs2LdMB8CVV8mN+G11Ba/7KTawIvFz+VYQ5iEHBUm8fqEDOfJOu2tX3ZO
0tZQzLkMb+oY16FHxdD8lwKUcqOJ5cMoi965g2/sN/iBbb/+cxj93FTVfiM5Wafy
7nLMuEcYN3hf3YpXq3FoiSaQ114wW7O8MMuVx/50/2zxIa3tkBZljqj2BkhtQNkO
pluS72b5u0qubIWC3dJv+nQxX2uD5/3BnB7CjH62B+t+rw9Hn9TbZxLh159xvtfZ
J5wk0s7jwaL0JANlUCEHOeyazPb5uxkvEW2nZTMUyzda1rJhjZD9FUWLd9v5ms0L
+nVzTec8Lb04MR6ZndpS0D7xaoMJSPCDEOMacHFbUGVDAsgKjQVbAgMBAAGjUzBR
MB0GA1UdDgQWBBQChaAFLfQWUdMrAFndPjM5QQYZwTAfBgNVHSMEGDAWgBQChaAF
LfQWUdMrAFndPjM5QQYZwTAPBgNVHRMBAf8EBTADAQH/MA0GCSqGSIb3DQEBCwUA
A4ICAQARd+l7nU5oYvlbdgaI4ogH+DV4mPsfxlyZIEh6yrwymgSPHnA703v4lR2I
RoLm+b2wuY5fvlKgMXaBOFJ0PGVcUT9F2WyoqdisNaue3BBOyZLkJVEOW6VO4PFv
7kjDps8HDOqNulsnTbC5oU/zafa1Q+njmhlacujm6fEbW7X/xoxAkR7GiKZwlNeI
NGsFDgAKZ70mlFXxIUiQUw7nCrEhvCOObD6jV3eI56FegInqtQ5BbfSifrDHIb0+
sXpBjsuQXwU0AyjyzWvSxgJ06nWIyfd3XsJG+Awqf/OPrL504GjLKQK/LJjYim9/
KT50QwCp6viQvr3dP/b4zUJNf+lo11+VlImBsjNHaXJX2eWZslnjR/JDuIqeV1Sd
IJJQDLgulAQUk0GL3HL3gR6JxEF/a5JEVewWRmxPz2/8a9kSHD7tegdZJLrLy8Ax
sugzRZdajj8JhTvPyYJerOKVaJEfvjgKkT5Rak9L6nfOGX4NsyW4RzNVs7PG7F8O
tDgQkJcged8P/lIOh6+RBySLzNGxVyGarrd2dVDrWCVIn5jtx/pQCDW34USLG4XC
/0KnrkpxMEPCV4ZmX8gCgY9w6K2FZ4AMj+zp2kBfyfAANn7oOQpqKCnECHeDDb9m
QPhnv53R3IXyQpfpl/zieIqJw5CqAesWtelJhBMc+/aeYM2Zww==
-----END CERTIFICATE-----
`
const testPrivatekey = `
-----BEGIN ENCRYPTED PRIVATE KEY-----
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
eIxrQ0ucDlrE1bXXhYUK+Q==
-----END ENCRYPTED PRIVATE KEY-----
`

// bufferReadCloser is an implementation of the bufio.Reader which also implements the Closer interface
type bufferReadCloser struct {
	*bufio.Reader
	closer io.ReadCloser
}

func (b *bufferReadCloser) Close() error {
	return b.closer.Close()
}

type socketaceTester struct {
	clientPipe *streams.SimulatedConnection
	client     *ClientConnection
	clientErr  error

	serverPipe *streams.SimulatedConnection
	server     *ServerConnection
	serverErr  error

	wg sync.WaitGroup
}

func newSocketaceTester() *socketaceTester {
	p1Reader, p1Writer := io.Pipe()
	p2Reader, p2Writer := io.Pipe()
	p1 := streams.NewReadWriteCloser(&bufferReadCloser{
		Reader: bufio.NewReader(p1Reader),
		closer: p1Reader,
	}, p2Writer)
	p2 := streams.NewReadWriteCloser(&bufferReadCloser{
		Reader: bufio.NewReader(p2Reader),
		closer: p1Reader,
	}, p1Writer)

	res := &socketaceTester{
		clientPipe: streams.NewSimulatedConnection(p1, streams.Localhost, streams.Localhost),
		serverPipe: streams.NewSimulatedConnection(p2, streams.Localhost, streams.Localhost),
	}
	res.wg.Add(2)

	return res
}

func (st *socketaceTester) testRunServer(manager cert.TlsConfig, secure bool) {
	log.Info("Creating new server connection...")
	st.server, st.serverErr = NewServerConnection(st.serverPipe, manager, secure)
	if st.serverErr != nil {
		log.Infof("Server connection error: %v", st.serverErr)
	} else {
		log.Infof("Server connection established: %v", st.server)
	}
	st.wg.Done()

	return
}

func (st *socketaceTester) testRunClient(manager cert.TlsConfig, secure bool, host string) {
	log.Info("Creating new client connection...")
	st.client, st.clientErr = NewClientConnection(st.clientPipe, manager, secure, host)
	if st.clientErr != nil {
		log.Infof("Client connection error: %v", st.clientErr)
	} else {
		log.Infof("Client connection established: %v", st.client)
	}
	st.wg.Done()

	return
}

func Test_SimpleConnection(t *testing.T) {
	log.SetLevel(log.TraceLevel)

	tester := newSocketaceTester()
	go tester.testRunServer(nil, false)
	go tester.testRunClient(nil, false, "")

	tester.wg.Wait()

	require.NoError(t, tester.serverErr, "Could not setup a server connection!")
	require.NoError(t, tester.clientErr, "Could not setup a client connection!")
	require.False(t, tester.server.Secure())
	require.False(t, tester.client.Secure())
	require.Equal(t, "none", tester.server.SecurityTech())
	require.Equal(t, "none", tester.client.SecurityTech())
}

func Test_SecureServer1(t *testing.T) {
	log.SetLevel(log.TraceLevel)

	tester := newSocketaceTester()

	go tester.testRunServer(nil, true)
	go tester.testRunClient(nil, true, "")

	tester.wg.Wait()

	require.NoError(t, tester.serverErr, "Could not setup a server connection!")
	require.NoError(t, tester.clientErr, "Could not setup a client connection!")
	require.True(t, tester.server.Secure())
	require.True(t, tester.client.Secure())
	require.Equal(t, SecurityUnderlying, tester.server.SecurityTech())
	require.Equal(t, SecurityUnderlying, tester.client.SecurityTech())
}

func Test_SecureServer2(t *testing.T) {
	log.SetLevel(log.TraceLevel)

	tester := newSocketaceTester()

	serverManager := &cert.Config{
		CaCertificate:             "",
		CaCertificateFile:         "",
		Certificate:               testCertificate,
		CertificateFile:           "",
		PrivateKey:                testPrivatekey,
		PrivateKeyFile:            "",
		PrivateKeyPassword:        &testPassword,
		PrivateKeyPasswordProgram: "",
	}

	go tester.testRunServer(serverManager, true)
	go tester.testRunClient(nil, true, "")

	tester.wg.Wait()

	require.NoError(t, tester.serverErr, "Could not setup a server connection!")
	require.NoError(t, tester.clientErr, "Could not setup a client connection!")
	require.True(t, tester.server.Secure())
	require.True(t, tester.client.Secure())
	require.Equal(t, SecurityUnderlying, tester.server.SecurityTech())
	require.Equal(t, SecurityUnderlying, tester.client.SecurityTech())
}

func Test_SecureServerTls(t *testing.T) {
	log.SetLevel(log.TraceLevel)

	tester := newSocketaceTester()

	serverManager := &cert.Config{
		CaCertificate:             "",
		CaCertificateFile:         "",
		Certificate:               testCertificate,
		CertificateFile:           "",
		PrivateKey:                testPrivatekey,
		PrivateKeyFile:            "",
		PrivateKeyPassword:        &testPassword,
		PrivateKeyPasswordProgram: "",
	}

	clientManager := &cert.ClientConfig{
		InsecureSkipVerify: true,
	}

	go tester.testRunServer(serverManager, false)
	go tester.testRunClient(clientManager, false, "")

	tester.wg.Wait()

	require.NoError(t, tester.serverErr, "Could not setup a server connection!")
	require.NoError(t, tester.clientErr, "Could not setup a client connection!")
	require.True(t, tester.server.Secure())
	require.True(t, tester.client.Secure())
	require.Equal(t, SecurityTls, tester.server.SecurityTech())
	require.Equal(t, SecurityTls, tester.client.SecurityTech())
}
