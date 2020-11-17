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
	"encoding/base64"
	"fmt"
	"github.com/pkg/errors"
)

const (
	cb64 = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ-0123456789+"
)

var iodineBase64Encoding = base64.NewEncoding(cb64).WithPadding(base64.NoPadding)

// -------------------------------------------------------

// Base64Encoder encodes 3 bytes to 4 characters
type Base64Encoder struct {
}

func (b *Base64Encoder) Name() string {
	return "Base64"
}

func (b *Base64Encoder) String() string {
	return fmt.Sprintf("%v(%v)", b.Name(), string(b.Code()))
}

func (b *Base64Encoder) Code() byte {
	return 'S'
}

func (b *Base64Encoder) Encode(data []byte) []byte {
	return []byte(iodineBase64Encoding.EncodeToString(data))
}

func (b *Base64Encoder) Decode(data []byte) ([]byte, error) {
	res, err := iodineBase64Encoding.DecodeString(string(data))
	if err != nil {
		err = errors.WithStack(err)
		return nil, err
	}
	return res, nil
}

func (b *Base64Encoder) TestPatterns() [][]byte {
	return [][]byte{
		[]byte("aAbBcCdDeEfFgGhHiIjJkKlLmMnNoOpPqQrRsStTuUvVwWxXyYzZ+0129-"),
	}
}

func (b *Base64Encoder) Ratio() float64 {
	return 4.0 / 3.0
}
