package streams

import (
	"github.com/stretchr/testify/require"
	"os"
	"testing"
)

func Test_SafeReader_MultipleClose(t *testing.T) {
	f, err := os.OpenFile("testdata/file.bin", os.O_RDONLY, os.ModePerm)
	require.NoErrorf(t, err, "Could not open file %s: %v", "testdata/file.bin", err)

	obj := NewSafeReader(f)
	require.False(t, obj.Closed(), "Stream is closed when it shouldn't be!")

	err = obj.Close()
	require.NoErrorf(t, err, "Could not close file %s: %v", "testdata/file.bin", err)
	require.True(t, obj.Closed(), "Stream is not closed!")

	err = obj.Close()
	require.NoErrorf(t, err, "Error when retrying close on file %s: %v", "testdata/file.bin", err)

}
