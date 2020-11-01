package dns

import (
	"errors"
	"github.com/bokysan/socketace/v2/internal/streams/dns/util"
	"github.com/miekg/dns"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/dns/dnsmessage"
	"net"
	"os"
	"testing"
	"time"
)

const testDomain = "example.org"

type testCommunicator struct {
	closed  bool
	message OnMessage
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

func Test_DifferentQueryTypes(t *testing.T) {
	var srv *ServerDnsListener
	var client *ClientDnsConnection
	var err error
	var comm *testCommunicator

	comm = &testCommunicator{}
	srv = NewServerDnsListener(testDomain, comm)
	srv.allowedQueryTypes = []dnsmessage.Type{util.QueryTypeNull, util.QueryTypePrivate}
	client, err = NewClientDnsConnection(testDomain, comm)
	require.NoError(t, err)
	err = client.AutoDetectQueryType()
	require.NoError(t, err)

	comm = &testCommunicator{}
	srv = NewServerDnsListener(testDomain, comm)
	srv.allowedQueryTypes = []dnsmessage.Type{util.QueryTypePrivate, util.QueryTypeTxt, util.QueryTypeCname, util.QueryTypeA}
	client, err = NewClientDnsConnection(testDomain, comm)
	require.NoError(t, err)
	err = client.AutoDetectQueryType()
	require.NoError(t, err)

	comm = &testCommunicator{}
	srv = NewServerDnsListener(testDomain, comm)
	srv.allowedQueryTypes = []dnsmessage.Type{util.QueryTypeSrv, util.QueryTypeCname, util.QueryTypeA}
	client, err = NewClientDnsConnection(testDomain, comm)
	require.NoError(t, err)
	err = client.AutoDetectQueryType()
	require.NoError(t, err)

	comm = &testCommunicator{}
	srv = NewServerDnsListener(testDomain, comm)
	srv.allowedQueryTypes = []dnsmessage.Type{util.QueryTypeSrv, util.QueryTypeAAAA, util.QueryTypeA}
	client, err = NewClientDnsConnection(testDomain, comm)
	require.NoError(t, err)
	err = client.AutoDetectQueryType()
	require.NoError(t, err)

	comm = &testCommunicator{}
	srv = NewServerDnsListener(testDomain, comm)
	srv.allowedQueryTypes = []dnsmessage.Type{util.QueryTypeA}
	client, err = NewClientDnsConnection(testDomain, comm)
	require.NoError(t, err)
	err = client.AutoDetectQueryType()
	require.NoError(t, err)

}

func Test_Handshake(t *testing.T) {

	comm := &testCommunicator{}

	NewServerDnsListener(testDomain, comm)
	client, err := NewClientDnsConnection(testDomain, comm)
	require.NoError(t, err)

	err = client.AutoDetectQueryType()
	require.NoError(t, err)

	_, err = client.VersionHandshake()
	require.NoError(t, err)
}
