package streams

import (
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"os"
	"testing"
)

func Test_SafeWriter_MultipleClose(t *testing.T) {
	f, err := ioutil.TempFile("", "test")
	require.NoErrorf(t, err, "Could not create temp file: %v", err)
	defer os.Remove(f.Name())

	obj := NewSafeWriter(f)
	require.False(t, obj.Closed(), "Stream is closed when it shouldn't be!")

	err = obj.Close()
	require.NoErrorf(t, err, "Could not close file %s: %v", "testdata/file.bin", err)
	require.True(t, obj.Closed(), "Stream is not closed!")

	err = obj.Close()
	require.NoErrorf(t, err, "Error when retrying close on file %s: %v", "testdata/file.bin", err)
}
