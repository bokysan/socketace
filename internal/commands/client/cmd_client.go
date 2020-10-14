package client

import (
	"github.com/bokysan/socketace/v2/internal/client/listener"
	"github.com/bokysan/socketace/v2/internal/client/upstream"
	"github.com/bokysan/socketace/v2/internal/logging"
	"github.com/bokysan/socketace/v2/internal/util/cert"
	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"os"
	"os/signal"
	"reflect"
	"syscall"
)

type Command struct {
	cert.ClientConfig

	ListenList listener.ListenList `json:"listen"   short:"l" long:"listen"    env:"LISTEN" env-delim:" "     description:"List of addresses to listen on (for specific name). Use multiple times to listen to different services."`
	Upstream   upstream.Upstreams  `json:"upstream" short:"u" long:"upstream"  env:"UPSTREAM" required:"true" description:"Upstream server address(es). Will be tried in other specified on the command line e.g. 'tcp://example.org:1234', 'https://172.10.1.11/ws/all', 'tcp+tls://10.1.2.3:2222', 'stdin:'"`
}

func NewCommand() *Command {
	return &Command{}
}

func (s *Command) CertManager() cert.TlsConfig {
	return &s.ClientConfig
}

func (s *Command) Startup(interrupted <-chan os.Signal) error {
	select {
	case <-interrupted:
		return nil
	default:
		if err := s.ListenList.StartListening(&s.Upstream, s); err != nil {
			return errors.Wrapf(err, "Could not listen on some of the addresses: %s", err)
		}
		return nil
	}
}

func (s *Command) Shutdown() error {
	var errs error

	log.Infof("Graceful client shutdown...")
	s.Upstream.Shutdown()
	for _, srv := range s.ListenList {
		srvType := reflect.TypeOf(reflect.Indirect(reflect.ValueOf(srv)).Interface())
		log.Debugf("Shutting down %v: %v", srvType, srv.String())
		if err := srv.Shutdown(); err != nil {
			errs = multierror.Append(errs, errors.Wrapf(err, "Could not shutdown %v: %v", srvType, srv))
		}
	}

	return errs
}

//noinspection GoUnusedParameter
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
