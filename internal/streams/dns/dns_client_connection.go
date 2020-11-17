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
	"github.com/bokysan/socketace/v2/internal/streams/dns/commands"
	"github.com/bokysan/socketace/v2/internal/streams/dns/util"
	"github.com/bokysan/socketace/v2/internal/util/enc"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/xtaci/smux"
	"golang.org/x/net/dns/dnsmessage"
	"io"
	"math"
	"math/rand"
	"net"
	"os"
	"sync"
	"time"
)

var ErrHandshakeNotCompleted = errors.New("no initialization data available - handshake was most likely not completed")

// ClientDnsConnection will simulate connections over a DNS server request/response loop
type ClientDnsConnection struct {
	Communicator      ClientCommunicator
	Serializer        commands.Serializer
	protocolVersion   uint32
	handshakeComplete bool
	lazymode          bool       // -L 1: use lazy mode for low-latency (default). 0: don't (implies -I1)\n"
	selectTimeout     int        // How often to query the server, in milliseconds
	lastQuery         time.Time  // last time any query was executed against the DNS
	callMutex         sync.Mutex // synchronize calls to DNS
	commMutex         sync.Mutex // synchronize calls to DNS

	chunkId []uint16 // DNS chunk ID
	userId  uint16   // The sequential ID of the userConnection (basically "session ID")

	in  util.InQueue
	out util.OutQueue
}

// -I max interval between requests (default 4 sec) to prevent DNS timeouts\n"

// NewClientDnsConnection will create a new packet connection which will wrap a packet connection over DNS
func NewClientDnsConnection(topDomain string, communicator ClientCommunicator) (*ClientDnsConnection, error) {
	client := &ClientDnsConnection{
		protocolVersion: ProtocolVersion,
		chunkId:         []uint16{0, 0, 0},
		Communicator:    communicator,
		lazymode:        false,
		Serializer: commands.Serializer{
			Domain: topDomain,
			Upstream: util.UpstreamConfig{
				FragmentSize: DefaultUpstreamMtuSize,
			},
			Downstream: util.DownstreamConfig{},
		},
	}
	client.out.OnChunkAdded = client.outChunkAdded

	return client, nil
}

// Close will close the underlying stream. If the Close has already been called, it will do nothing
func (dc *ClientDnsConnection) Close() error {
	if !dc.Closed() {
		// Notify the server to do a clean shutdown, if handshake was complete
		var err error
		// Acknowledge last received chunk
		err = dc.SendAndReceive(nil)
		if err != nil {
			log.WithError(err).Warnf("Failed acknowleding last received packet: %v", err.Error())
		}

		// Shutdown the connection safely
		t := true
		cmd := &commands.SetOptionsRequest{
			UserId: dc.userId,
			Closed: &t,
		}
		_, err = dc.Query(cmd, 5*time.Second)
		if err != nil && err != commands.BadConn {
			log.WithError(err).Warnf("Failed shutting down the server connection: %v", err.Error())
		}
	}

	return dc.Communicator.Close()
}

// Closed will return `true` if SafeStream.Close has been called at least once
func (dc *ClientDnsConnection) Closed() bool {
	return dc.Communicator.Closed()
}

// Query is a low-level function which will take the (already calculated) full hostname and
// execute a DNS lookup query using the given type. It will not do any transcoding / encoding. It is
// expected from the caller to have already done appropriate conversion. If the call succeeds, it returns
// a (low-level) DNS reply, which is exptected to be parsed by the caller.
func (dc *ClientDnsConnection) Query(req commands.Request, timeout time.Duration) (commands.Response, error) {
	return dc.QueryWithData(
		req,
		timeout,
		*dc.Serializer.Upstream.QueryType,
		dc.Serializer.Upstream.Encoder,
		dc.Serializer.Downstream.Encoder,
	)
}

