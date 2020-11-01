package dns

// This is a rewrite of of iodine code into go with some modifications
/*
 * Copyright (c) 2006-2014 Erik Ekman <yarrick@kryo.se>,
 * 2006-2009 Bjorn Andersson <flex@kryo.se>
 *
 * Permission to use, copy, modify, and/or distribute this software for any
 * purpose with or without fee is hereby granted, provided that the above
 * copyright notice and this permission notice appear in all copies.
 *
 * THE SOFTWARE IS PROVIDED "AS IS" AND THE AUTHOR DISCLAIMS ALL WARRANTIES
 * WITH REGARD TO THIS SOFTWARE INCLUDING ALL IMPLIED WARRANTIES OF
 * MERCHANTABILITY AND FITNESS. IN NO EVENT SHALL THE AUTHOR BE LIABLE FOR
 * ANY SPECIAL, DIRECT, INDIRECT, OR CONSEQUENTIAL DAMAGES OR ANY DAMAGES
 * WHATSOEVER RESULTING FROM LOSS OF USE, DATA OR PROFITS, WHETHER IN AN
 * ACTION OF CONTRACT, NEGLIGENCE OR OTHER TORTIOUS ACTION, ARISING OUT OF
 * OR IN CONNECTION WITH THE USE OR PERFORMANCE OF THIS SOFTWARE.
 */

import (
	"bytes"
	"container/list"
	"crypto/rand"
	"encoding/binary"
	"github.com/bokysan/socketace/v2/internal/streams/dns/commands"
	"github.com/bokysan/socketace/v2/internal/streams/dns/util"
	"github.com/bokysan/socketace/v2/internal/util/enc"
	"github.com/miekg/dns"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/xtaci/smux"
	"golang.org/x/net/dns/dnsmessage"
	"io"
	"net"
	"net/http"
	"time"
)

// ClientDnsConnection will simulate connections over a DNS server request/response loop
type ClientDnsConnection struct {
	Communicator    ClientCommunicator
	protocolVersion uint32

	Serializer commands.Serializer

	fragmentSize      uint16
	handshakeComplete bool
	lazymode          commands.LazyMode // -L 1: use lazy mode for low-latency (default). 0: don't (implies -I1)\n"
	selectTimeout     int

	querySize byte

	chunkId []uint16
	userId  byte // The sequential ID of the user (basically "session ID")

	output *list.List // Queue of outgoing packets
	input  *list.List // Queue of incoming packets

	outpkt *Packet // Current outgoing packet
	inpkt  *Packet // Current incoming packet
}

// -I max interval between requests (default 4 sec) to prevent DNS timeouts\n"

// NewClientDnsConnection will create a new packet connection which will wrap a packet connection over DNS
func NewClientDnsConnection(topDomain string, communicator ClientCommunicator) (*ClientDnsConnection, error) {

	return &ClientDnsConnection{
		protocolVersion: ProtocolVersion,
		chunkId:         []uint16{0, 0, 0},
		Communicator:    communicator,
		lazymode:        commands.LazyModeOff,
		Serializer: commands.Serializer{
			Domain: topDomain,
			Upstream: util.UpstreamConfig{
				MtuSize: DefaultUpstreamMtuSize,
			},
			Downstream: util.DownstreamConfig{},
		},
	}, nil
}

// Close will close the underlying stream. If the Close has already been called, it will do nothing
func (dc *ClientDnsConnection) Close() error {
	return dc.Communicator.Close()
}

// Closed will return `true` if SafeStream.Close has been called at least once
func (dc *ClientDnsConnection) Closed() bool {
	return dc.Communicator.Closed()
}

// QueryDns is a low-level function which will take the (already calculated) full hostname and
// execute a DNS lookup query using the given type. It will not do any transcoding / encoding. It is
// expected from the caller to have already done appropriate conversion. If the call succeeds, it returns
// a (low-level) DNS reply, which is exptected to be parsed by the caller.
func (dc *ClientDnsConnection) Query(req commands.Request, timeout time.Duration) (commands.Response, error) {
	reqMsg, err := dc.Serializer.EncodeDnsRequest(req)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	// Push previous chunks down the queue
	dc.chunkId = append([]uint16{dc.chunkId[0] + 7727}, dc.chunkId[0:2]...)
	if dc.chunkId[0] == 0 {
		/* 0 is used as "no-query" in iodined.c */
		dc.chunkId[0] = 7727
	}
	reqMsg.Id = dc.chunkId[0]

	respMsg, _, err := dc.Communicator.SendAndReceive(reqMsg, &timeout)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	resp, err := dc.Serializer.DecodeDnsResponse(respMsg)
	if err != nil {
		return resp, err
	}

	if req.Command().Code != resp.Command().Code {
		return resp, errors.Errorf("Invalid response. Sent request %v, but got %v.", req.Command().Code, resp.Command().Code)
	}

	return resp, nil
}

