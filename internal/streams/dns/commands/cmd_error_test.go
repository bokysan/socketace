package commands

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func Test_ErrorResponse(t *testing.T) {
	r1 := &ErrorResponse{
		Err: BadCodec,
	}
	encoded, err := r1.Encode(nil)
	require.NoError(t, err)

	r2 := &ErrorResponse{}
	err = r2.Decode(nil, encoded)
	require.NoError(t, err)

	require.Equal(t, r1.Err, r2.Err)
}
