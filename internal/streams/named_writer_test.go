package streams

import (
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"os"
	"testing"
)

func Test_NamedWriter(t *testing.T) {
	f, err := ioutil.TempFile("", "test")
	require.NoErrorf(t, err, "Could not create temp file: %v", err)
	defer os.Remove(f.Name())

	obj := NewNamedWriter(f, f.Name())
	defer obj.Close()

	require.Equal(t, obj.String(), f.Name())
}

func Test_WrappedNamedWriter(t *testing.T) {
	f, err := ioutil.TempFile("", "test")
	require.NoErrorf(t, err, "Could not create temp file: %v", err)
	defer os.Remove(f.Name())

	obj1 := NewNamedWriter(f, f.Name())
	obj2 := NewSafeWriter(obj1)
	obj3 := NewNamedWriter(obj2, "demo")
	defer obj3.Close()

	require.Equal(t, obj3.String(), "demo->"+f.Name())
}
