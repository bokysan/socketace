package server

import (
	"github.com/bokysan/socketace/v2/internal/args"
	"github.com/bokysan/socketace/v2/internal/logging"
	"github.com/go-chi/chi/middleware"
	"net"
	"net/http"
	"strings"
)

type NextHandlerFunc func(next http.Handler) http.Handler

func GetRequestLogger(address *net.TCPAddr) (logger NextHandlerFunc) {
	if args.General.LogFormat == "json" {
		logger = middleware.RequestLogger( // Write requests to log
			&logging.JSONLogFormatter{
				ServerAddress: address,
			},
		)
	} else {
		color := strings.TrimSpace(strings.ToLower(args.General.LogColor))
		logger = middleware.RequestLogger( // Write requests to log
			&middleware.DefaultLogFormatter{
				Logger:  &logging.ChiLogWriter{},
				NoColor: color == "no" || color == "false" || color == "0",
			},
		)
	}

	return
}
