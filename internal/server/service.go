package server

import (
	"github.com/bokysan/socketace/v2/internal/logging"
	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"net/http"
	"os"
	"os/signal"
	"reflect"
	"syscall"
)

type Service struct {
	Channels ChannelList `json:"channels"    short:"c" long:"channel"     env:"UPSTREAM"                description:"Add and endpoint. Syntax: '<name>-><protocol>:<address>', e.g. 'ssh->tcp:127.0.0.1:22'"`
	Servers  ServerList  `json:"servers"     short:"s" long:"server"      env:"SERVER" env-delim:" "    description:"UpstreamList of listening server."`
}

func NewService() *Service {
	s := Service{
		Channels: make(ChannelList, 0),
		Servers:  make(ServerList, 0),
	}

	return &s
}

func (host *Service) Execute(args []string) error {
	logging.SetupLogging()

	var e error
	couldNotStart := make(chan struct{}, 100)

	for _, srv := range host.Servers {
		srv.SetService(host)
		go func(srv Server) {
			if err := srv.Execute(args); err != nil && err != http.ErrServerClosed {
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

/*
func (service *Service) Usage() string {
	return ""
}
*/