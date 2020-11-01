package dns

import (
	"crypto/rand"
	"encoding/binary"
	"github.com/bokysan/socketace/v2/internal/util/enc"
	"github.com/miekg/dns"
	"github.com/pkg/errors"
	"golang.org/x/net/dns/dnsmessage"
	"sort"
	"strings"
)

// Request represents a (serialized) request to a DNS server
type Request interface {
	// Command is the command that this request reffers to
	Command() Command
	// Encode will encode this requires into a DNS-compatible query, potentially using the encoder specified. Note that
	// encoding allways happens in the "hostname/domain" format -- e.g. so you can execute a A, CNAME, or a MX query
	// whith this.
	Encode(e enc.Encoder, domain string) (string, error)
	// Decode will decode the data from the query into this object
	Decode(e enc.Encoder, request, domain string) error
}

// Response it the response from the DNS server
type Response interface {
	// Command is the command that this request reffers to
	Command() Command
	// EncodeResponse will encode this response into a data stream which can bi the sent as a DNS response.
	Encode(e enc.Encoder, queryType dnsmessage.Type) (*dns.Msg, error)
	// DecodeResponse will take a byte (data) stream and create a response object
	Decode(e enc.Encoder, msg *dns.Msg) error
}

// randomChars returns three random characters which will make sure that the request is not cached
func randomChars() (string, error) {
	/* Add lower 15 bits of rand seed as base32, followed by a dot and the tunnel domain and send */
	seed := make([]byte, 3)
	if err := binary.Read(rand.Reader, binary.LittleEndian, &seed); err != nil {
		return "", err
	}

	seed[0] = enc.ByteToBase32Char(seed[0])
	seed[1] = enc.ByteToBase32Char(seed[1])
	seed[2] = enc.ByteToBase32Char(seed[2])

	return string(seed), nil
}

// prepareHostname will finalize hostname -- add dots in the name, if needed. It will verify that the total
// lenght of the hostname does not exiceed HostnameMaxLen and throw an error it it does.
func prepareHostname(data, domain string) (string, error) {
	if len(data) > LabelMaxlen {
		data = Dotify(data)
	}
	hostname := data + "." + domain
	if len(hostname) > HostnameMaxLen-2 {
		return "", ErrTooLong
	}

	return hostname, nil
}

// stripDomain will remove the domain from the end of data string and return the string without this domain.
// If the string does not end with the domain, it does nothing.
func stripDomain(data, domain string) string {
	if strings.HasSuffix(strings.ToLower(data), "."+strings.ToLower(domain)) {
		l2 := len(data)
		l1 := len(domain) + 1
		return data[0 : l2-l1]
	} else {
		return data
	}
}

// TypePriority calculates the supplied type's priority used for sorting
func TypePriority(rr dns.RR) uint32 {
	switch v := rr.(type) {
	case *dns.NULL:
		// first two bytes represent the order
		return 10000 + uint32(binary.LittleEndian.Uint16([]byte(v.Data[0:2])))
	case *dns.PrivateRR:
		// first two bytes represent the order
		return 20000 + uint32(binary.LittleEndian.Uint16([]byte(v.Data.String()[0:2])))
	case *dns.TXT:
		// First two characters represent the byte order
		i1 := enc.Base32CharToInt(v.Txt[0][0])
		i2 := enc.Base32CharToInt(v.Txt[0][1])
		return 30000 + uint32(i1+i2*32)
	case *dns.MX:
		// Use Preference for order
		return 40000 + uint32(v.Preference)
	case *dns.SRV:
		// Use Priority for order
		return 50000 + uint32(v.Priority)
	case *dns.CNAME:
		// First two characters represent the order
		i1 := enc.Base32CharToInt(v.Target[0])
		i2 := enc.Base32CharToInt(v.Target[1])
		return 60000 + uint32(i1+i2*32)
	case *dns.AAAA:
		// First two bytes represent the order
		return 70000 + uint32(binary.LittleEndian.Uint16(v.AAAA[0:2]))
	case *dns.A:
		// First byte represent the order
		return 80000 + uint32(v.A[0])
	}

	// Unknown response type
	return 90000
}

