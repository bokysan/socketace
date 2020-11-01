package dns

import (
	"github.com/miekg/dns"
	"strings"
)

const TypeSocketAce uint16 = 0xFFA0

// A crazy new RR type :)
type SocketAcePrivate struct {
	data []byte
}

func init() {
	dns.PrivateHandle("SOCKETACE", TypeSocketAce, NewSocketAcePrivate)
}

func NewSocketAcePrivate() dns.PrivateRdata { return &SocketAcePrivate{} }

func (rd *SocketAcePrivate) Len() int {
	if rd.data != nil {
		return len(rd.data)
	}
	return 0
}

func (rd *SocketAcePrivate) String() string {
	if rd.data != nil {
		return string(rd.data)
	}
	return ""
}

func (rd *SocketAcePrivate) Parse(txt []string) error {
	rd.data = []byte(strings.Join(txt, ""))
	return nil
}

func (rd *SocketAcePrivate) Pack(buf []byte) (int, error) {
	n := copy(buf, rd.data)
	if n != len(rd.data) {
		return n, dns.ErrBuf
	}
	return n, nil
}

func (rd *SocketAcePrivate) Unpack(buf []byte) (int, error) {
	rd.data = make([]byte, len(buf))
	return copy(rd.data, buf), nil
}

func (rd *SocketAcePrivate) Copy(dest dns.PrivateRdata) error {
	scrr, ok := dest.(*SocketAcePrivate)
	if !ok {
		return dns.ErrRdata
	}
	scrr.data = rd.data
	return nil
}