// Query is a low-level function which will take the (already calculated) full hostname and
// execute a DNS lookup query using the given type. It will not do any transcoding / encoding. It is
// expected from the caller to have already done appropriate conversion. If the call succeeds, it returns
// a (low-level) DNS reply, which is exptected to be parsed by the caller.
func (dc *ClientDnsConnection) QueryWithData(req commands.Request, timeout time.Duration, qt dnsmessage.Type, upstream, downstream enc.Encoder) (commands.Response, error) {
	dc.callMutex.Lock()

	reqMsg, err := dc.Serializer.EncodeDnsRequestWithParams(req, qt, upstream)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	// log.Debugf("Sending request: %v", reqMsg.Question[0].String())

	// Push previous chunks down the queue
	dc.chunkId = append([]uint16{dc.chunkId[0] + 7727}, dc.chunkId[0:2]...)
	if dc.chunkId[0] == 0 {
		/* 0 is used as "no-query" in iodined.c */
		dc.chunkId[0] = 7727
	}
	reqMsg.Id = dc.chunkId[0]

	respMsg, _, err := dc.Communicator.SendAndReceive(reqMsg, &timeout)

	dc.callMutex.Unlock()

	if err != nil {
		return nil, errors.WithStack(err)
	}

	resp, err := dc.Serializer.DecodeDnsResponseWithParams(respMsg, downstream)
	if err != nil {
		return resp, errors.WithStack(err)
	}

	if e, ok := resp.(*commands.ErrorResponse); ok {
		if e.Err == commands.BadCommand {
			log.Warnf("Server did not understand our command: %v", reqMsg)
		}

		return resp, e.Err
	}

	if req.Command().Code != resp.Command().Code {
		return resp, errors.Errorf(
			"Invalid response. Sent request %v, but got %v.",
			req.Command().String(),
			resp.Command().String(),
		)
	}

	return resp, nil
}

func (dc *ClientDnsConnection) SendEncodingTestUpstream(pattern []byte, timeout time.Duration) (*commands.TestUpstreamEncoderResponse, error) {
	/* NOTE: String may be at most 63-4=59 chars to fit in 1 dns chunk. */

	req := &commands.TestUpstreamEncoderRequest{
		UserId:  dc.userId,
		Pattern: pattern,
	}

	resp, err := dc.Query(req, timeout)
	if err != nil {
		return nil, err
	} else if r, ok := resp.(*commands.TestUpstreamEncoderResponse); !ok {
		return nil, errors.Errorf("Invalid response type -- expected TestUpstreamEncoderResponse")
	} else {
		return r, nil
	}
}

func (dc *ClientDnsConnection) SendQueryTypeTest(q dnsmessage.Type, timeout time.Duration) error {
	var s = util.DownloadCodecCheck
	slen := len(util.DownloadCodecCheck)
	var trycodec enc.Encoder

	if q == util.QueryTypeNull || q == util.QueryTypePrivate {
		trycodec = enc.RawEncoding
	} else {
		trycodec = enc.Base32Encoding
	}

	/*
	   We could use 'Z' bouncing here, but 'Y' also tests that 0-255
	   byte values can be returned, which is needed for NULL/PRIVATE
	   to work.
	*/
	req := &commands.TestDownstreamEncoderRequest{
		DownstreamEncoder: trycodec,
	}
	resp, err := dc.QueryWithData(req, timeout, q, dc.Serializer.Upstream.Encoder, trycodec)

	if err != nil {
		return errors.WithStack(err)
	}

	response, ok := resp.(*commands.TestDownstreamEncoderResponse)
	if !ok {
		return errors.Errorf("Invalid response -- expected TestDownstreamEncoderResponse")
	}

	if response.Err != nil {
		return response.Err
	}

	data := response.Data
	read := len(data)

	if read != slen {
		return errors.Errorf("Encoder response returned %v bytes but expected %v: %q", read, slen, data)
	}

	for k := 0; k < slen; k++ {
		if data[k] != s[k] {
			/* corrupted */
			return errors.Errorf("Got back corrupted stream at position %v!", k)
		}
	}

	/* if still here, then all okay */
	return nil
}

