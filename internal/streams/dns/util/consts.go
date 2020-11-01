package util

import (
	"github.com/bokysan/socketace/v2/internal/util/enc"
	"golang.org/x/net/dns/dnsmessage"
)

const (
	HostnameMaxLen = 253 // including the final period, and the non-printed zero octet for root makes 255
	LabelMaxlen    = 60  // See https://stackoverflow.com/questions/32290167/what-is-the-maximum-length-of-a-dns-name

	// DownloadCodecCheck is the hard coded string available on both client and server which
	// is used to check if the downstream codec works properly or not.
	DownloadCodecCheck = "\000\000\000\000\377\377\377\377\125\125\125\125\252\252\252\252" +
		"\201\143\310\322\307\174\262\027\137\117\316\311\111\055\122\041" +
		"\141\251\161\040\045\263\006\163\346\330\104\060\171\120\127\277"
)

type DownstreamConfig struct {
	MtuSize *int        // -m max size of downstream fragments (default: autodetect)
	Encoder enc.Encoder // -O force downstream encoding for -T other than NULL: Base32Encoding, Base64Encoding, Base64uEncoding, Base128Encoding, or (only for TXT:) RawEncoding (default: autodetect)
}

type UpstreamConfig struct {
	MtuSize   int              // -M max size of upstream hostnames (~100-255, default: 255)
	Encoder   enc.Encoder      // -O force downstream encoding for -T other than NULL: Base32, Base64, Base64u,  Base128, or (only for TXT:) Raw  (default: autodetect)
	QueryType *dnsmessage.Type // -T force dns type: QueryTypeNull, QueryTypePrivate, QueryTypeTxt, QueryTypeSrv, QueryTypeMx, QueryTypeCname, QueryTypeAAAA, QueryTypeA (default: autodetect)
}
