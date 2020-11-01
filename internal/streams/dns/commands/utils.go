package commands

import (
	"crypto/rand"
	"encoding/binary"
	"github.com/bokysan/socketace/v2/internal/streams/dns/util"
	"github.com/bokysan/socketace/v2/internal/util/enc"
	"strings"
)

// randomChars returns three random characters which will make sure that the request is not cached
func randomChars() (string, error) {
	/* Add lower 15 bits of rand seed as base32, followed by a dot and the tunnel domain and send */
	seed := make([]byte, 3)
	if err := binary.Read(rand.Reader, binary.LittleEndian, &seed); err != nil {
		return "", err
	}

	seed[0] = enc.ByteToBase32Char(seed[0])
	seed[1] = enc.ByteToBase32Char(seed[1])
	seed[2] = enc.ByteToBase32Char(seed[2])

	return string(seed), nil
}

// prepareHostname will finalize hostname -- add dots in the name, if needed. It will verify that the total
// lenght of the hostname does not exiceed HostnameMaxLen and throw an error it it does.
func prepareHostname(data, domain string) (string, error) {
	if len(data) > util.LabelMaxlen {
		data = util.Dotify(data)
	}
	hostname := data + "." + domain
	if len(hostname) > util.HostnameMaxLen-2 {
		return "", util.ErrTooLong
	}

	return hostname, nil
}

// stripDomain will remove the domain from the end of data string and return the string without this domain.
// If the string does not end with the domain, it does nothing.
func stripDomain(data, domain string) string {
	if strings.HasSuffix(strings.ToLower(data), "."+strings.ToLower(domain)) {
		l2 := len(data)
		l1 := len(domain) + 1
		return util.Undotify(data[0 : l2-l1])
	} else {
		return util.Undotify(data)
	}
}
