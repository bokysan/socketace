package enc

import (
	"bytes"
	"fmt"
)

// Base192 could take the top 192 characters (leaving out the bottom 32, which are usually control characters).
// Base192 encoding could achieve near-raw efficiency, as it encodes 7.5 bits / byte. Or, in other words: every
// 15 bytes get encoded into 16 octets. This yields an appropriate 6.66% encoding loss.
const MinAsciiCode = 255 - 192

// Base128Encoder encodes 7.5 bits to 1 octet
type Base192Encoder struct {
}

func (b *Base192Encoder) Name() string {
	return "Base192"
}

func (b *Base192Encoder) String() string {
	return fmt.Sprintf("%v(%v)", b.Name(), string(b.Code()))
}

func (b *Base192Encoder) Code() byte {
	return 'Y'
}

func (b *Base192Encoder) Encode(src []byte) []byte {
	if src == nil {
		return nil
	}

	dst := &bytes.Buffer{}

	whichBit := uint8(1)
	bufNum := uint16(0)

	l := len(src)

	// Take 15 bits at a time and encode them into a 16 bit 192-encoded number
	last := false
	for i := 0; i < l; i++ {
		var val uint16
		if i == l-1 {
			val = uint16(src[i]) << 8
			last = true
		} else {
			val = uint16(src[i])<<8 | uint16(src[i+1])
			last = false
			i++
		}

		// Value is a 16 bit number. We must take 15 bits of it, but use the sliding window.

		// We need to take "first" (whichByte) bits from the number.
		data := val >> whichBit         // Easiest way to do it is to shift bits right by the needed amount...
		rem := val - (data << whichBit) // Get the remaining bits

		elem := bufNum | data // Combine previous data and new data
		dst.WriteByte(byte(elem / 192))
		dst.WriteByte(byte(elem % 192))

		bufNum = rem << (15 - whichBit) // Shift the remaining bits to start

		whichBit++

		if whichBit == 16 {
			whichBit = 1
		}
	}

	whichBit--
	// whichByte will go from 1 to 15. It is basically our iterator loop.

	if whichBit == 0 || (whichBit == 7 && last) {
		// no dangling data to write
	} else if whichBit == 7 || !last {
		// Write the last "7.5" bits to the output
		dst.WriteByte(byte(bufNum / 192))
	}

	if dst.Len() == 0 {
		return []byte{}
	}

	return dst.Bytes()
}

func (b *Base192Encoder) Decode(src []byte) ([]byte, error) {
	if src == nil {
		return nil, nil
	}

	dst := &bytes.Buffer{}

	l := len(src)

	whichBit := uint(1)
	bufNum := uint16(0)

	// The cycle repeats every 15 bytes:
	//
	// 1 byte  gets encoded to  2 bytes --> whichByte = 2
	// 2 bytes gets encoded to  3 bytes --> whichByte = 3
	// 3 bytes gets encoded to  4 bytes --> whichByte = 4
	// 4 bytes gets encoded to  5 bytes --> whichByte = 5
	// 5 bytes gets encoded to  6 bytes --> whichByte = 6
	// 6 bytes gets encoded to  7 bytes --> whichByte = 7
	// 7 bytes gets encoded to  8 bytes --> whichByte = 8
	// 8 bytes gets encoded to  9 bytes --> whichByte = 9
	// 9 bytes gets encoded to 10 bytes --> whichByte = 10
	//10 bytes gets encoded to 11 bytes --> whichByte = 11
	//11 bytes gets encoded to 12 bytes --> whichByte = 12
	//12 bytes gets encoded to 13 bytes --> whichByte = 13
	//13 bytes gets encoded to 14 bytes --> whichByte = 14
	//14 bytes gets encoded to 15 bytes --> whichByte = 15
	//15 bytes gets encoded to 16 bytes --> whichByte = 1

	// Take 16 bits at a time but decode 15 bits at a time

	for i := 0; i < l; i++ {
		var val uint16
		if i == l-1 {
			val = uint16(src[i]) << 8
		} else {
			val = uint16(src[i])<<8 | uint16(src[i+1])
			i++
		}

		decoded := ((val >> 8) * 192) | (val & 0xFF) // Decode the number into 15 bits

		if whichBit != 1 {
			top := decoded >> (16 - whichBit)  // Get the remaining bits
			bufNum = bufNum | top              // ..and add them to our buffer
			dst.WriteByte(byte(bufNum >> 8))   // Put the hight bits in the destination
			dst.WriteByte(byte(bufNum & 0xFF)) // And follow up by low bits
		}
		bufNum = decoded << whichBit // push the bits over

		whichBit++

		if whichBit == 15 {
			whichBit = 0
		}
	}

	if dst.Len() == 0 {
		return []byte{}, nil
	}

	return dst.Bytes(), nil
}

func (b *Base192Encoder) TestPatterns() [][]byte {
	str := make([]byte, 192)
	for k := range str {
		str[k] = byte(k + MinAsciiCode)
	}

	return [][]byte{
		str,
	}
}

func (b *Base192Encoder) Ratio() float64 {
	return 8.0 / 7.5
}
