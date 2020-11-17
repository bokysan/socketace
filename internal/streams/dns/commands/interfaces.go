package commands

import "github.com/bokysan/socketace/v2/internal/util/enc"

// Request represents a (serialized) request to a DNS server
type Request interface {
	// Command is the command that this request reffers to
	Command() Command
	// Encode will encode this requires into a DNS-compatible query, potentially using the encoder specified. Note that
	// encoding allways happens in the "hostname/domain" format -- e.g. so you can execute a A, CNAME, or a MX query
	// whith this.
	Encode(e enc.Encoder) ([]byte, error)
	// Decode will decode the data from the query into this object
	Decode(e enc.Encoder, request []byte) error
}

// Response it the response from the DNS server
type Response interface {
	// Command is the command that this request reffers to
	Command() Command
	// EncodeResponse will encode this response into a data stream which can be the sent as a DNS response.
	Encode(e enc.Encoder) ([]byte, error)
	// DecodeResponse will take a byte (data) stream and create a response object
	Decode(e enc.Encoder, response []byte) error
}
