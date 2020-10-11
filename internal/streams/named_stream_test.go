package streams

import (
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"os"
	"testing"
)

func Test_NamedStream(t *testing.T) {
	f, err := ioutil.TempFile("", "test")
	require.NoErrorf(t, err, "Could not create temp file: %v", err)
	defer os.Remove(f.Name())

	obj := NewNamedStream(f, f.Name())
	defer obj.Close()

	require.Equal(t, obj.String(), f.Name())
}

func Test_WrappedNamedStream(t *testing.T) {
	f, err := ioutil.TempFile("", "test")
	require.NoErrorf(t, err, "Could not create temp file: %v", err)
	defer os.Remove(f.Name())

	obj1 := NewNamedStream(f, f.Name())
	obj2 := NewSafeStream(obj1)
	obj3 := NewNamedStream(obj2, "demo")
	defer obj3.Close()

	require.Equal(t, obj3.String(), "demo->"+f.Name())
}
