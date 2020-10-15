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
	"sync"
	"syscall"
)

type Command struct {
	Channels server.Channels `json:"channels"    short:"L" long:"channel"     env:"UPSTREAM"                description:"Add an endpoint. Syntax: '<name>-><protocol>:<address>', e.g. 'ssh->tcp:127.0.0.1:22'"`
	Servers  server.Servers  `json:"servers"     short:"s" long:"server"      env:"SERVER" env-delim:" "    description:"UpstreamList of listening server."`
}

func NewCommand() *Command {
	s := Command{
		Channels: make(server.Channels, 0),
		Servers:  make(server.Servers, 0),
	}

	return &s
}

func (s *Command) Startup(interrupted <-chan os.Signal) error {
	var errs error
	m := &sync.Mutex{}
	wg := &sync.WaitGroup{}
	wg.Add(len(s.Servers))

	for _, srv := range s.Servers {
		go func(srv server.Server) {
			select {
			case <-interrupted:
				wg.Done()
			default:
				if err := srv.Startup(s.Channels); err != nil && err != http.ErrServerClosed {
					m.Lock()
					errs = multierror.Append(errs, err)
					m.Unlock()
				}
				wg.Done()
			}
		}(srv)
	}
	wg.Wait()

	return errs
}

func (s *Command) Shutdown() error {
	var errs error

	log.Infof("Graceful server shutdown...")
	for _, srv := range s.Servers {
		srvType := reflect.TypeOf(reflect.Indirect(reflect.ValueOf(srv)).Interface())
		log.Debugf("[Server] Shutting down %v: %v", srvType, srv.String())
		if err := srv.Shutdown(); err != nil {
			errs = multierror.Append(errs, errors.Wrapf(err, "Could not shutdown %v: %v", srvType, srv))
		}
	}

	return errs

}

func (s *Command) Execute(args []string) error {
	logging.SetupLogging()

	interrupted := make(chan os.Signal, 1)
	signal.Notify(interrupted, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	if err := s.Startup(interrupted); err != nil {
		return err
	}

	select {
	case <-interrupted:
		return s.Shutdown()
	}
}
