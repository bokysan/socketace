package dns

import (
	"time"
)

const (
	ProtocolVersion        = 0x00001000
	DefaultUpstreamMtuSize = 0xFF

	// downloadCodecCheck is the hard coded string available on both client and server which
	// is used to check if the downstream codec works properly or not.
	downloadCodecCheck = "\000\000\000\000\377\377\377\377\125\125\125\125\252\252\252\252" +
		"\201\143\310\322\307\174\262\027\137\117\316\311\111\055\122\041" +
		"\141\251\161\040\045\263\006\163\346\330\104\060\171\120\127\277"
)

type Packet struct {
	SeqNo    uint16
	Fragment uint8
	Data     []byte
}

func secs(i int) time.Duration {
	return time.Second * time.Duration(i)
}
