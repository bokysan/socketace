package streams

import (
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"os"
	"testing"
)

func Test_ReadWriterStream(t *testing.T) {
	f1, err := os.OpenFile("testdata/file.bin", os.O_RDONLY, os.ModePerm)
	require.NoErrorf(t, err, "Could not open file %s: %v", "testdata/file.bin", err)

	f2, err := ioutil.TempFile("", "test")
	require.NoErrorf(t, err, "Could not create temp file: %v", err)
	defer os.Remove(f2.Name())

	obj := NewReadWriteCloser(f1, f2)

	require.False(t, obj.Closed(), "Stream is closed when it shouldn't be!")

	err = obj.Close()
	require.NoErrorf(t, err, "Could not close stream: %v", err)
	require.True(t, obj.Closed(), "Stream is not closed!")

	err = obj.Close()
	require.NoErrorf(t, err, "Error when retrying close straem: %v", err)

	require.EqualError(t, f1.Close(), "close testdata/file.bin: file already closed", "File is not closed!")
	require.EqualError(t, f2.Close(), "close "+f2.Name()+": file already closed", "File is not closed!")

}