// QueryDns is a low-level function which will take the (already calculated) full hostname and
// execute a DNS lookup query using the given type. It will not do any transcoding / encoding. It is
// expected from the caller to have already done appropriate conversion. If the call succeeds, it returns
// a (low-level) DNS reply, which is exptected to be parsed by the caller.
func (dc *ClientDnsConnection) QueryDns(hostname string, timeout time.Duration) (*dns.Msg, error) {
	// Push previous chunks down the queue
	dc.chunkId = append([]uint16{dc.chunkId[0] + 7727}, dc.chunkId[0:2]...)
	if dc.chunkId[0] == 0 {
		/* 0 is used as "no-query" in iodined.c */
		dc.chunkId[0] = 7727
	}

	qt := util.QueryTypeCname
	if dc.Serializer.Upstream.QueryType != nil {
		qt = *dc.Serializer.Upstream.QueryType
	}

	msg := &dns.Msg{}
	msg.RecursionDesired = true
	msg.Id = dc.chunkId[0]
	msg.Question = []dns.Question{
		{
			Name:   hostname,
			Qtype:  uint16(qt),
			Qclass: uint16(dnsmessage.ClassINET),
		},
	}

	if dc.Serializer.UseEdns0 {
		msg.SetEdns0(16384, true)
	}

	msg, _, err := dc.Communicator.SendAndReceive(msg, &timeout)

	return msg, err

}

func (dc *ClientDnsConnection) dns_decode(q *dns.Msg) ([]byte, int, error) {
	rr := q.Answer[0]
	switch v := rr.(type) {
	case *dns.NULL:
		return []byte(v.Data), len(v.Data), nil
	case *dns.PrivateRR:
		return []byte(v.Data.String()), v.Data.Len(), nil
	case *dns.CNAME:
		// TODO: Copy this from iodine
		return nil, 0, http.ErrNotSupported
	case *dns.A:
		return nil, 0, http.ErrNotSupported
	case *dns.AAAA:
		return nil, 0, http.ErrNotSupported
	case *dns.MX:
		/* We support 250 records, 250*(255+header) ~= 64kB.
		   Only exact 10-multiples are accepted, and gaps in
		   numbering are not jumped over (->truncated).
		   Hopefully DNS servers won't mess around too much.
		*/
		for i := 0; i < len(q.Answer); i++ {
			return nil, 0, http.ErrNotSupported
		}
	case *dns.SRV:
		/* We support 250 records, 250*(255+header) ~= 64kB.
		   Only exact 10-multiples are accepted, and gaps in
		   numbering are not jumped over (->truncated).
		   Hopefully DNS servers won't mess around too much.
		*/
		return nil, 0, http.ErrNotSupported
	case *dns.TXT:
		return nil, 0, http.ErrNotSupported
	}

	return nil, 0, errors.Errorf("Unknown response time: %v", rr)

}

// ParseDnsResponse will unwrap the specified message and extract the raw bytes from the response.
func (dc *ClientDnsConnection) ParseDnsResponse(expectedCommand commands.Command, q *dns.Msg) ([]byte, int, error) {
	if !q.Response {
		err := errors.Errorf("Expected response but got request")
		log.WithError(err).Warnf("Could not parse response: %v", err)
		return nil, 0, err
	}

	if q.Id != dc.chunkId[0] {
		err := errors.Errorf("Expected response for chunk %v but got response for chunk %v", dc.chunkId, q.Id)
		log.WithError(err).Warnf("Could not parse response: %v", err)
		return nil, 0, err
	}

	if q.Answer == nil || len(q.Answer) == 0 {
		err := errors.Errorf("Missing answer section")
		log.WithError(err).Warnf("Could not parse response: %v", err)
		return nil, 0, err
	}

	ans := q.Answer[0]
	name := ans.Header().Name

	if !expectedCommand.IsOfType(name) {
		err := errors.Errorf("Invalid command. Expected %v, got %v", expectedCommand, name[0])
		log.WithError(err).Warnf("Could not parse response: %v", err)
		return nil, 0, err
	}

	data, read, err := dc.dns_decode(q)

	if err != nil {
		return data, read, err
	}

	/* if still here: reply matches our latest query */

	/* Non-recursive DNS servers (such as [a-m].root-servers.net)
	   return no answer, but only additional and authority records.
	   Can't explicitly test for that here, just assume that
	   NOERROR is such situation. Only trigger on the very first
	   requests (Y or V, depending if -T given).
	*/

	if q.Rcode == dns.RcodeSuccess && !expectedCommand.ExpectsEmptyReply() {
		err := errors.Errorf(
			"Got empty reply. "+
				"This nameserver may not be resolving recursively, use another. "+
				"Try \"iodine [options] ns.%s %s\" first, it might just work.",
			dc.Serializer.Domain, dc.Serializer.Domain)
		log.WithError(err).Warnf("Could not parse response: %v", err)
		return nil, 0, err
	}

	/* If we get an immediate SERVFAIL on the Handshake query
	   we're waiting for, wait a while before sending the next.
	   SERVFAIL reliably happens during fragsize autoprobe, but
	   mostly long after we've moved along to some other queries.
	   However, some DNS relays, once they throw a SERVFAIL, will
	   for several secs apply it immediately to _any_ new query
	   for the same topdomain. When this happens, waiting a while
	   is the only option that works.
	*/
	if q.Rcode == dns.RcodeServerFailure {
		<-time.After(time.Second)
	}

	return data, read, err

}

func (dc *ClientDnsConnection) SendEncodingTestUpstream(pattern string, timeout time.Duration) (*dns.Msg, error) {
	/* NOTE: String may be at most 63-4=59 chars to fit in 1 dns chunk. */

	seed := make([]byte, 5)

	var randSeed uint32
	if err := binary.Read(rand.Reader, binary.LittleEndian, &randSeed); err != nil {
		return nil, err
	}

	seed[1] = enc.IntToBase32Char(int(randSeed>>10) & 0x1f)
	seed[2] = enc.IntToBase32Char(int(randSeed>>5) & 0x1f)
	seed[3] = enc.IntToBase32Char(int(randSeed) & 0x1f)

	data := commands.CmdTestUpstreamEncoder.String() +
		string(seed) +
		pattern +
		"." +
		dc.Serializer.Domain

	return dc.QueryDns(data, timeout)
}

