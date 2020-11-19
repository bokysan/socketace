package upstream

import (
	"bufio"
	"bytes"
	"github.com/bokysan/socketace/v2/internal/socketace"
	"github.com/bokysan/socketace/v2/internal/streams"
	"github.com/bokysan/socketace/v2/internal/streams/dns"
	"github.com/bokysan/socketace/v2/internal/util/addr"
	"github.com/bokysan/socketace/v2/internal/util/cert"
	dns2 "github.com/miekg/dns"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"os/exec"
	"runtime"
	"strings"
)

// Dns will create a client which will establish a SocketAce connection via DNS request-response loop. This is not
// the fastest nor the most optimal way to establish connections. It is, however, the only way to do it in some cases.
// You won't be able to stream movies, but this should be sufficient to connect to a remote terminal session, check
// your email and do some light browsing.
//
// DNS connections are a bit more complex as a lot of probing needs to be done before the connection is established.
// The steps to establish a DNS connection are as follows:
// - Try connecting directly via TCP to our target host (if direct connection is allowed). Failing that,
// - Try connecting directly via UDP to our target host. Failing that,
// - Use user supplied DNS servers, if they are available. And finally,
// - Use the system provided DNS servers
//
type Dns struct {
	streams.Connection

	// Address is the parsed representation of the address and calculated automatically while unmarshalling
	Address addr.ProtoAddress
}

func (ups *Dns) String() string {
	return ups.Address.String()
}

func (ups *Dns) Connect(manager cert.TlsConfig, mustSecure bool) error {

	if ups.Address.Scheme != "dns" {
		return errors.Errorf("DNS can only handle 'dns' schemes. Cannot handle: %q", ups.Address.String())
	}

	topDomain := ups.Address.Hostname()

	var err error

	addDirect := true
	if x, ok := ups.Address.Query()["direct"]; ok {
		if strings.ToLower(x[0]) == "false" {
			addDirect = false
		}
	}

	servers := make(dns.AddressList, 0)
	if addDirect {
		servers.ResolveAndAddAddress(topDomain)
	}

	// Get a list of alternate DNS servers
	if x, ok := ups.Address.Query()["dns"]; ok {
		// Allow for ?dns=1.2.3.4&dns=5.6.7.8
		for _, y := range x {
			// And for ?dns=1.2.3.4,5.6.7.8 syntax
			for _, z := range strings.Split(y, ",") {
				servers.ResolveAndAddAddress(z)
			}
		}
	}

	if runtime.GOOS == "windows" {
		log.Debugf("Adding servers from ipconfig /all")
		ups.addWindowsDnsServers(servers)
	} else if runtime.GOOS == "darwin" {
		log.Debugf("Adding servers from scutil --dns")
		ups.addDarwinDnsServers(servers)
	}

	c, err := dns2.ClientConfigFromFile("/etc/resolv.conf")
	if err == nil {
		log.Debugf("Adding servers from /etc/resolv.conf")
		for _, server := range c.Servers {
			servers.ResolveAndAddAddress(server)
		}
	}

	conf := &dns.ClientConfig{
		Servers: servers,
	}

	var conn *dns.ClientDnsConnection

main:
	for true {
		var comm dns.ClientCommunicator
		var err error

		comm, err = dns.NewNetConnectionClientCommunicator(conf)
		if err != nil {
			return err
		}

		conn, err = dns.NewClientDnsConnection(topDomain, comm)
		if err != nil {
			return err
		}

		if err = conn.Handshake(); err != nil {
			if len(conf.Servers) == 1 {
				// Nothing more to do, give up
				return err
			}

			// There can be a case where the server responds but does not allow our queries to go through.
			// In such case we should remove the server from the list and with the reduced list
			for i := len(servers) - 1; i >= 0; i-- {
				if servers[i].String() == comm.RemoteAddr().String() && servers[i].Network() == comm.RemoteAddr().Network() {
					conf.Servers = servers[i+1:]
					if len(conf.Servers) == 0 {
						return errors.Wrapf(err, "No more servers to try, sorry.")
					} else {
						continue main
					}
				}
			}

			return errors.Wrapf(err, "Tried all servers on the list, but no success -- is DNS opened: %v", servers)
		} else {
			break
		}
	}

	if conn == nil {
		return errors.Errorf("Connection not established!")
	}

	cc, err := socketace.NewClientConnection(conn, manager, false, ups.Address.Host)
	if err != nil {
		return errors.Wrapf(err, "Could not open connection")
	} else if mustSecure && !cc.Secure() {
		return errors.Errorf("Could not establish a secure connection to %v", ups.Address)
	}

	ups.Connection = streams.NewNamedConnection(streams.NewNamedConnection(cc, ups.Address.String()), "dns")

	return nil
}

// Capturing the output of ipconfig command is not really the nicest way to go about it, but for the time being
// it will need to do.
func (ups *Dns) addWindowsDnsServers(servers dns.AddressList) {
	buf := &bytes.Buffer{}
	cmd := exec.Command("ipconfig", "/all")
	cmd.Stdout = buf
	err := cmd.Run()
	if err != nil {
		log.Debugf("Can't run ipconfig -- will ignore output: %v", err)
	} else {
		scanner := bufio.NewScanner(buf)
		inDns := false
		for scanner.Scan() {
			if err := scanner.Err(); err != nil {
				log.Warnf("Failed parsing output from scutil: %v", err)
				break
			}
			line := scanner.Text()

			if inDns {
				if strings.Contains(line, ". :") {
					inDns = false
				} else {
					servers.ResolveAndAddAddress(strings.TrimSpace(line))
				}
			} else if strings.HasPrefix(strings.TrimSpace(line), "DNS Servers") {
				d := strings.Split(line, ":")
				if len(d) > 1 {
					inDns = true
					servers.ResolveAndAddAddress(strings.TrimSpace(d[1]))
				} else {
					log.Warnf("Invalid inline in ipconfig response: %q -- ignoring segment", line)
				}
			}
		}
	}
}

// Capturing the output of scutil command is not really the nicest way to go about it, but for the time being
// it will need to do.
func (ups *Dns) addDarwinDnsServers(servers dns.AddressList) {
	buf := &bytes.Buffer{}
	cmd := exec.Command("scutil", "--dns")
	cmd.Stdout = buf
	err := cmd.Run()
	if err != nil {
		log.Debugf("Can't run scutil -- will ignore output: %v", err)
	} else {
		scanner := bufio.NewScanner(buf)
		for scanner.Scan() {
			if err := scanner.Err(); err != nil {
				log.Warnf("Failed parsing output from scutil: %v", err)
				break
			}
			line := scanner.Text()
			f := strings.Fields(line)

			// Look for
			//   nameserver[0] : 192.168.8.1
			if len(f) < 3 {
				continue
			}

			if strings.HasPrefix(f[0], "nameserver") {
				servers.ResolveAndAddAddress(f[2])
			}

		}
	}
}
