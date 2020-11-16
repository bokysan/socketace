package commands

import (
	"github.com/bokysan/socketace/v2/internal/streams/dns/util"
	"github.com/bokysan/socketace/v2/internal/util/enc"
	"github.com/miekg/dns"
	"math/rand"
	"sort"
	"strings"
)

// randomChars returns three random characters which will make sure that the request is not cached
func randomChars() string {
	const BASE36 = "abcdefghijklmnopqrstuvwxyz0123456789"

	seed := make([]byte, 3)
	seed[0] = BASE36[rand.Intn(36)]
	seed[1] = BASE36[rand.Intn(36)]
	seed[2] = BASE36[rand.Intn(36)]

	return string(seed)
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

// ComposeRequest will take a DNS message and recompose it back to a complete request, if multiQuery was used
func ComposeRequest(msg *dns.Msg, domain string) string {
	var data string
	if len(msg.Question) > 1 {
		questions := append([]dns.Question{}, msg.Question...)
		sort.Slice(questions, func(i, j int) bool {
			i1 := enc.Base32CharToInt(questions[i].Name[0])
			i2 := enc.Base32CharToInt(questions[i].Name[1])
			j1 := enc.Base32CharToInt(questions[j].Name[0])
			j2 := enc.Base32CharToInt(questions[j].Name[1])
			return i1+i2*32 < j1+j2*32
		})
		for _, v := range questions {
			// remove first two characters
			data += stripDomain(v.Name, domain)[2:]
		}
	} else {
		data = stripDomain(msg.Question[0].Name, domain)
	}
	return data
}