func (dc *ClientDnsConnection) SendQueryTypeTest(timeout time.Duration) error {
	var s = downloadCodecCheck
	slen := len(downloadCodecCheck)
	var trycodec enc.Encoder

	if *dc.Serializer.Upstream.QueryType == util.QueryTypeNull || *dc.Serializer.Upstream.QueryType == util.QueryTypePrivate {
		trycodec = enc.RawEncoding
	} else {
		trycodec = enc.Base32Encoding
	}

	/* We could use 'Z' bouncing here, but 'Y' also tests that 0-255
	   byte values can be returned, which is needed for NULL/PRIVATE
	   to work. */

	var resp *dns.Msg
	if r, err := dc.SendEncodingTestDownstream(trycodec, timeout); err != nil {
		return err
	} else {
		resp = r
	}

	in, read, err := dc.ParseDnsResponse(commands.CmdTestDownstreamEncoder, resp)
	if err != nil {
		return err
	}

	if read != slen {
		return errors.Errorf("Got %v bytes but expected %v", slen, read)
	}

	for k := 0; k < slen; k++ {
		if in[k] != s[k] {
			/* corrupted */
			return errors.Errorf("Got back corrupted stream!")
		}
	}

	/* if still here, then all okay */
	return nil
}

// BuildHostname will encode the given data using the supplied encoder a into a dns-valid hostname. E.g. if your
// domain is `d.example.com` you will get something like `lsdfkowefowfe.jvjs7unhjnklax....d.example.com`. The method
// will make sure that:
// - the hostname length does not exceed 250 bytes ([RFC 1035](https://www.ietf.org/rfc/rfc1035.txt))
// - each specific segment does not exeed 60 bytes (safe value; RFC staates it can be up to 63 bytes though)
//
// It will not do any cache-invalidation -- it's up to the caller to ensure proper uniqueness of the hostname
func (dc *ClientDnsConnection) BuildHostname(cmd commands.Command, data []byte, encoder enc.Encoder) (string, error) {
	// cmd
	// data
	// .topdomain

	buf := ""
	buf += cmd.String()
	buf += encoder.Encode(data)

	buf = util.Dotify(buf)

	if buf[len(buf)-1] != '.' {
		buf += "."
	}
	buf += dc.Serializer.Domain

	if len(buf) > util.HostnameMaxLen-2 {
		return "", util.ErrTooLong
	}

	return buf, nil
}

// SendAndReceive will take the (already encoded) data, slap a random at the end (to prevent query caching),
// add the top domain and call QueryDns.
func (dc *ClientDnsConnection) SendAndReceive(cmd commands.Command, data string, timeout time.Duration) (*dns.Msg, error) {
	/* Add lower 15 bits of rand seed as base32, followed by a dot and the tunnel domain and send */
	seed := make([]byte, 3)
	if err := binary.Read(rand.Reader, binary.LittleEndian, &seed); err != nil {
		return nil, err
	}

	seed[0] = enc.ByteToBase32Char(seed[0])
	seed[1] = enc.ByteToBase32Char(seed[1])
	seed[2] = enc.ByteToBase32Char(seed[2])

	hostname := cmd.String()
	if cmd.RequiresUser() {
		hostname += string(enc.ByteToBase32Char(dc.userId))
	}
	hostname += string(seed)
	if data != "" {
		hostname += data
		if len(hostname) > util.LabelMaxlen {
			hostname = util.Dotify(hostname)
		}
	}
	hostname += "." + dc.Serializer.Domain
	if len(hostname) > util.HostnameMaxLen-2 {
		return nil, util.ErrTooLong
	}

	return dc.QueryDns(hostname, timeout)
}

func (dc *ClientDnsConnection) SendChunk(timeout time.Duration) (*dns.Msg, error) {
	//datacmc := 0
	//datacmcchars := "abcdefghijklmnopqrstuvwxyz0123456789"
	//
	//p = outpkt.data
	//p += outpkt.offset
	//avail = outpkt.len - outpkt.offset
	//
	///* Note: must be same, or smaller than SendFragmentSizeTest() */
	//outpkt.sentlen = build_hostname(buf+5, sizeof(buf)-5, p, avail,
	//	topdomain, dataenc, HostnameMaxlen)
	//
	///* Build upstream data header (see doc/proto_xxxxxxxx.txt) */
	//
	//buf[0] = dc.userid_char /* First byte is hex userid */
	//
	//code = ((outpkt.seqno & 7) << 2) | ((outpkt.fragment & 15) >> 2)
	//buf[1] = b32_5to8(code) /* Second byte is 3 bits seqno, 2 upper bits fragment count */
	//
	//code = ((outpkt.fragment & 3) << 3) | (inpkt.seqno & 7)
	//buf[2] = b32_5to8(code) /* Third byte is 2 bits lower fragment count, 3 bits downstream packet seqno */
	//
	//code = ((inpkt.fragment & 15) << 1) | (outpkt.sentlen == avail)
	//buf[3] = b32_5to8(code) /* Fourth byte is 4 bits downstream fragment count, 1 bit last frag flag */
	//
	//buf[4] = datacmcchars[datacmc] /* Fifth byte is data-CMC */
	//datacmc++
	//if datacmc >= 36 {
	//	datacmc = 0
	//}
	//
	//return dc.QueryDns(buf)
	return nil, nil
}

