package util

import (
	"github.com/jessevdk/go-flags"
	log "github.com/sirupsen/logrus"
	"os"
)

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
		log.Exit(int(flagsError.Type))
	}

}