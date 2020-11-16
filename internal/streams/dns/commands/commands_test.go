package commands

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func Test_EncodeRequestHeaderNoUser(t *testing.T) {

	cmd := Command{
		Code:        'x',
		NeedsUserId: false,
	}

	h := EncodeRequestHeader(cmd, 123)
	require.Len(t, h, 4)

	rem, userId, err := DecodeRequestHeader(cmd, h)
	require.NoError(t, err)
	require.Len(t, rem, 0)
	require.Equal(t, uint16(0), userId)

}

func Test_EncodeRequestHeaderUser(t *testing.T) {

	cmd := Command{
		Code:        'x',
		NeedsUserId: true,
	}

	h := EncodeRequestHeader(cmd, 123)
	require.Len(t, h, 6)

	rem, userId, err := DecodeRequestHeader(cmd, h)
	require.NoError(t, err)
	require.Len(t, rem, 0)
	require.Equal(t, uint16(123), userId)

}

func Test_EncodeRequestHeaderUserWithData(t *testing.T) {

	cmd := Command{
		Code:        'x',
		NeedsUserId: true,
	}

	h := EncodeRequestHeader(cmd, 123)

	data := "0123456789"
	rem, userId, err := DecodeRequestHeader(cmd, h+data)
	require.NoError(t, err)
	require.Equal(t, data, rem)
	require.Equal(t, uint16(123), userId)

}