func (dc *ClientDnsConnection) SendPing(timeout time.Duration) (*dns.Msg, error) {
	data := []byte{
		byte((dc.inpkt.SeqNo&7)<<4) | (dc.inpkt.Fragment & 15),
	}

	return dc.SendAndReceive(commands.CmdPing, string(enc.Base32Encoding.Encode(data)), timeout)
}

func (dc *ClientDnsConnection) VersionHandshake() (serverVersion uint32, err error) {
	for i := 0; !dc.Closed() && i < 5; i++ {
		var resp commands.Response
		resp, err = dc.Query(&commands.VersionRequest{
			ClientVersion: dc.protocolVersion,
		}, time.Second*time.Duration(i))
		if err == nil {
			response := resp.(*commands.VersionResponse)
			dc.userId = response.UserId

			log.Debugf("Version ok, both using protocol v 0x%08x. You are user #%d", ProtocolVersion, dc.userId)
			return response.ServerVersion, nil
		}
		log.WithError(err).Infof("Retrying version check: %v", err)
	}
	if err != nil {
		err = errors.Wrapf(err, "couldn't connect to server (maybe other -T options will work)")
	} else {
		err = errors.New("couldn't connect to server (maybe other -T options will work)")
	}
	return
}

// SendEncodingTestDownstream will send a specific downstream encoder to the server and expect a
// pre-determined response. We know that the encoder works properly because we will match the response
// to what we have on file. If the strings match -- encoder works.
func (dc *ClientDnsConnection) SendEncodingTestDownstream(downenc enc.Encoder, timeout time.Duration) (*dns.Msg, error) {
	return dc.SendAndReceive(commands.CmdTestDownstreamEncoder, string(downenc.Code()), timeout)
}

func (dc *ClientDnsConnection) AutoDetectQueryType() error {
	highestWorking := 100

	log.Debugf("Autodetecting DNS query type")

	/*
	   Method: try all "interesting" qtypes with a 1-sec timeout, then try
	   all "still-interesting" qtypes with a 2-sec timeout, etc.
	   "Interesting" means: qtypes that (are expected to) have higher
	   bandwidth than what we know is working already (highest working).
	   Note that DNS relays may not immediately resolve the first (NULL)
	   query in 1 sec, due to long recursive lookups, so we keep trying
	   to see if things will start working after a while.
	*/

	for timeout := 1; !dc.Closed() && timeout <= 3; timeout++ {
		for qtNumber := 0; !dc.Closed() && qtNumber < highestWorking; qtNumber++ {
			if qtNumber >= len(util.QueryTypesByPriority) {
				break /* this round finished */
			}
			queryType := util.QueryTypesByPriority[qtNumber]

			log.Tracef("Testing for %s...", queryType)

			if err := dc.SendQueryTypeTest(secs(timeout)); err == nil {
				/* okay */
				highestWorking = qtNumber
				break
				/* try others with longer timeout */
			} else {
				/* else: try next qtype with same timeout */
				log.Tracef("Testing for %s failed", queryType)
			}
		}
		if highestWorking == 0 {
			/* good, we have NULL; abort immediately */
			break
		}
	}

	if dc.Closed() {
		err := errors.Wrapf(io.ErrClosedPipe, "Stopped while autodetecting DNS query type.")
		log.WithError(err).Warnf("%v", err)
		return err /* problem */
	}

	/* finished */
	if highestWorking >= len(util.QueryTypesByPriority) {

		/* also catches highestworking still 100 */
		err := errors.Errorf("No suitable DNS query type found. Are you connected to a network?")
		return err /* problem */
	}

	/* "using qtype" message printed in Handshake function */
	dc.Serializer.Upstream.QueryType = &util.QueryTypesByPriority[highestWorking]

	return nil /* okay */
}

func (dc *ClientDnsConnection) AutodetectEdns0Extension() {
	var trycodec enc.Encoder

	if *dc.Serializer.Upstream.QueryType == util.QueryTypeNull {
		trycodec = enc.RawEncoding
	} else {
		trycodec = enc.Base32Encoding
	}

	for i := 0; !dc.Closed() && i < 3; i++ {
		var resp *dns.Msg
		if r, err := dc.SendEncodingTestDownstream(trycodec, secs(i+1)); err != nil {
			log.WithError(err).Warnf("Could not send request, will not enable EDNS0: %+v", err)
			dc.Serializer.UseEdns0 = false
			return
		} else {
			resp = r
		}

		in, read, err := dc.ParseDnsResponse(commands.CmdTestDownstreamEncoder, resp)
		if err != nil {
			log.WithError(err).Warnf("Could not parse response, will not enable EDNS0: %+v", err)
			dc.Serializer.UseEdns0 = false
			return
		}

		if read > 0 && read != len(downloadCodecCheck) {
			log.WithError(err).Warnf("reply incorrect = unreliable, will not enable EDNS0: %+v", err)
			dc.Serializer.UseEdns0 = false
			return
		}

		if read > 0 {
			for k := 0; k < len(downloadCodecCheck); k++ {
				if in[k] != downloadCodecCheck[k] {
					log.WithError(err).Warnf("reply cannot be matched, will not enable EDNS0: %+v", err)
					dc.Serializer.UseEdns0 = false
					return
				}
			}
			/* if still here, then all okay */
			log.Debugf("Using EDNS0 extension")
			dc.Serializer.UseEdns0 = true
			return
		}

		log.Debugf("Retrying EDNS0 support test...")
	}

	log.Debugf("Timeout. Will not enable EDNS0 extension.")
	dc.Serializer.UseEdns0 = false
}

