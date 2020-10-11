package util

import (
	"bou.ke/monkey"
	"errors"
	"github.com/jessevdk/go-flags"
	"github.com/stretchr/testify/require"
	"os"
	"sync"
	"testing"
)

// seqMutex makes sure that we are executing the code sequentially, as we are monkey-patching the code in-memory.
// This is not thread safe or safe in any kind of way
var seqMutex sync.Mutex

// lock will lock the mutex and return a function to unlock the mutex. This a

func Test_MustErrorNilOrExit_NilError(t *testing.T) {
	seqMutex.Lock()
	defer seqMutex.Unlock()

	var exited bool
	fakeExit := func(int) {
		exited = true
	}
	patch := monkey.Patch(os.Exit, fakeExit)
	defer patch.Unpatch()

	MustErrorNilOrExit(nil)

	require.False(t, exited, "MustErrorNilOrExit existed the program and it shouldn't have done so.")
}

func Test_MustErrorNilOrExit_FlagsError(t *testing.T) {
	seqMutex.Lock()
	defer seqMutex.Unlock()

	var exitCode int
	fakeExit := func(i int) {
		exitCode = i
	}
	patch := monkey.Patch(os.Exit, fakeExit)
	defer patch.Unpatch()

	err := &flags.Error{
		Type:    flags.ErrShortNameTooLong,
		Message: "Short name too long",
	}

	MustErrorNilOrExit(err)

	require.Equal(t, int(flags.ErrShortNameTooLong), exitCode, "MustErrorNilOrExit did not return a proper exit code")
}

func Test_MustErrorNilOrExit_GenericError(t *testing.T) {
	seqMutex.Lock()
	defer seqMutex.Unlock()

	var exitCode int
	fakeExit := func(i int) {
		exitCode = i
	}
	patch := monkey.Patch(os.Exit, fakeExit)
	defer patch.Unpatch()

	err := errors.New("demo")

	MustErrorNilOrExit(err)

	require.Equal(t, int(ErrGeneric), exitCode, "MustErrorNilOrExit did not return a proper exit code")
}
