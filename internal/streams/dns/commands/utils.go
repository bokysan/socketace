package commands

import (
	"github.com/bokysan/socketace/v2/internal/util/enc"
	"github.com/miekg/dns"
	log "github.com/sirupsen/logrus"
	"math/rand"
	"regexp"
	"sort"
	"strconv"
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

var Digits = regexp.MustCompile("^[0-9]{3}")

// StripDomain will remove the domain from the end of data string and return the string without this domain.
// If the string does not end with the domain, it does nothing.
func StripDomain(data []byte, domain string) (res []byte) {
	if strings.HasSuffix(strings.ToLower(string(data)), "."+strings.ToLower(domain)+".") {
		l2 := len(data)
		l1 := len(domain) + 2
		data = data[0 : l2-l1]
	}

	for len(data) > 0 {
		if c := data[0]; c == '.' {
			// Skip dots in the name
			data = data[1:]
		} else if c != '\\' {
			// Add escaped char as-is
			res = append(res, c)
			data = data[1:]
		} else if Digits.MatchString(string(data[1:])) {
			// Parse ascii escapes
			digits := string(data[1:4])
			num, err := strconv.ParseInt(digits, 10, 16)
			if err != nil {
				log.WithError(err).Errorf(
					"Failed to parse %q at position #%d to number -- ignoring",
					num,
					len(res),
				)
			} else {
				res = append(res, byte(num))
			}
			data = data[4:]
		} else {
			// Add char normally
			res = append(res, data[1])
			data = data[2:]
		}
	}

	return
}

// ComposeRequest will take a DNS message and recompose it back to a complete request, if multiQuery was used
func ComposeRequest(msg *dns.Msg, domain string) (data []byte) {
	if l := len(msg.Question); l > 1 {
		log.Debugf("Multi-query request, len=%q", l)
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
			s := []byte(v.Name[2:])
			s = StripDomain(s, domain)
			data = append(data, s...)
		}
	} else {
		s := []byte(msg.Question[0].Name)
		s = StripDomain(s, domain)
		data = append(data, s...)
	}
	return data
}