// EncodingTestUpstream will test different encodings and see if upstream supports them or not
func (dc *ClientDnsConnection) EncodingTestUpstream(testPattern string) error {
	/* NOTE: *s may be max 59 chars; must start with "aA" for case-swap check
	   Returns:
	   -1: case swap, no need for any further test: error printed; or Ctrl-C
	   0: not identical or error or timeout
	   1: identical string returned
	*/
	slen := len(testPattern)

	for i := 0; !dc.Closed() && i < 3; i++ {
		var resp *dns.Msg
		if r, err := dc.SendEncodingTestUpstream(testPattern, secs(i+1)); err != nil {
			return err
		} else {
			resp = r
		}

		in, read, err := dc.ParseDnsResponse(commands.CmdTestUpstreamEncoder, resp)
		if err != nil {
			return err
		}

		if read > 0 && read < slen+4 {
			return errors.Errorf("reply too short (chars dropped). Expected: %v, Got: %v", slen+4, read)
		}

		if read > 0 {
			/* quick check if case swapped, to give informative error msg */
			if in[4] == 'A' {
				err := util.ErrCaseSwap
				log.Infof("errors.New(\"DNS queries get changed to uppercase, keeping upstream codec Base32: %v", err.Error())
				return err
			}
			if in[5] == 'a' {
				err := util.ErrCaseSwap
				log.Infof("\"DNS queries get changed to lowercase, keeping upstream codec Base32: %v", err.Error())
				return err
			}

			for k := 0; k < slen; k++ {
				if in[k+4] != testPattern[k] {
					/* Definitely not reliable */
					if in[k+4] >= ' ' && in[k+4] <= '~' && testPattern[k] >= ' ' && testPattern[k] <= '~' {
						log.Debugf("DNS query char %q gets changed into %q", testPattern[k], in[k+4])
					} else {
						log.Debugf("DNS query char %q gets changed into %q", testPattern[k], in[k+4])
					}
					return errors.New("DNS changed characters")
				}
			}

			/* if still here, then all okay */
			return nil
		}

		log.Debug("Retrying upstream codec test...")
	}

	if dc.Closed() {
		return io.ErrClosedPipe
	}

	/* timeout */
	return smux.ErrTimeout
}

// AutodetectEncodingUpstream will try to guess the most efficient upstream encoding
// by gradually going from the most efficient to the least efficient encoding
func (dc *ClientDnsConnection) AutodetectEncodingUpstream() {
	/* Note: max 59 chars, must start with "aA".
	   pat64: If 0129 work, assume 3-8 are okay too.

	   RFC1035 par 2.3.1 states that [A-Z0-9-] allowed, but only
	   [A-Z] as first, and [A-Z0-9] as last char _per label_.
	   Test by having '-' as last char.
	*/

	/* Start with Base128, than move on to Base64, starting very gently to not draw attention */
	for _, e := range []enc.Encoder{enc.Base128Encoding, enc.Base64Encoding, enc.Base64uEncoding} {
		ok := true
		for _, pat := range e.TestPatterns() {
			if err := dc.EncodingTestUpstream(pat); err == util.ErrCaseSwap {
				/* DNS swaps case, msg already printed; or Ctrl-C */
				e := enc.Base32Encoding
				log.Tracef("DNS swaps case, falling base to %v", e.Name())
				dc.Serializer.Upstream.Encoder = e
				return
			} else if err != nil {
				/* Probably not okay, skip this encoding entirely */
				ok = false
				break
			}
		}
		if ok {
			log.Tracef("Selected upstream encoding %v", e.Name())
			dc.Serializer.Upstream.Encoder = e
			return
		}
	}

	e := enc.Base32Encoding
	/* if here, then nonthing worked */
	log.Tracef("Selected upstream encoding %v", e.Name())
	dc.Serializer.Upstream.Encoder = e
	return
}

func (dc *ClientDnsConnection) SwitchEncodingUpstream() error {
	data := string(dc.Serializer.Upstream.Encoder.Code())

	log.Infof("Switching upstream to codec to %v", dc.Serializer.Upstream.Encoder.Name())

	for i := 0; dc.Closed() && i < 5; i++ {
		var resp *dns.Msg

		if r, err := dc.SendAndReceive(commands.CmdSetUpstreamEncoder, data, secs(i+1)); err != nil {
			return err
		} else {
			resp = r
		}

		in, read, err := dc.ParseDnsResponse(commands.CmdSetUpstreamEncoder, resp)
		if err != nil {
			return err
		}

		if read > 0 {
			if commands.BadLen.Is(in) {
				e := enc.Base32Encoding
				log.Warnf("Server got bad message length. Falling back to upstream codec: %v", e)
				dc.Serializer.Upstream.Encoder = e
				return nil
			} else if commands.BadIp.Is(in) {
				e := enc.Base32Encoding
				log.Warnf("Server rejected sender IP address. Falling back to upstream codec: %v", e)
				dc.Serializer.Upstream.Encoder = e
				return nil
			} else if commands.BadCodec.Is(in) {
				e := enc.Base32Encoding
				log.Warnf("Server rejected the %v codec. Falling back to upstream codec: %v", dc.Serializer.Upstream.Encoder, e)
				dc.Serializer.Upstream.Encoder = e
				return nil
			}

			log.Debugf("Server switched upstream to codec: %v", dc.Serializer.Upstream.Encoder)
			return nil
		}
	}

	e := enc.Base32Encoding
	log.Debugf("No reply from server on codec switch. Falling back to upstream codec: %v", e)
	dc.Serializer.Upstream.Encoder = e
	return nil
}

