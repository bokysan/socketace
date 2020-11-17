package enc

// NOTE: This is a translation of base32.c from IODINE project into go.
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
	"encoding/base32"
	"fmt"
	"github.com/pkg/errors"
	"strings"
)

const (
	cb32      = "abcdefghijklmnopqrstuvwxyz012345"
	cb32Ucase = "ABCDEFGHIJKLMNOPQRSTUVWXYZ012345"
)

var iodineBase32Encoding = base32.NewEncoding(cb32).WithPadding(base32.NoPadding)

// IntToBase32Char will covert the given number into a letter from the Base32 alphabet.
// Or to put it in another term It will return the letter from the Base32 alphabet
// at a position given by the argument. If the number is larger than 31, it will
// "wrap over" and work on reminder of the parameter divided by 32.
func IntToBase32Char(in int) byte {
	return cb32[in&31]
}

func ByteToBase32Char(in byte) byte {
	return IntToBase32Char(int(in))
}

func Base32CharToInt(in byte) int {
	pos := strings.IndexByte(cb32, in)
	if pos == -1 {
		pos = strings.IndexByte(cb32Ucase, in)
	}
	return pos
}

// -------------------------------------------------------

// Base32Encoder encodes 5 bytes to 8 characters. Good because it's not case-sensitive.
type Base32Encoder struct {
}

func (b *Base32Encoder) Name() string {
	return "Base32"
}

func (b *Base32Encoder) String() string {
	return fmt.Sprintf("%v(%v)", b.Name(), string(b.Code()))
}

func (b *Base32Encoder) Code() byte {
	return 'T'
}

func (b *Base32Encoder) Encode(data []byte) []byte {
	return []byte(iodineBase32Encoding.EncodeToString(data))
}

func (b *Base32Encoder) Decode(data []byte) ([]byte, error) {
	res, err := iodineBase32Encoding.DecodeString(string(data))
	if err != nil {
		err = errors.WithStack(err)
		return nil, err
	}
	return res, nil
}

func (b *Base32Encoder) TestPatterns() [][]byte {
	return [][]byte{
		[]byte("aA" + cb32),
	}
}

func (b *Base32Encoder) Ratio() float64 {
	return 8.0 / 5.0
}
