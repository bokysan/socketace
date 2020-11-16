package commands

import (
	"github.com/bokysan/socketace/v2/internal/streams/dns/util"
	"github.com/bokysan/socketace/v2/internal/util/enc"
	"github.com/miekg/dns"
	"github.com/pkg/errors"
	"golang.org/x/net/dns/dnsmessage"
	"math"
)

type Serializer struct {
	Upstream      util.UpstreamConfig
	Downstream    util.DownstreamConfig
	UseEdns0      bool
	UseMultiQuery bool
	UseLazyMode   bool
	Domain        string
}

// DetectCommandType will try to detect the type of command from the given data stream. If it cannot be detected,
// it returns `nil`.
func (cl Serializer) DetectCommandType(data string) *Command {
	for _, v := range Commands {
		if v.IsOfType(data) {
			return &v
		}
	}
	return nil
}

// EncodeDnsResponse will take a DNS response and create a DNS message
func (cl Serializer) EncodeDnsResponse(resp Response) (*dns.Msg, error) {
	return cl.EncodeDnsResponseWithParams(resp, *cl.Upstream.QueryType, cl.Downstream.Encoder)
}

// EncodeDnsResponse will take a DNS response and create a DNS message
func (cl Serializer) EncodeDnsResponseWithParams(resp Response, qt dnsmessage.Type, downstream enc.Encoder) (*dns.Msg, error) {
	data, err := resp.Encode(downstream)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return util.WrapDnsResponse([]byte(data), qt)
}

// DecodeDnsResponse will take a DNS message and decode it into one of the DNS response object
func (cl Serializer) DecodeDnsResponse(msg *dns.Msg) (Response, error) {
	return cl.DecodeDnsResponseWithParams(msg, cl.Downstream.Encoder)
}

// DecodeDnsResponse will take a DNS message and decode it into one of the DNS response object
func (cl Serializer) DecodeDnsResponseWithParams(msg *dns.Msg, downstream enc.Encoder) (Response, error) {
	data := util.UnwrapDnsResponse(msg)
	for _, c := range Commands {
		if c.IsOfType(data) {
			req := c.NewResponse()
			err := req.Decode(downstream, data)
			return req, err
		}
	}
	return nil, errors.Errorf("Invalid response. Don't know how to handle command type: %v", data[0])
}

// EncodeDnsRequest will take a Request and encode it as a DNS message
func (cl Serializer) EncodeDnsRequest(req Request) (*dns.Msg, error) {
	qt := util.QueryTypeCname
	if cl.Upstream.QueryType != nil {
		qt = *cl.Upstream.QueryType
	}

	return cl.EncodeDnsRequestWithParams(req, qt, cl.Upstream.Encoder)
}

// EncodeDnsRequestWithParams will take a Request and encode it as a DNS message using given (overriden) params
func (cl Serializer) EncodeDnsRequestWithParams(req Request, qt dnsmessage.Type, upstream enc.Encoder) (*dns.Msg, error) {
	data, err := req.Encode(upstream)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	msg := &dns.Msg{}
	msg.RecursionDesired = true

	// MaxLen = maximum length - domain name - dot - order
	maxLen := util.HostnameMaxLen - len(cl.Domain) - 1 - 2
	// make spaces for dots
	maxLen = maxLen - int(math.Ceil(float64(maxLen)/float64(util.LabelMaxlen)))

	if len(data) > maxLen && cl.UseMultiQuery {
		msg.Question = []dns.Question{}

		order := uint16(0)
		for len(data) > 0 {
			if order >= 1024 {
				return nil, errors.Errorf("Message too long!")
			}

			d := ""
			// First two characters represent the byte order
			d += string(enc.IntToBase32Char(int(order)))
			d += string(enc.IntToBase32Char(int(order) >> 4))

			order += 1

			// Limit strings to 255 characters
			if len(data) > maxLen {
				d += data[0:maxLen]
				data = data[maxLen:]
			} else {
				d += data
				data = data[0:0]
			}

			d, err = prepareHostname(d, cl.Domain)
			if err != nil {
				return nil, errors.WithStack(err)
			}

			msg.Question = append(msg.Question, dns.Question{
				Name:   d,
				Qtype:  uint16(qt),
				Qclass: uint16(dnsmessage.ClassINET),
			})
		}
	} else {
		data, err = prepareHostname(data, cl.Domain)
		if err != nil {
			return nil, errors.WithStack(err)
		}

		msg.Question = []dns.Question{
			{
				Name:   data,
				Qtype:  uint16(qt),
				Qclass: uint16(dnsmessage.ClassINET),
			},
		}
	}

	if cl.UseEdns0 {
		msg.SetEdns0(16384, true)
	}

	return msg, nil
}

// DecodeDnsRequest will take a DNS message and decode it into one of the DNS requests objects
func (cl Serializer) DecodeDnsRequest(request string) (Request, error) {
	for _, c := range Commands {
		if c.IsOfType(request) {
			req := c.NewRequest()
			err := req.Decode(cl.Upstream.Encoder, request)
			if err != nil {
				err = errors.Wrapf(err, "Could not decode request using %v upstream encoder: %q", cl.Upstream.Encoder, request)
			}
			return req, err
		}
	}
	return nil, errors.Errorf("Invalid request. Don't know how to handle command type: %v", string(request[0]))
}