func (dc *ClientDnsConnection) TestDownstreamEncoder(trycodec enc.Encoder) error {

	for i := 0; !dc.Closed() && i < 3; i++ {
		var resp *dns.Msg

		if r, err := dc.SendEncodingTestDownstream(trycodec, secs(i+1)); err != nil {
			return nil
		} else {
			resp = r
		}

		in, read, err := dc.ParseDnsResponse(commands.CmdTestDownstreamEncoder, resp)
		if err != nil {
			return err /* hard error */
		}

		if read > 0 && read != len(downloadCodecCheck) {
			return errors.New("reply incorrect = unreliable")
		}

		if read > 0 {
			for k := 0; k < len(downloadCodecCheck); k++ {
				if in[k] != downloadCodecCheck[k] {
					return errors.New("Definitely not reliable")
				}
			}
			/* if still here, then all okay */
			return nil
		}

		log.Debugf("Retrying downstream codec test...")
	}

	/* timeout */
	return smux.ErrTimeout
}

func (dc *ClientDnsConnection) AutodetectEncodingDowntream() {
	/* Returns codec char (or ' ' if no advanced codec works) */

	if *dc.Serializer.Upstream.QueryType == util.QueryTypeNull || *dc.Serializer.Upstream.QueryType == util.QueryTypePrivate {
		/* no other choice than raw */
		log.Debugf("Based on query type, no alternative downstream codec available, using default (Raw)")
		dc.Serializer.Downstream.Encoder = enc.RawEncoding
		return
	}

	log.Debugf("Autodetecting downstream codec (use -O to override)")

	activeEncoder := enc.Base32Encoding
	for _, e := range []enc.Encoder{enc.Base64Encoding, enc.Base64uEncoding, enc.Base85Encoding, enc.Base91Encoding, enc.Base128Encoding} {
		if dc.Closed() {
			return
		}
		if err := dc.TestDownstreamEncoder(e); err != nil {
			log.Infof("Encoding %v does not working properly: %+v", e, err)
			// Try Base64uEncoding before giving up
			if e != enc.Base64Encoding {
				break
			}
		} else {
			activeEncoder = e
		}
	}

	/* If 128 works, then TXT may give us Raw as well */
	if activeEncoder == enc.Base128Encoding && *dc.Serializer.Upstream.QueryType == util.QueryTypeTxt {
		if err := dc.TestDownstreamEncoder(enc.RawEncoding); err != nil {
			log.Infof("Using downstream encoder: %v", enc.RawEncoding)
			dc.Serializer.Downstream.Encoder = enc.RawEncoding
			return
		}
	} else {
		log.Infof("Using downstream encoder: %v", activeEncoder)
		dc.Serializer.Downstream.Encoder = activeEncoder
	}
}

func (dc *ClientDnsConnection) SwitchEncodingDownstream() error {
	data := string(dc.Serializer.Downstream.Encoder.Code()) + string(dc.lazymode)

	log.Debugf("Switching downstream to codec %s", dc.Serializer.Downstream.Encoder.Name())
	for i := 0; !dc.Closed() && i < 5; i++ {
		var resp *dns.Msg

		if r, err := dc.SendAndReceive(commands.CmdSetDownstreamEncoder, data, secs(i+1)); err != nil {
			return err
		} else {
			resp = r
		}

		in, read, err := dc.ParseDnsResponse(commands.CmdSetDownstreamEncoder, resp)
		if err != nil {
			return err
		}

		if read > 0 {
			if commands.BadLen.Is(in) {
				err := errors.New("Server got bad message length. Falling back to downstream codec Base32")
				log.Errorf("%v", err)
				return nil
			} else if commands.BadIp.Is(in) {
				err := errors.New("Server rejected sender IP address. Falling back to downstream codec Base32")
				log.Errorf("%v", err)
				return nil
			} else if commands.BadCodec.Is(in) {
				err := errors.New("Server rejected the selected codec. Falling back to downstream codec Base32")
				log.Errorf("%v", err)
				return nil
			}
			log.Infof("Server switched downstream to codec %s", dc.Serializer.Downstream.Encoder)
			return nil
		}

		log.Debugf("Retrying codec switch...")
	}
	if dc.Closed() {
		return nil
	}

	log.Infof("No reply from server on codec switch. Falling back to downstream codec Base32")

	return nil
}

func (dc *ClientDnsConnection) TryEnableLazyMode(timeout time.Duration) (*dns.Msg, error) {
	data := string(dc.Serializer.Downstream.Encoder.Code()) + string(commands.LazyModeOn)
	return dc.SendAndReceive(commands.CmdSetDownstreamEncoder, data, timeout)
}

// SendFragmentSizeTest will send a request for a "junk" fragment of specified size. This will allow us to check
// if the (response) fragment of that size can pass through the DNS or not
func (dc *ClientDnsConnection) SendFragmentSizeTest(fragsize uint16, timeout time.Duration) (*dns.Msg, error) {
	data := &bytes.Buffer{}
	if err := binary.Write(data, binary.LittleEndian, &fragsize); err != nil {
		return nil, err
	}
	return dc.SendAndReceive(commands.CmdTestFragmentSize, string(enc.Base32Encoding.Encode(data.Bytes())), timeout)
}

