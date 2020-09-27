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

func VerbosityName() string {
	switch log.GetLevel() {
	case log.PanicLevel:
		return "PANIC"
	case log.FatalLevel:
		return "FATAL"
	case log.ErrorLevel:
		return "ERROR"
	case log.WarnLevel:
		return "WARN"
	case log.InfoLevel:
		return "INFO"
	case log.DebugLevel:
		return "DEBUG"
	default:
		return "TRACE"
	}
}
