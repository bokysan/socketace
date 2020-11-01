package enc

// NOTE: This is a translation of base64u.c from IODINE project into go.
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
	"github.com/pkg/errors"
)

const (
	cb64u = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ-0123456789_"
)

var iodineBase64uEncoding = base64.NewEncoding(cb64u).WithPadding(base64.NoPadding)

// -------------------------------------------------------

// Base64uEncoder encodes 3 bytes to 4 characters and uses an alternative character map.
type Base64uEncoder struct {
}

func (b *Base64uEncoder) Name() string {
	return "Base64u"
}

func (b *Base64uEncoder) Code() byte {
	return 'U'
}

func (b *Base64uEncoder) Encode(data []byte) string {
	return iodineBase64uEncoding.EncodeToString(data)
}

func (b *Base64uEncoder) Decode(data string) ([]byte, error) {
	res, err := iodineBase64uEncoding.DecodeString(data)
	if err != nil {
		err = errors.WithStack(err)
		return nil, err
	}
	return res, nil
}

func (b *Base64uEncoder) TestPatterns() []string {
	return []string{
		"aAbBcCdDeEfFgGhHiIjJkKlLmMnNoOpPqQrRsStTuUvVwWxXyYzZ_0129-",
	}
}