func (dc *ClientDnsConnection) CheckFragmentSizeResponse(in []byte, proposed uint16, max uint16) (uint16, bool, error) {
	/* Returns:
	0: keep checking == true
	1: break loop (either okay or definitely wrong) == false
	*/

	if commands.BadIp.Is(in) {
		return 0, false, errors.Errorf("got BADIP (Try iodined -c)..")
	}

	var acknowledged uint16
	if err := binary.Read(bytes.NewBuffer(in), binary.LittleEndian, &acknowledged); err != nil {
		return max, false, err
	}

	if acknowledged != proposed {
		/*
		 * got ack for wrong fragsize, maybe late response for
		 * earlier query, or ack corrupted
		 */
		return max, true, errors.Errorf("Expected %d bytes but server acknowledged %d", proposed, acknowledged)
	}

	in = in[2:]

	if uint16(len(in)) != proposed {
		/*
		 * correctly acked fragsize but read too little (or too
		 * much): this fragsize is definitely not reliable
		 */
		return max, true, errors.Errorf("Expected %d bytes but got back %d. Maybe decoding issue?", proposed, len(in))
	}

	/* Check for corruption */
	v := byte(107)
	for idx, i := range in {
		if i != v {
			if dc.Serializer.Downstream.Encoder == enc.Base32Encoding {
				return 0, false, errors.Errorf("corruption at byte %d using %v encoder this won't work.", idx+2, dc.Serializer.Downstream.Encoder)
			} else {
				return 0, false, errors.Errorf("corruption at byte %d using %v encoder this won't work. Try Base32 downstream encoder.", idx+2, dc.Serializer.Downstream.Encoder)
			}
		}
		v = (v + 107) & 0xff
	}

	return acknowledged, true, nil
}

func (dc *ClientDnsConnection) AutodetectFragmentSize() (uint16, error) {
	var proposed uint16 = 768
	var fragmentRange uint16 = 768
	var max uint16 = 0

	log.Debugf("Autoprobing max downstream fragment size... (skip with -m fragsize)")
	for !dc.Closed() && (fragmentRange >= 8 || max < 300) {
		/* stop the slow probing early when we have enough bytes anyway */
		for i := 0; !dc.Closed() && i < 3; i++ {
			var resp *dns.Msg
			if r, err := dc.SendFragmentSizeTest(proposed, secs(1)); err != nil {
				return 0, err
			} else {
				resp = r
			}

			in, read, err := dc.ParseDnsResponse(commands.CmdTestFragmentSize, resp)
			if err != nil {
				return 0, err
			}

			if read > 0 {
				/* We got a reply */
				if m, ok, err := dc.CheckFragmentSizeResponse(in, proposed, max); ok {
					max = m
					break
				} else if !ok && err != nil {
					return 0, errors.WithStack(err)
				} else {
					max = m
				}
			}
			if max < 0 {
				break
			}

			fragmentRange = fragmentRange >> 1

			if max == proposed {
				/* Try bigger */
				log.Tracef("%d ok, will try %d next.. ", proposed, proposed+fragmentRange)
				proposed += fragmentRange
			} else {
				/* Try smaller */
				log.Tracef("%d not ok, will try %d next.. ", proposed, proposed-fragmentRange)
				proposed -= fragmentRange
			}
		}
	}
	if dc.Closed() {
		err := errors.New("stopped while autodetecting fragment size (Try setting manually with -m)")
		log.Warnf(err.Error())
		return 0, err
	}
	if max <= 2 {
		/* Tried all the way down to 2 and found no good size.
		   But we _did_ do all Handshake before this, so there must
		   be some workable connection. */
		err := errors.New("Found no accepted fragment size. Try setting -M to 200 or lower, or try other -T or -O options.")
		log.Warnf(err.Error())
		return 0, err
	}

	/* data header adds 2 bytes */
	log.Infof("will use %d-2=%d\n", max, max-2)

	/* need 1200 / 16frags = 75 bytes fragsize */
	if max < 82 {
		err := errors.New("Note: this probably won't work well. Try setting -M to 200 or lower, or try other DNS types (-T option).")
		log.Warnf(err.Error())
		return 0, err
	} else if max < 202 &&
		(*dc.Serializer.Upstream.QueryType == util.QueryTypeNull || *dc.Serializer.Upstream.QueryType == util.QueryTypePrivate ||
			*dc.Serializer.Upstream.QueryType == util.QueryTypeTxt || *dc.Serializer.Upstream.QueryType == util.QueryTypeSrv ||
			*dc.Serializer.Upstream.QueryType == util.QueryTypeMx) {
		log.Warn("Note: this isn't very much. Try setting -M to 200 or lower, or try other DNS types (-T option).")
	}

	return max - 2, nil
}

func (dc *ClientDnsConnection) AutodetectLazyMode() {

	log.Debugf("Switching to lazy mode for low-latency")
	for i := 0; !dc.Closed() && i < 5; i++ {
		var resp *dns.Msg

		if r, err := dc.TryEnableLazyMode(secs(i + 1)); err != nil {
			log.WithError(err).Errorf("Could set lazy mode: %v", err)
			dc.lazymode = commands.LazyModeOff
			dc.selectTimeout = 1
			return
		} else {
			resp = r
		}

		in, read, err := dc.ParseDnsResponse(commands.CmdSetDownstreamEncoder, resp)
		if err != nil {
			log.WithError(err).Errorf("Could not parse response: %v", err)
			dc.lazymode = commands.LazyModeOff
			dc.selectTimeout = 1
			return
		}

		if read > 0 {
			if commands.BadLen.Is(in) {
				log.Errorf("Server got bad message length. Falling back to legacy mode.")
				dc.lazymode = commands.LazyModeOff
				dc.selectTimeout = 1
				return
			} else if commands.BadIp.Is(in) {
				log.Errorf("Server rejected sender IP address. Falling back to legacy mode.")
				dc.lazymode = commands.LazyModeOff
				dc.selectTimeout = 1
				return
			} else if commands.BadCodec.Is(in) {
				log.Errorf("Server rejected lazy mode. Falling back to legacy mode.")
				dc.lazymode = commands.LazyModeOff
				dc.selectTimeout = 1
				return
			} else if commands.LazyModeOk.Is(in) {
				log.Debugf("Server switched to lazy mode.")
				dc.lazymode = commands.LazyModeOn
				return
			}
		}

		log.Debugf("Retrying lazy mode switch...")
	}
	if dc.Closed() {
		return
	}

	log.Debugf("No reply from server on lazy switch.")
	log.Infof("Falling back to legacy mode.")
	dc.lazymode = commands.LazyModeOff
	dc.selectTimeout = 1

}

