package dns

import (
	"time"
)

const (
	ProtocolVersion        = 0x00001000
	DefaultUpstreamMtuSize = 0xFF
)

type Packet struct {
	SeqNo    uint16
	Fragment uint8
	Data     []byte
}

func secs(i int) time.Duration {
	return time.Second * time.Duration(i)
}
