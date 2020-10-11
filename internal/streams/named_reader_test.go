package streams

import (
	"github.com/stretchr/testify/require"
	"os"
	"testing"
)

func Test_NamedReader(t *testing.T) {
	f, err := os.OpenFile("testdata/file.bin", os.O_RDONLY, os.ModePerm)
	require.NoErrorf(t, err, "Could not open file %s: %v", "testdata/file.bin", err)

	obj := NewNamedReader(f, f.Name())
	defer obj.Close()

	require.Equal(t, obj.String(), f.Name())
}

func Test_WrappedNamedReader(t *testing.T) {
	f, err := os.OpenFile("testdata/file.bin", os.O_RDONLY, os.ModePerm)
	require.NoErrorf(t, err, "Could not open file %s: %v", "testdata/file.bin", err)

	obj1 := NewNamedReader(f, f.Name())
	obj2 := NewSafeReader(obj1)
	obj3 := NewNamedReader(obj2, "demo")
	defer obj3.Close()

	require.Equal(t, obj3.String(), "demo->"+f.Name())
}
