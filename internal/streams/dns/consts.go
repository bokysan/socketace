package dns

import (
	"github.com/bokysan/socketace/v2/internal/util/enc"
	"github.com/pkg/errors"
)

const (
	ProtocolVersion        = 0x00001000
	DefaultUpstreamMtuSize = 0xFF
	HostnameMaxLen         = 253 // including the final period, and the non-printed zero octet for root makes 255
	LabelMaxlen            = 60  // See https://stackoverflow.com/questions/32290167/what-is-the-maximum-length-of-a-dns-name

	// downloadCodecCheck is the hard coded string available on both client and server which
	// is used to check if the downstream codec works properly or not.
	downloadCodecCheck = "\000\000\000\000\377\377\377\377\125\125\125\125\252\252\252\252" +
		"\201\143\310\322\307\174\262\027\137\117\316\311\111\055\122\041" +
		"\141\251\161\040\045\263\006\163\346\330\104\060\171\120\127\277"
)

type ClientServerResponse string

func (cse *ClientServerResponse) Error() string {
	return string(*cse)
}

// Is will check if the first few bytes of the suppied byte array match the given exception.
func (cse *ClientServerResponse) Is(data []byte) bool {
	return len(data) >= len(*cse) && string(data[:len(*cse)]) == string(*cse)
}

// Strip will remove the prefix of the ClientServerResponse from the given stream. No validation takes place.
func (cse *ClientServerResponse) Strip(data []byte) []byte {
	return data[len(*cse):]
}

// Declare a few of standard error responses
var (
	BadVersion    ClientServerResponse = "BADVER"
	BadLen        ClientServerResponse = "BADLEN"
	BadIp         ClientServerResponse = "BADIP"
	BadCodec      ClientServerResponse = "BADCODEC"
	BadFrag       ClientServerResponse = "BADFRAG"
	BadServerFull ClientServerResponse = "VFUL"
	VersionOk     ClientServerResponse = "VACK"
	LazyModeOk    ClientServerResponse = "LACK"
	VersionNotOk  ClientServerResponse = "VNAK"
)

var BadErrors = []ClientServerResponse{
	BadVersion, BadLen, BadIp, BadCodec, BadFrag, BadServerFull,
}

type Packet struct {
	SeqNo    uint16
	Fragment uint8
	Data     []byte
}

var (
	ErrTooLong  = errors.New("token too long")
	ErrCaseSwap = errors.New("case swap, no need for any further test")
)

// Declare a list of encodings
var (
	Base32Encoding  enc.Encoder = &enc.Base32Encoder{}
	Base64Encoding  enc.Encoder = &enc.Base64Encoder{}
	Base64uEncoding enc.Encoder = &enc.Base64uEncoder{}
	Base85Encoding  enc.Encoder = &enc.Base85Encoder{}
	Base91Encoding  enc.Encoder = &enc.Base91Encoder{}
	Base128Encoding enc.Encoder = &enc.Base128Encoder{}
	RawEncoding     enc.Encoder = &enc.RawEncoder{}
)
