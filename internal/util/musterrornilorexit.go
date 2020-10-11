package util

import (
	"github.com/jessevdk/go-flags"
	log "github.com/sirupsen/logrus"
	"os"
)

const (
	ErrGeneric = 99
)

// MustErrorNilOrExit will check the provided argument. If it's `nil` it will simply return. If it's
// not `nil`, it will log the rrror as `log.FatalLevel` and exit immediately with provided error code.
// Error code is unwrapped from `flags.Error` object. If it's a different kind of error, a generic
// error code - 99 - is returned
func MustErrorNilOrExit(err error) {
	if err == nil {
		return
	}

	if flagsError, ok := err.(*flags.Error); ok {
		if flagsError.Type == flags.ErrHelp {
			os.Exit(0)
		}

		log.StandardLogger().WithError(err).Logf(log.FatalLevel, "Error: %+v", err)
		log.Exit(int(flagsError.Type))
	} else {
		log.StandardLogger().WithError(err).Logf(log.FatalLevel, "Error: %+v", err)
		log.Exit(ErrGeneric)
	}

}