func (dc *ClientDnsConnection) VersionHandshake() (err error) {
	for i := 0; !dc.Closed() && i < 5; i++ {
		var resp commands.Response
		resp, err = dc.Query(&commands.VersionRequest{
			ClientVersion: dc.protocolVersion,
		}, time.Second*time.Duration(i))
		if err == nil {
			response := resp.(*commands.VersionResponse)
			dc.userId = response.UserId

			log.Debugf("Version ok, both using protocol v 0x%08x. You are user #%d", ProtocolVersion, dc.userId)
			return nil
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
func (dc *ClientDnsConnection) SendEncodingTestDownstream(downenc enc.Encoder, timeout time.Duration) (*commands.TestDownstreamEncoderResponse, error) {
	resp, err := dc.QueryWithData(
		&commands.TestDownstreamEncoderRequest{
			DownstreamEncoder: downenc,
		},
		timeout,
		*dc.Serializer.Upstream.QueryType,
		dc.Serializer.Upstream.Encoder,
		downenc,
	)
	if err != nil {
		return nil, errors.WithStack(err)
	} else if r, ok := resp.(*commands.TestDownstreamEncoderResponse); ok {
		return r, nil
	} else {
		return nil, errors.Errorf("Invalid return type -- not TestDownstreamEncoderResponse")
	}
}

func (dc *ClientDnsConnection) AutoDetectQueryType() error {
	var highestWorking dnsmessage.Type = 0
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
		for _, q := range util.QueryTypesByPriority {
			log.Tracef("Testing for %v...", q)
			if err := dc.SendQueryTypeTest(q, secs(timeout)); err == nil {
				if highestWorking == 0 || util.QueryTypesByPriority.Before(q, highestWorking) {
					log.Infof("Query type %v works.", q)
					/* okay */
					highestWorking = q
					break
				} else {
					log.Debugf("Query type %v works, but more optional (%v) already found.", q, highestWorking)
				}
				/* try others with longer timeout */
			} else {
				/* else: try next qtype with same timeout */
				log.WithError(err).Warnf("Query %s failed: %v", q, err)
			}
		}
		if highestWorking == util.QueryTypeNull {
			/* good, we have NULL; abort immediately */
			break
		}
	}

	if dc.Closed() {
		err := errors.Wrapf(os.ErrClosed, "Stopped while autodetecting DNS query type.")
		log.WithError(err).Warnf("%v", err)
		return err /* problem */
	}

	/* finished */
	if highestWorking == 0 {

		/* also catches highestworking still 100 */
		err := errors.Errorf("No suitable DNS query type found. Are you connected to a network?")
		return err /* problem */
	}

	/* "using qtype" message printed in Handshake function */
	dc.Serializer.Upstream.QueryType = &highestWorking

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
		resp, err := dc.SendEncodingTestDownstream(trycodec, secs(i+1))
		if err != nil {
			log.WithError(err).Warnf("Could not communicate with the server, will retry EDNS0: %+v", err)
			continue
		}

		if len(resp.Data) != len(util.DownloadCodecCheck) {
			log.WithError(err).Warnf("reply incorrect = unreliable, will not enable EDNS0: %+v", err)
			dc.Serializer.UseEdns0 = false
			return
		}

		for k := 0; k < len(util.DownloadCodecCheck); k++ {
			if resp.Data[k] != util.DownloadCodecCheck[k] {
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

	log.Debugf("Timeout. Will not enable EDNS0 extension.")
	dc.Serializer.UseEdns0 = false
}

// EncodingTestUpstream will test different encodings and see if upstream supports them or not
func (dc *ClientDnsConnection) EncodingTestUpstream(testPattern []byte) error {
	/* NOTE: *s may be max 59 chars; must start with "aA" for case-swap check
	   Returns:
	   -1: case swap, no need for any further test: error printed; or Ctrl-C
	   0: not identical or error or timeout
	   1: identical string returned
	*/

	if string(testPattern[0:2]) != "aA" {
		testPattern = append([]byte("aA"), testPattern...) // Prefix pattern with case sensitivity test
	}
	slen := len(testPattern)

	for i := 0; !dc.Closed() && i < 3; i++ {
		var resp *commands.TestUpstreamEncoderResponse
		if r, err := dc.SendEncodingTestUpstream(testPattern, secs(i+1)); err == smux.ErrTimeout {
			log.Debug("Retrying upstream codec test...")
			continue
		} else if err != nil {
			return err
		} else {
			resp = r
		}

		if l1, l2 := len(resp.Data), len(testPattern); l1 != l2 {
			return errors.Errorf("reply of invalid len, exp=%d, got=%v. Expected: %q, Got: %q", l2, l1, testPattern, resp.Data)
		}

		/* quick check if case swapped, to give informative error msg */
		if resp.Data[0] != 'a' {
			err := util.ErrCaseSwap
			log.Infof("DNS queries get changed to uppercase, keeping upstream codec Base32: %v", err.Error())
			return err
		}
		if resp.Data[1] != 'A' {
			err := util.ErrCaseSwap
			log.Infof("DNS queries get changed to lowercase, keeping upstream codec Base32: %v", err.Error())
			return err
		}

		for k := 0; k < slen; k++ {
			if resp.Data[k] != testPattern[k] {
				/* Definitely not reliable */
				return errors.New("DNS changed characters")
			}
		}

		/* if still here, then all okay */
		return nil
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
	for _, e := range []enc.Encoder{enc.Base128Encoding, enc.Base91Encoding, enc.Base85Encoding, enc.Base64Encoding, enc.Base64uEncoding} {
		ok := true
		for _, pat := range e.TestPatterns() {
			if err := dc.EncodingTestUpstream(pat); err == util.ErrCaseSwap {
				/* DNS swaps case, msg already printed; or Ctrl-C */
				e := enc.Base32Encoding
				log.Warnf("DNS swaps case, falling base to %v", e.Name())
				dc.Serializer.Upstream.Encoder = e
				return
			} else if err != nil {
				/* Probably not okay, skip this encoding entirely */
				log.Warnf("Encoding %v not OK: %v", e.Name(), err)
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

func (dc *ClientDnsConnection) SendSetEncodingUpstream(timeout time.Duration) (*commands.SetOptionsResponse, error) {
	cmd := &commands.SetOptionsRequest{
		UserId:          dc.userId,
		UpstreamEncoder: dc.Serializer.Upstream.Encoder,
	}
	resp, err := dc.Query(cmd, timeout)
	if err != nil {
		return nil, err
	} else if r, ok := resp.(*commands.SetOptionsResponse); !ok {
		return nil, errors.Errorf("Invalid response -- expected SetUpstreamEncoderResponse: %v", resp)
	} else {
		return r, nil
	}
}

func (dc *ClientDnsConnection) SetEncodingUpstream() error {
	log.Infof("Switching upstream to codec to %v", dc.Serializer.Upstream.Encoder.Name())
	for i := 0; !dc.Closed() && i < 5; i++ {
		resp, err := dc.SendSetEncodingUpstream(secs(i + 1))
		if err == smux.ErrTimeout {
			log.Debugf("No response, retrying...")
			continue
		} else if err != nil {
			e := enc.Base32Encoding
			log.Warnf("Communication error, reverting to upstream encoder %v: %v", e, err)
			dc.Serializer.Upstream.Encoder = e
			return nil
		} else if resp.Err != nil {
			e := enc.Base32Encoding
			log.WithError(resp.Err).Warnf("Server error, reverting to upstream encoder %v: %v", e, resp.Err)
			dc.Serializer.Upstream.Encoder = e
			return nil
		} else {
			log.Debugf("Upstream coded switched to %v", dc.Serializer.Upstream.Encoder)
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
		resp, err := dc.SendEncodingTestDownstream(trycodec, secs(i+1))
		if err != nil {
			log.WithError(err).Warnf("Could not communicate with the server, will retry EDNS0: %+v", err)
			continue
		}

		if l1, l2 := len(resp.Data), len(util.DownloadCodecCheck); l1 != l2 {
			return errors.Errorf("reply incorrect, got %v bytes but expected %v", l1, l2)
		}

		for k := 0; k < len(util.DownloadCodecCheck); k++ {
			if resp.Data[k] != util.DownloadCodecCheck[k] {
				return errors.Wrapf(err, "reply cannot be matched, unreiable: %+v", err)
			}
		}

		/* if still here, then all okay */
		log.Debugf("Codec %+v OK", trycodec)
		return nil
	}

	/* timeout */
	return smux.ErrTimeout
}

func (dc *ClientDnsConnection) AutodetectEncodingDowntream() {
	/* Returns codec char (or ' ' if no advanced codec works) */

	if *dc.Serializer.Upstream.QueryType == util.QueryTypeNull || *dc.Serializer.Upstream.QueryType == util.QueryTypePrivate {
		/* no other choice than raw */
		log.Debugf("QueryType is NULL or PRIVATE, using the most optimal (raw) downstream encoding.")
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
			log.Infof("Encoding %v does not working properly: %v", e, err.Error())
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

func (dc *ClientDnsConnection) SendSetEncodingDownstream(timeout time.Duration) (*commands.SetOptionsResponse, error) {
	cmd := &commands.SetOptionsRequest{
		UserId:            dc.userId,
		DownstreamEncoder: dc.Serializer.Downstream.Encoder,
		LazyMode:          &dc.lazymode,
	}
	resp, err := dc.Query(cmd, timeout)
	if err != nil {
		return nil, err
	} else if r, ok := resp.(*commands.SetOptionsResponse); !ok {
		return nil, errors.Errorf("Invalid response -- expected SetUpstreamEncoderResponse: %v", resp)
	} else {
		return r, nil
	}
}

func (dc *ClientDnsConnection) SetEncodingDownstream() error {
	log.Infof("Switching downstream to codec to %v", dc.Serializer.Downstream.Encoder.Name())
	for i := 0; !dc.Closed() && i < 5; i++ {
		resp, err := dc.SendSetEncodingDownstream(secs(i + 1))
		if err == smux.ErrTimeout {
			log.Debugf("No response, retrying...")
			continue
		} else if err != nil {
			e := enc.Base32Encoding
			log.WithError(err).Warnf("Communication error, reverting to downstream encoder %v: %v", e, err)
			dc.Serializer.Downstream.Encoder = e
			return nil
		} else if resp.Err != nil {
			e := enc.Base32Encoding
			log.WithError(resp.Err).Warnf("Server error, reverting to downstream encoder %v: %v", e, resp.Err)
			dc.Serializer.Downstream.Encoder = e
			return nil
		} else {
			log.Debugf("Downstream coded switched to %v", dc.Serializer.Downstream.Encoder)
			return nil
		}
	}

	e := enc.Base32Encoding
	log.Debugf("No reply from server on codec switch. Falling back to downstream codec: %v", e)
	dc.Serializer.Downstream.Encoder = e
	return nil
}

// SendFragmentSizeTest will send a request for a "junk" fragment of specified size. This will allow us to check
// if the (response) fragment of that size can pass through the DNS or not
func (dc *ClientDnsConnection) SendFragmentSizeTest(fragsize uint32, timeout time.Duration) (*commands.TestDownstreamFragmentSizeResponse, error) {
	req := &commands.TestDownstreamFragmentSizeRequest{
		UserId:       dc.userId,
		FragmentSize: fragsize,
	}
	resp, err := dc.Query(req, timeout)
	if err != nil {
		return nil, err
	} else if r, ok := resp.(*commands.TestDownstreamFragmentSizeResponse); !ok {
		return nil, errors.Errorf("Invalid response -- expected TestDownstreamFragmentSizeResponse")
	} else {
		return r, nil
	}
}

func (dc *ClientDnsConnection) CheckFragmentSizeResponse(in []byte) error {
	/* Check for corruption */
	v := byte(107)
	for idx, i := range in {
		if i != v {
			return errors.Errorf("corruption at byte %d using %v encoder this won't work.", idx, dc.Serializer.Downstream.Encoder)
		}
		v = (v + 107) & 0xff
	}

	return nil
}

func (dc *ClientDnsConnection) AutodetectFragmentSize() (uint32, error) {
	var proposed uint32 = 768
	var fragmentRange = 8192 - proposed
	var max uint32 = 0

	log.Debugf("Autoprobing max downstream fragment size... (skip with -m fragsize)")
	for !dc.Closed() && (fragmentRange >= 8 || max < 300) {
		/* stop the slow probing early when we have enough bytes anyway */
		for i := 0; !dc.Closed() && i < 3; i++ {
			resp, err := dc.SendFragmentSizeTest(proposed, secs(1))
			if err == smux.ErrTimeout {
				continue
			} else if err != nil {
				log.WithError(err).Warnf("Communication error: %v", err)
				break
			} else if resp.Err != nil {
				log.WithError(resp.Err).Warnf("Server error: %v", resp.Err)
				break
			}

			if proposed != resp.FragmentSize {
				// Keep max as is
				log.Warnf("Expected %d bytes but server acknowledged %d", proposed, resp.FragmentSize)
				break
			} else if uint32(len(resp.Data)) != resp.FragmentSize {
				log.Warnf("Expected %d bytes but server returned %d", proposed, resp.FragmentSize)
				break
			} else if err := dc.CheckFragmentSizeResponse(resp.Data); err != nil {
				err = errors.WithStack(err)
				if dc.Serializer.Downstream.Encoder == enc.Base32Encoding {
					log.WithError(err).Errorf("Corruption in downstream even with the most basic (Base32) encoder: %v", err)
					return 0, err
				} else {
					log.WithError(err).Errorf("Corruption in downstream even with %v. Try Base32 downstream enodeer: %v", dc.Serializer.Downstream.Encoder, err)
					return 0, err
				}
			} else {
				max = proposed
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
	log.Infof("will use %d-2=%d", max, max-2)

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
	dc.lazymode = true

	for i := 0; !dc.Closed() && i < 5; i++ {
		resp, err := dc.SendSetEncodingDownstream(secs(i + 1))
		if err == smux.ErrTimeout {
			log.Debugf("No response, retrying...")
			continue
		} else if err != nil {
			log.WithError(err).Warnf("Communication error, disabling lazy mode: %v", err)
			dc.lazymode = false
		} else if resp.Err != nil {
			log.WithError(resp.Err).Warnf("Server error, disabling lazy mode: %v", resp.Err)
			dc.lazymode = false
		} else {
			log.Debugf("Lazy mode switched to %v", dc.lazymode)
			return
		}
	}

	log.Debugf("No reply from server on lazy mode switch. Falling back to legacy mode.")
	dc.lazymode = false
	dc.selectTimeout = 1000
}

func (dc *ClientDnsConnection) SendSetDownstreamFragmentSize(fragsize uint32, timeout time.Duration) (*commands.SetOptionsResponse, error) {
	cmd := &commands.SetOptionsRequest{
		UserId:                 dc.userId,
		DownstreamFragmentSize: &fragsize,
	}
	resp, err := dc.Query(cmd, timeout)
	if err != nil {
		return nil, err
	} else if r, ok := resp.(*commands.SetOptionsResponse); !ok {
		return nil, errors.Errorf("Invalid response -- expected SetOptionsResponse: %v", resp)
	} else {
		return r, nil
	}
}

func (dc *ClientDnsConnection) SwitchFragmentSize(requested uint32) error {
	log.Infof("Setting downstream fragment size to max %d...", requested)

	for i := 0; !dc.Closed() && i < 5; i++ {
		resp, err := dc.SendSetDownstreamFragmentSize(requested, secs(i+1))

		if err == smux.ErrTimeout {
			log.Debugf("Retrying set fragsize...")
			continue
		} else if err != nil {
			err = errors.WithStack(err)
			log.WithError(err).Warnf("Communication error. Keeping default fragment size.")
			return err
		} else if resp.Err != nil {
			log.WithError(resp.Err).Warnf("Server error. Keeping default fragment size %v", resp.Err)
			return err
		} else {
			dc.Serializer.Downstream.FragmentSize = requested
			return nil
		}
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

	if err := dc.VersionHandshake(); err != nil {
		return err
	}

	// if err := client.handshake_login(); err != nil {
	//	return err
	// }

	dc.AutodetectEdns0Extension()
	if dc.Closed() {
		return errors.Wrapf(os.ErrClosed, "Stream closed, stopping Handshake.")
	}

	dc.AutodetectEncodingUpstream()
	if dc.Closed() {
		return errors.Wrapf(os.ErrClosed, "Stream closed, stopping Handshake.")
	}

	if err := dc.SetEncodingUpstream(); err != nil {
		return err
	}
	if dc.Closed() {
		return errors.Wrapf(os.ErrClosed, "Stream closed, stopping Handshake.")
	}

	dc.Serializer.Upstream.FragmentSize = dc.getUpstreamMtu()

	dc.AutodetectEncodingDowntream()
	if dc.Closed() {
		return errors.Wrapf(os.ErrClosed, "Stream closed, stopping Handshake.")
	}

	if err := dc.SetEncodingDownstream(); err != nil {
		return err
	}
	if dc.Closed() {
		return errors.Wrapf(os.ErrClosed, "Stream closed, stopping Handshake.")
	}

	dc.AutodetectLazyMode()
	if dc.Closed() {
		return errors.Wrapf(os.ErrClosed, "Stream closed, stopping Handshake.")
	}

	if f, err := dc.AutodetectFragmentSize(); err != nil {
		return err
	} else if err := dc.SwitchFragmentSize(f); err != nil {
		return err
	}

	dc.Serializer.Upstream.FragmentSize = dc.getUpstreamMtu()

	go func() {
		var errCount int
		var lastErr error

		for !dc.Closed() {
			jitter := rand.Intn(300) - 150
			duration := time.Duration(dc.selectTimeout+jitter+(errCount*200)) * time.Millisecond
			if duration < 0 {
				duration = 250
			}
			select {
			case <-time.After(duration):
				if !dc.lastQuery.Add(duration).After(time.Now()) {
					chunk := dc.out.NextChunk()
					if err := dc.SendAndReceive(chunk); err == commands.BadConn {
						// Server closed the connection
						log.Infof("Server closed the connection. Closing on our end.")
						if !dc.Closed() {
							err = errors.WithStack(dc.Close())
							if err != nil {
								log.WithError(err).Warnf("Failed closing the connection: %v", err)
							}
						}
					} else if err != nil {
						if lastErr == err {
							errCount++

							if errCount > 5 {
								log.WithError(err).Warnf("Can't communicate. Giving up: %v", err)
								if !dc.Closed() {
									err = errors.WithStack(dc.Close())
									if err != nil {
										log.WithError(err).Warnf("Failed closing the connection: %v", err)
									}
								}
							}
						} else {
							lastErr = err
							errCount = 0
						}
					} else {
						lastErr = nil
						errCount = 0
					}
				}
			}
		}
	}()

	log.Infof(
		"Handshake complete. "+
			"Connected to %v. "+
			"You are user #%v. "+
			"Upstream{%v (%.2f%% loss), MTU=%d} "+
			"Downstream{%v (%.2f%% loss). MTU=%d}",

		dc.Serializer.Domain,
		dc.userId,
		dc.Serializer.Upstream.Encoder.Name(),
		(1-1/dc.Serializer.Upstream.Encoder.Ratio())*100,
		dc.Serializer.Upstream.FragmentSize,
		dc.Serializer.Downstream.Encoder.Name(),
		(1-1/dc.Serializer.Downstream.Encoder.Ratio())*100,
		dc.Serializer.Downstream.FragmentSize,
	)

	return nil
}

func (dc *ClientDnsConnection) getUpstreamMtu() uint32 {

	// Available space is maximum query length
	space := float64(util.HostnameMaxLen)
	// minus domain length minus dot before and after domain
	space = space - float64(len(dc.Serializer.Domain)) - 2

	// minus command len and cache invalidation len
	space = space - 4

	// And decrease by the space the encoder spends
	space = space / dc.Serializer.Upstream.Encoder.Ratio()

	// minus header that's reserved for Chunk request (with safety margin)
	space = space - 10

	// minus all dots that need to be inserted
	space = space - space/util.LabelMaxlen

	if dc.Serializer.UseMultiQuery {
		space = space - 2
		space = space * 1024
	}

	return uint32(math.Floor(space))
}

// SendAndReceive will send a chunk of data to the server (if available). If not it will "just" send a ping and
// receive any data waiting for the client.
func (dc *ClientDnsConnection) SendAndReceive(chunk *util.Packet) error {
	dc.commMutex.Lock()
	defer dc.commMutex.Unlock()

	req := &commands.PacketRequest{
		UserId:         dc.userId,
		LastAckedSeqNo: dc.in.NextSeqNo - 1,
		Packet:         chunk,
	}

	for i := 1; i <= 5; i++ {
		timeout := time.Duration(i) * time.Second
		if resp, err := dc.Query(req, timeout); err == smux.ErrTimeout {
			if i == 5 {
				return err
			} else {
				continue
			}
		} else if err != nil {
			return err
		} else {
			dc.lastQuery = time.Now()
			if packet, ok := resp.(*commands.PacketResponse); ok {
				if packet.Err != nil {
					return packet.Err
				}
				dc.out.UpdateAcked(packet.LastAckedSeqNo)

				return dc.in.Append(packet.Packet)
			} else {
				return errors.Errorf("Invalid response -- expected Packet")
			}
		}
	}
	return nil
}

// outChunkAdded is called whenever a new chunk is created for the outgoing stream
func (dc *ClientDnsConnection) outChunkAdded() error {
	for chunk := dc.out.NextChunk(); chunk != nil; chunk = dc.out.NextChunk() {
		err := dc.SendAndReceive(chunk)
		if err != nil {
			return err
		}
	}
	return nil
}

func (dc *ClientDnsConnection) Write(b []byte) (n int, err error) {
	if dc.Closed() {
		return 0, os.ErrClosed
	}
	if dc.Serializer.Upstream.QueryType == nil {
		return 0, ErrHandshakeNotCompleted
	}

	return dc.out.Write(b, dc.Serializer.Upstream.FragmentSize)
}

func (dc *ClientDnsConnection) Read(b []byte) (n int, err error) {
	if dc.Closed() && !dc.in.HasData() {
		return 0, io.EOF
	}
	return dc.in.Read(b)
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
