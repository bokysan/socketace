package logging

import (
	log "github.com/sirupsen/logrus"
)

// SetVerbosity defines the verbosity level of the application
func SetVerbosity(v []bool) {

	verbosity := log.Level(len(v))
	if verbosity < 0 {
		verbosity = log.PanicLevel
	} else if verbosity > 6 {
		verbosity = log.TraceLevel
	}
	log.SetLevel(verbosity)
}
