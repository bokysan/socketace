package util

import (
	"github.com/miekg/dns"
	"strings"
)

const TypeSocketAce uint16 = 0xFFA0

// A crazy new RR type :)
type SocketAcePrivate struct {
	Data []byte
}

func init() {
	dns.PrivateHandle("SOCKETACE", TypeSocketAce, NewSocketAcePrivate)
}

func NewSocketAcePrivate() dns.PrivateRdata { return &SocketAcePrivate{} }

func (rd *SocketAcePrivate) Len() int {
	if rd.Data != nil {
		return len(rd.Data)
	}
	return 0
}

func (rd *SocketAcePrivate) String() string {
	if rd.Data != nil {
		return string(rd.Data)
	}
	return ""
}

func (rd *SocketAcePrivate) Parse(txt []string) error {
	rd.Data = []byte(strings.Join(txt, ""))
	return nil
}

func (rd *SocketAcePrivate) Pack(buf []byte) (int, error) {
	n := copy(buf, rd.Data)
	if n != len(rd.Data) {
		return n, dns.ErrBuf
	}
	return n, nil
}

func (rd *SocketAcePrivate) Unpack(buf []byte) (int, error) {
	rd.Data = make([]byte, len(buf))
	return copy(rd.Data, buf), nil
}

func (rd *SocketAcePrivate) Copy(dest dns.PrivateRdata) error {
	scrr, ok := dest.(*SocketAcePrivate)
	if !ok {
		return dns.ErrRdata
	}
	scrr.Data = rd.Data
	return nil
}