func EncodeDnsResponse(data []byte, queryType dnsmessage.Type) (*dns.Msg, error) {
	switch queryType {
	case QueryTypeNull:
		return EncodeDnsResponseNull(data)
	case QueryTypePrivate:
		return EncodeDnsResponsePrivate(data)
	case QueryTypeTxt:
		return EncodeDnsResponseTxt(data)
	case QueryTypeMx:
		return EncodeDnsResponseMx(data)
	case QueryTypeSrv:
		return EncodeDnsResponseSrv(data)
	case QueryTypeCname:
		return EncodeDnsResponseCname(data)
	case QueryTypeAAAA:
		return EncodeDnsResponseAAAA(data)
	case QueryTypeA:
		return EncodeDnsResponseA(data)
	}

	return nil, errors.Errorf("Unknown query type: %v", queryType)
}

func EncodeDnsResponseA(data []byte) (*dns.Msg, error) {
	msg := &dns.Msg{}
	msg.Authoritative = true

	order := uint16(0)

	for len(data) > 0 {
		order += 1
		d := make([]byte, 1)

		if order > 255 {
			return nil, errors.Errorf("Message too long.")
		}

		d[0] = byte(order)

		if len(data) > 3 {
			d = append(d, data[0:3]...)
			data = data[3:]
		} else {
			d = append(d, data...)
			data = data[0:0]
		}
		msg.Answer = append(msg.Answer, &dns.A{
			Hdr: dns.RR_Header{
				Rrtype:   uint16(QueryTypeA),
				Ttl:      1,
				Rdlength: uint16(len(d)),
			},
			A: d,
		})
	}

	return msg, nil
}

func EncodeDnsResponseAAAA(data []byte) (*dns.Msg, error) {
	msg := &dns.Msg{}
	msg.Authoritative = true

	order := uint16(0)

	for len(data) > 0 {
		order += 1
		d := make([]byte, 2)

		// First two characters represent the byte order
		binary.LittleEndian.PutUint16(d, order)

		if len(data) > 14 {
			d = append(d, data[0:14]...)
			data = data[14:]
		} else {
			d = append(d, data...)
			data = data[0:0]
		}
		msg.Answer = append(msg.Answer, &dns.AAAA{
			Hdr: dns.RR_Header{
				Rrtype:   uint16(QueryTypeAAAA),
				Ttl:      1,
				Rdlength: uint16(len(d)),
			},
			AAAA: d,
		})
	}

	return msg, nil
}

func EncodeDnsResponseCname(data []byte) (*dns.Msg, error) {
	msg := &dns.Msg{}
	msg.Authoritative = true

	order := uint16(0)

	for len(data) > 0 {
		order += 1
		d := make([]byte, 2)

		// First two characters represent the byte order
		d[0] = enc.IntToBase32Char(int(order))
		d[1] = enc.IntToBase32Char(int(order) >> 4)

		if len(data) > HostnameMaxLen-3 {
			d = append(d, data[0:HostnameMaxLen-3]...)
			data = data[HostnameMaxLen-3:]
		} else {
			d = append(d, data...)
			data = data[0:0]
		}
		msg.Answer = append(msg.Answer, &dns.CNAME{
			Hdr: dns.RR_Header{
				Rrtype:   uint16(QueryTypeCname),
				Ttl:      1,
				Rdlength: uint16(len(d)),
			},
			Target: string(d),
		})
	}

	return msg, nil
}
func EncodeDnsResponseSrv(data []byte) (*dns.Msg, error) {
	msg := &dns.Msg{}
	msg.Authoritative = true

	order := uint16(0)

	for len(data) > 0 {
		var d []byte
		order += 1

		if len(data) > HostnameMaxLen-3 {
			d = data[0 : HostnameMaxLen-3]
			data = data[HostnameMaxLen-3:]
		} else {
			d = data
			data = data[0:0]
		}
		msg.Answer = append(msg.Answer, &dns.SRV{
			Hdr: dns.RR_Header{
				Rrtype:   uint16(QueryTypeSrv),
				Ttl:      1,
				Rdlength: uint16(len(d)),
			},
			Priority: order,
			Target:   string(d),
		})
	}

	return msg, nil
}

