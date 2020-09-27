package logging

import (
	"bufio"
	"github.com/bokysan/socketace/v2/internal/args"
	"github.com/bokysan/socketace/v2/internal/util"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"os"
	"strings"
)

func SetupLogging() {
	SetVerbosity(args.General.Verbose)

	if args.General.LogReportCaller {
		log.AddHook(&ContextHook{})
	}

	if args.General.LogFormat == "json" {
		log.SetFormatter(&log.JSONFormatter{
			FieldMap: log.FieldMap{
				log.FieldKeyTime:  "timestamp",
				log.FieldKeyLevel: "@level",
				log.FieldKeyMsg:   "message",
				log.FieldKeyFunc:  "@caller",
			},
		})
	} else {
		color := strings.TrimSpace(strings.ToLower(args.General.LogColor))
		fullTimestamp := args.General.LogFullTimestamp
		log.SetFormatter(&log.TextFormatter{
			ForceColors:   color == "yes" || color == "true" || color == "1",
			DisableColors: color == "no" || color == "false" || color == "0",
			FullTimestamp: fullTimestamp,
		})
	}
	log.SetReportCaller(args.General.LogReportCaller)
	log.Infof("Verbosity level: %v", VerbosityName())

	if args.General.LogFile != nil && len(*args.General.LogFile) > 0 && *args.General.LogFile != "-" {
		f, err := os.OpenFile(*args.General.LogFile, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0666)
		util.MustErrorNilOrExit(errors.WithStack(err))
		writer := bufio.NewWriter(f)
		/*
		writer := nbtee.NewWriter(1024 * 64)
		writer.Add(f)
		writer.Add(os.Stderr)
		writer.Start()
		 */
		log.SetOutput(writer)
	}

}