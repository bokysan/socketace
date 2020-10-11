package streams

import (
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"os"
	"testing"
)

func Test_NamedConnection(t *testing.T) {
	f, err := ioutil.TempFile("", "test")
	require.NoErrorf(t, err, "Could not create temp file: %v", err)
	defer os.Remove(f.Name())

	f2 := NewSimulatedConnection(f, Localhost, Localhost)
	obj := NewNamedConnection(f2, f.Name())
	defer obj.Close()

	require.Equal(t, obj.String(), f.Name())
}

func Test_WrappedNamedConnection(t *testing.T) {
	f, err := ioutil.TempFile("", "test")
	require.NoErrorf(t, err, "Could not create temp file: %v", err)
	defer os.Remove(f.Name())

	f2 := NewSimulatedConnection(f, Localhost, Localhost)
	obj1 := NewNamedConnection(f2, f.Name())
	obj2 := NewSafeConnection(obj1)
	obj3 := NewNamedConnection(obj2, "demo")
	defer obj3.Close()

	require.Equal(t, obj3.String(), "demo->"+f.Name())
}