func EncodeDnsResponseMx(data []byte) (*dns.Msg, error) {
	msg := &dns.Msg{}
	msg.Authoritative = true

	order := uint16(0)

	for len(data) > 0 {
		var d []byte
		order += 10 // MX servers usually skip by 10

		if len(data) > HostnameMaxLen-3 {
			d = data[0 : HostnameMaxLen-3]
			data = data[HostnameMaxLen-3:]
		} else {
			d = data
			data = data[0:0]
		}
		msg.Answer = append(msg.Answer, &dns.MX{
			Hdr: dns.RR_Header{
				Rrtype:   uint16(QueryTypeMx),
				Ttl:      1,
				Rdlength: uint16(len(d)),
			},
			Preference: order,
			Mx:         string(d),
		})
	}

	return msg, nil
}

func EncodeDnsResponseTxt(data []byte) (*dns.Msg, error) {
	msg := &dns.Msg{}
	msg.Authoritative = true

	order := uint16(0)

	// Max len for TXT record type is 255 octects
	var d []byte
	txtData := make([]string, 0)

	for len(data) > 0 {
		if len(txtData) == 0 {
			d = make([]byte, 2)

			// First two characters represent the byte order
			d[0] = enc.IntToBase32Char(int(order))
			d[1] = enc.IntToBase32Char(int(order) >> 4)

			order += 1
		} else {
			d = make([]byte, 0)
		}

		// Limit strings to 255 characters
		if len(data) > 253 {
			d = append(d, data[0:253]...)
			data = data[253:]
		} else {
			d = append(d, data...)
			data = data[0:0]
		}

		txtData = append(txtData, string(d))

		// Limit answer to 250 strings
		if len(txtData) == 250 {
			msg.Answer = append(msg.Answer, &dns.TXT{
				Hdr: dns.RR_Header{
					Rrtype:   uint16(QueryTypeTxt),
					Ttl:      1,
					Rdlength: uint16(len(strings.Join(txtData, ""))),
				},
				Txt: txtData,
			})
			txtData = make([]string, 0)
		}
	}
	if len(txtData) > 0 {
		msg.Answer = append(msg.Answer, &dns.TXT{
			Hdr: dns.RR_Header{
				Rrtype:   uint16(QueryTypeTxt),
				Ttl:      1,
				Rdlength: uint16(len(strings.Join(txtData, ""))),
			},
			Txt: txtData,
		})
	}

	return msg, nil
}

func EncodeDnsResponsePrivate(data []byte) (*dns.Msg, error) {
	msg := &dns.Msg{}
	msg.Authoritative = true

	// Max len for NULL record type is 65535 octects
	order := uint16(0)

	for len(data) > 0 {
		d := make([]byte, 2)
		order += 1
		binary.LittleEndian.PutUint16(d, order)

		if len(data) > 65530 {
			d = append(d, data[0:65530]...)
			data = data[65530:]
		} else {
			d = append(d, data...)
			data = data[0:0]
		}
		msg.Answer = append(msg.Answer, &dns.PrivateRR{
			Hdr: dns.RR_Header{
				Rrtype:   uint16(QueryTypePrivate),
				Ttl:      1,
				Rdlength: uint16(len(d)),
			},
			Data: &SocketAcePrivate{
				data: d,
			},
		})
	}
	return msg, nil
}

