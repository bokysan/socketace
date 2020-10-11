package server

import (
	"github.com/bokysan/socketace/v2/internal/logging"
	"github.com/bokysan/socketace/v2/internal/server"
	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"net/http"
	"os"
	"os/signal"
	"reflect"
	"syscall"
)

type Command struct {
	Channels server.ChannelList `json:"channels"    short:"L" long:"channel"     env:"UPSTREAM"                description:"Add and endpoint. Syntax: '<name>-><protocol>:<address>', e.g. 'ssh->tcp:127.0.0.1:22'"`
	Servers  server.Servers     `json:"servers"     short:"s" long:"server"      env:"SERVER" env-delim:" "    description:"UpstreamList of listening server."`
}

func NewCommand() *Command {
	s := Command{
		Channels: make(server.ChannelList, 0),
		Servers:  make(server.Servers, 0),
	}

	return &s
}

func (host *Command) Execute(args []string) error {
	logging.SetupLogging()

	var e error
	couldNotStart := make(chan struct{}, 100)

	for _, srv := range host.Servers {
		go func(srv server.Server) {
			if err := srv.Startup(host.Channels); err != nil && err != http.ErrServerClosed {
				couldNotStart <- struct{}{}
				e = multierror.Append(e, err)
			}
		}(srv)
	}

	if e != nil {
		return e
	} else {
		interrupted := make(chan os.Signal, 1)
		signal.Notify(interrupted, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
		select {
		case <-couldNotStart:
			return e
		case <-interrupted:
			log.Infof("Graceful shutdown...")
			for _, srv := range host.Servers {
				srvType := reflect.TypeOf(reflect.Indirect(reflect.ValueOf(srv)).Interface())
				log.Debugf("Shutting down %s: %s", srvType, srv)
				if err := srv.Shutdown(); err != nil {
					e = multierror.Append(e, errors.Wrapf(err, "Could not shutdown %v: %v", srvType, srv))
				}
			}
		}
	}

	return e
}
