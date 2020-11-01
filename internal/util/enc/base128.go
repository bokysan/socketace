package enc

// NOTE: This is a translation of base64.c from IODINE project into go.
/*
 * Copyright (c) 2006-2014 Erik Ekman <yarrick@kryo.se>,
 * 2006-2009 Bjorn Andersson <flex@kryo.se>
 * Mostly rewritten 2009 J.A.Bezemer@opensourcepartners.nl
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
	"fmt"
	"github.com/pkg/errors"
	"go.chromium.org/luci/common/data/base128"
	"sync"
)

const (

	/*
	 * Don't use '-' (restricted to middle of labels), prefer iso_8859-1
	 * accent chars since they might readily be entered in normal use,
	 * don't use 254-255 because of possible function overloading in DNS systems.
	 */
	cb128 = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789" +
		"\274\275\276\277" +
		"\300\301\302\303\304\305\306\307\310\311\312\313\314\315\316\317" +
		"\320\321\322\323\324\325\326\327\330\331\332\333\334\335\336\337" +
		"\340\341\342\343\344\345\346\347\350\351\352\353\354\355\356\357" +
		"\360\361\362\363\364\365\366\367\370\371\372\373\374\375"
)

var cb128Invert map[byte]byte
var cbInitialized sync.Once

func init() {
	setupCb128Invert()
}

func setupCb128Invert() {
	cbInitialized.Do(func() {
		cb128Invert = make(map[byte]byte)
		for i, v := range []byte(cb128) {
			cb128Invert[v] = byte(i)
		}
	})
}

// -------------------------------------------------------

// Base128Encoder encodes 7 bytes to 8 characters
type Base128Encoder struct {
}

func (b *Base128Encoder) Name() string {
	return "Base128"
}

func (b *Base128Encoder) String() string {
	return fmt.Sprintf("%v(%v)", b.Name(), string(b.Code()))
}

func (b *Base128Encoder) Code() byte {
	return 'V'
}

func (b *Base128Encoder) Encode(src []byte) string {

	dst := make([]byte, 0)

	whichByte := uint(1)
	bufByte := byte(0)

	for _, val := range src {
		// Take the current buffer, add current value, shifted.
		// E.g. first round is first 7 bits of value
		elem := bufByte | (val >> whichByte)
		dst = append(dst, elem)

		// Prepare the remaining data for the next buffer.
		// E.g. first round is the remaining bit
		bufByte = val & ((1 << whichByte) - 1)

		// Shift the remaining value to the left
		bufByte = bufByte << (7 - whichByte)

		if whichByte == 7 {
			dst = append(dst, bufByte)
			bufByte = 0
			whichByte = 0
		}

		whichByte++
	}

	dst = append(dst, bufByte)
	dst = escape128(dst)
	return string(dst)
}

func escape128(src []byte) []byte {
	res := make([]byte, len(src))
	for i, v := range src {
		res[i] = cb128[v]
	}
	return res
}

func unescape128(src []byte) []byte {
	if len(cb128Invert) == 0 {
		setupCb128Invert()
	}
	res := make([]byte, len(src))
	for i, v := range src {
		res[i] = cb128Invert[v]
	}
	return res
}

func (b *Base128Encoder) Decode(data string) ([]byte, error) {
	src := unescape128([]byte(data))
	res, err := base128.DecodeString(string(src))
	if err != nil {
		err = errors.WithStack(err)
		return nil, err
	}
	return res, nil
}

func (b *Base128Encoder) TestPatterns() []string {
	return []string{
		"aA-Aaahhh-Drink-mal-ein-J\344germeister-",
		"aA-La-fl\373te-na\357ve-fran\347aise-est-retir\351-\340-Cr\350te",
		"aAbBcCdDeEfFgGhHiIjJkKlLmMnNoOpPqQrRsStTuUvVwWxXyYzZ",
		"aA0123456789\274\275\276\277\300\301\302\303\304\305\306\307\310\311\312\313\314\315\316\317",
		"aA\320\321\322\323\324\325\326\327\330\331\332\333\334\335\336\337\340\341\342\343\344\345\346\347\350\351\352\353\354\355\356\357\360\361\362\363\364\365\366\367\370\371\372\373\374\375",
	}

}