func (dc *ClientDnsConnection) SendSetDownstreamFragmentSize(fragsize uint16, timeout time.Duration) (*dns.Msg, error) {
	data := &bytes.Buffer{}
	data.Write([]byte{dc.userId})
	if err := binary.Write(data, binary.LittleEndian, &fragsize); err != nil {
		return nil, err
	}
	return dc.SendAndReceive(commands.CmdSetDownstreamFragmentSize, string(enc.Base32Encoding.Encode(data.Bytes())), timeout)
}

func (dc *ClientDnsConnection) SwitchFragmentSize(requested uint16) error {
	log.Infof("Setting downstream fragment size to max %d...", requested)

	for i := 0; !dc.Closed() && i < 5; i++ {
		var resp *dns.Msg
		if r, err := dc.SendSetDownstreamFragmentSize(requested, secs(i+1)); err != nil {
			return err
		} else {
			resp = r
		}

		in, read, err := dc.ParseDnsResponse(commands.CmdSetDownstreamFragmentSize, resp)
		if err != nil {
			return err
		}

		if read > 0 {
			if commands.BadFrag.Is(in) {
				log.Warnf("Server rejected fragsize. Keeping default.")
				return nil
			} else if commands.BadIp.Is(in) {
				log.Warnf("Server rejected fragsize (BADIP). Keeping default.")
				return nil
			}

			var accepted uint16
			data := bytes.NewBuffer(in)
			if err := binary.Read(data, binary.LittleEndian, &accepted); err != nil {
				return err
			}

			if accepted != requested {
				log.Warnf("Server responded with the different frag size (%d) than requested (%d). Keeping default.", accepted, requested)
				return nil
			}

			log.Debugf("Upgrading fragment size to %d...", accepted)
			dc.fragmentSize = accepted
			return nil
		}

		log.Debugf("Retrying set fragsize...")
	}
	if dc.Closed() {
		return nil
	}

	log.Debugf("No reply from server when setting fragsize. Keeping default.")
	return nil
}

func (dc *ClientDnsConnection) Handshake() error {
	dc.Serializer.UseEdns0 = false

	/* qtype message printed in Handshake function */
	if dc.Serializer.Upstream.QueryType == nil {
		err := dc.AutoDetectQueryType()
		if err != nil {
			return err
		}
	}

	log.Infof("Using DNS type %s queries", dc.Serializer.Upstream.QueryType)

	if _, err := dc.VersionHandshake(); err != nil {
		return err
	}

	// if err := client.handshake_login(); err != nil {
	//	return err
	// }

	dc.AutodetectEdns0Extension()
	if dc.Closed() {
		return errors.Wrapf(io.ErrClosedPipe, "Stream closed, stopping Handshake.")
	}

	dc.AutodetectEncodingUpstream()
	if dc.Closed() {
		return errors.Wrapf(io.ErrClosedPipe, "Stream closed, stopping Handshake.")
	}

	if err := dc.SwitchEncodingUpstream(); err != nil {
		return err
	}
	if dc.Closed() {
		return errors.Wrapf(io.ErrClosedPipe, "Stream closed, stopping Handshake.")
	}

	dc.AutodetectEncodingDowntream()
	if dc.Closed() {
		return errors.Wrapf(io.ErrClosedPipe, "Stream closed, stopping Handshake.")
	}

	if err := dc.SwitchEncodingDownstream(); err != nil {
		return err
	}
	if dc.Closed() {
		return errors.Wrapf(io.ErrClosedPipe, "Stream closed, stopping Handshake.")
	}

	dc.AutodetectLazyMode()
	if dc.Closed() {
		return errors.Wrapf(io.ErrClosedPipe, "Stream closed, stopping Handshake.")
	}

	if f, err := dc.AutodetectFragmentSize(); err != nil {
		return err
	} else if err := dc.SwitchFragmentSize(f); err != nil {
		return err
	}

	return nil
}

func (dc *ClientDnsConnection) Read(b []byte) (n int, err error) {
	panic("implement me")
}

func (dc *ClientDnsConnection) Write(b []byte) (n int, err error) {
	panic("implement me")
}

func (dc *ClientDnsConnection) LocalAddr() net.Addr {
	return dc.Communicator.LocalAddr()
}

func (dc *ClientDnsConnection) RemoteAddr() net.Addr {
	return dc.Communicator.RemoteAddr()
}

func (dc *ClientDnsConnection) SetDeadline(t time.Time) error {
	return dc.Communicator.SetDeadline(t)
}

func (dc *ClientDnsConnection) SetReadDeadline(t time.Time) error {
	return dc.Communicator.SetReadDeadline(t)
}

func (dc *ClientDnsConnection) SetWriteDeadline(t time.Time) error {
	return dc.Communicator.SetWriteDeadline(t)
}