func EncodeDnsResponseNull(data []byte) (*dns.Msg, error) {
	msg := &dns.Msg{}
	msg.Authoritative = true

	// Max len for NULL record type is 65535 octects
	order := uint16(0)

	for len(data) > 0 {
		d := make([]byte, 2)
		order += 1
		binary.LittleEndian.PutUint16(d, order)

		if len(data) > 65530 {
			d = append(d, data[0:65530]...)
			data = data[65530:]
		} else {
			d = append(d, data...)
			data = data[0:0]
		}
		msg.Answer = append(msg.Answer, &dns.NULL{
			Hdr: dns.RR_Header{
				Rrtype:   uint16(QueryTypeNull),
				Ttl:      1,
				Rdlength: uint16(len(d)),
			},
			Data: string(d),
		})
	}
	return msg, nil
}

// DecodeDnsResponse will decode the DNS message and return the plain text response
func DecodeDnsResponse(q *dns.Msg) string {
	resp := make([]string, 0)
	answers := append([]dns.RR{}, q.Answer...)

	sort.Slice(answers, func(i, j int) bool {
		return TypePriority(answers[i]) < TypePriority(answers[j])
	})

	for _, rr := range answers {
		switch v := rr.(type) {
		case *dns.NULL:
			// Remove first two bytes
			resp = append(resp, v.Data[2:])
		case *dns.PrivateRR:
			// Remove first two bytes
			resp = append(resp, v.Data.String()[2:])
		case *dns.TXT:
			resp = append(resp, strings.Join(v.Txt, "")[2:])
		case *dns.MX:
			// Nothing to remove, Preference takes care of it
			resp = append(resp, v.Mx)
		case *dns.SRV:
			// Nothing to remove, Priority takes care of it
			resp = append(resp, v.Target)
		case *dns.CNAME:
			// Remove first two characters
			resp = append(resp, v.Target[2:])
		case *dns.AAAA:
			// Remove first two bytes
			resp = append(resp, string(v.AAAA[2:]))
		case *dns.A:
			// Remove first byte
			resp = append(resp, string(v.A[1:]))
		}
	}

	return strings.Join(resp, "")
}

type Command byte
type Commands []Command

type LazyMode byte

const (
	// Command 0123456789abcdef are reserved for user IDs
	CmdLogin                     Command = 'l'
	CmdPing                      Command = 'p'
	CmdTestFragmentSize          Command = 'r'
	CmdSetDownstreamFragmentSize Command = 'n'
	CmdTestDownstreamEncoder     Command = 'y'
	CmdSetDownstreamEncoder      Command = 'o'
	CmdTestUpstreamEncoder       Command = 'z'
	CmdSetUpstreamEncoder        Command = 's'
	CmdTestMultiQuery            Command = 'm'
)

const (
	LazyModeOn  LazyMode = 'l'
	LazyModeOff LazyMode = 'i'
)

// RequiresUser returs true if the command requires the user ID
func (c Command) RequiresUser() bool {
	return c == CmdPing ||
		c == CmdSetDownstreamFragmentSize ||
		c == CmdSetDownstreamEncoder ||
		c == CmdTestFragmentSize ||
		c == CmdTestUpstreamEncoder

}

// ExpectsEmptyReply will return true if the command expects empty reply (no data
func (c Command) ExpectsEmptyReply() bool {
	return c == CmdVersion || c == CmdTestDownstreamEncoder || c == CmdTestMultiQuery
}

// String will retun the command code as string, e.g. 'z', 's', 'v'...
func (c Command) String() string {
	return string(c)
}

// ValidateType will check if the supplied string starts with the given command type and return an error if its not.
func (c Command) ValidateType(data string) error {
	if !c.IsOfType(data) {
		return errors.Errorf("Invalid command type. Expected %v, got, %v", c, data[0])
	}
	return nil
}

// IsOfType will check if the supplied string starts with the given command type
func (c Command) IsOfType(data string) bool {
	if len(data) < 0 {
		return false
	}
	if data[0] == uint8(c) {
		return true
	}
	if strings.ToLower(data[0:1])[0] == uint8(c) {
		return true
	}
	return false
}

// DetectCommandType will try to detect the type of command from the given data stream. If it cannot be detected,
// it returns `nil`.
func (cl Commands) DetectCommandType(data string) *Command {
	for _, v := range cl {
		if v.IsOfType(data) {
			return &v
		}
	}
	return nil
}
