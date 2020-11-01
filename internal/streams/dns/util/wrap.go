package util

import (
	"encoding/binary"
	"github.com/bokysan/socketace/v2/internal/util/enc"
	"github.com/miekg/dns"
	"github.com/pkg/errors"
	"golang.org/x/net/dns/dnsmessage"
	"sort"
	"strings"
)

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

// WrapDnsResponse will wrap the given data into the specificed message type. It will call one of the other
// methods to actually do the wrapping
func WrapDnsResponse(data []byte, queryType dnsmessage.Type) (*dns.Msg, error) {
	switch queryType {
	case QueryTypeNull:
		return WrapDnsResponseNull(data)
	case QueryTypePrivate:
		return WrapDnsResponsePrivate(data)
	case QueryTypeTxt:
		return WrapDnsResponseTxt(data)
	case QueryTypeMx:
		return WrapDnsResponseMx(data)
	case QueryTypeSrv:
		return WrapDnsResponseSrv(data)
	case QueryTypeCname:
		return WrapDnsResponseCname(data)
	case QueryTypeAAAA:
		return WrapDnsResponseAAAA(data)
	case QueryTypeA:
		return WrapDnsResponseA(data)
	}

	return nil, errors.Errorf("Unknown query type: %v", queryType)
}

// WrapDnsResponseA will wrap the data into a A-type DNS response message
func WrapDnsResponseA(data []byte) (*dns.Msg, error) {
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

// WrapDnsResponseAAAA will wrap the data into a AAAA-type DNS response message
func WrapDnsResponseAAAA(data []byte) (*dns.Msg, error) {
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

// WrapDnsResponseCname will wrap the data into a CNAME-type DNS response message
func WrapDnsResponseCname(data []byte) (*dns.Msg, error) {
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

// WrapDnsResponseSrv will wrap the data into a SRV-type DNS response message
func WrapDnsResponseSrv(data []byte) (*dns.Msg, error) {
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

// WrapDnsResponseMx will wrap the data into a MX-type DNS response message
func WrapDnsResponseMx(data []byte) (*dns.Msg, error) {
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

// WrapDnsResponseTxt will wrap the data into a TXT-type DNS response message
func WrapDnsResponseTxt(data []byte) (*dns.Msg, error) {
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

// WrapDnsResponsePrivate will wrap the data into a PRIVATE-type DNS response message
func WrapDnsResponsePrivate(data []byte) (*dns.Msg, error) {
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
				Data: d,
			},
		})
	}
	return msg, nil
}

// WrapDnsResponseNull will wrap the data into a NULL-type DNS response message
func WrapDnsResponseNull(data []byte) (*dns.Msg, error) {
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

// UnwrapDnsResponse will decode the DNS message and return the bytes in the response
func UnwrapDnsResponse(q *dns.Msg) string {
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
