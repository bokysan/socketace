package util

import (
	"github.com/bokysan/socketace/v2/internal/util/enc"
	log "github.com/sirupsen/logrus"
	"golang.org/x/net/dns/dnsmessage"
	"math"
)

const (
	HostnameMaxLen = 253 // including the final period, and the non-printed zero octet for root makes 255
	LabelMaxlen    = 60  // See https://stackoverflow.com/questions/32290167/what-is-the-maximum-length-of-a-dns-name
)

var (
	// DownloadCodecCheck is the hard coded string available on both client and server which
	// is used to check if the downstream codec works properly or not.
	DownloadCodecCheck = []byte(
		"\000\000\000\000\377\377\377\377\125\125\125\125\252\252\252\252" +
			"\201\143\310\322\307\174\262\027\137\117\316\311\111\055\122\041" +
			"\141\251\161\040\045\263\006\163\346\330\104\060\171\120\127\277")
)

type DownstreamConfig struct {
	FragmentSize uint32      // -m max size of downstream fragments (default: autodetect)
	Encoder      enc.Encoder // -O force downstream encoding for -T other than NULL: Base32Encoding, Base64Encoding, Base64uEncoding, Base128Encoding, or (only for TXT:) RawEncoding (default: autodetect)
}

type UpstreamConfig struct {
	FragmentSize uint32           // -M max size of upstream hostnames (~100-255, default: 255)
	Encoder      enc.Encoder      // -O force downstream encoding for -T other than NULL: Base32, Base64, Base64u,  Base128, or (only for TXT:) Raw  (default: autodetect)
	QueryType    *dnsmessage.Type // -T force dns type: QueryTypeNull, QueryTypePrivate, QueryTypeTxt, QueryTypeSrv, QueryTypeMx, QueryTypeCname, QueryTypeAAAA, QueryTypeA (default: autodetect)
}

// GetLongestDataString returns the longest data string available, when all dots and domain are included in the calculation
func GetLongestDataString(domain string) int {

	// Available space is maximum query length
	space := HostnameMaxLen
	// minus domain length minus dot before and after domain
	space = space - len(domain) - 2

	// minus command len
	space = space - 1

	// minus all dots that need to be inserted
	space = space - int(math.Ceil(float64(space)/float64(LabelMaxlen)))

	return space
}

// PrepareHostname will finalize hostname -- add dots in the name, if needed. It will verify that the total
// lenght of the hostname does not exiceed HostnameMaxLen and throw an error it it does.
func PrepareHostname(data []byte, domain string) ([]byte, error) {
	if len(data) > LabelMaxlen {
		data = Dotify(data)
	}
	hostname := make([]byte, 0)
	hostname = append(hostname, data...)
	hostname = append(hostname, '.')
	hostname = append(hostname, []byte(domain)...)
	hostname = append(hostname, '.')

	if l := len(hostname); l > HostnameMaxLen-2 {
		log.Warnf("Token len %d exceedes %d: %v", l, HostnameMaxLen-2, hostname)
		return []byte{}, ErrTooLong
	}

	return hostname, nil
}
