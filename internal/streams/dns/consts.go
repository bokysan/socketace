package dns

import (
	"time"
)

const (
	ProtocolVersion        = 0x00001000
	DefaultUpstreamMtuSize = 0xFF
)

func secs(i int) time.Duration {
	return time.Second * time.Duration(i)
}
