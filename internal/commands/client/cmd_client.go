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
	Upstream   upstream.ServerList `json:"upstream" short:"u" long:"upstream"  env:"UPSTREAM" required:"true" description:"Upstream server address(es). Will be tried in other specified on the command line e.g. 'tcp://example.org:1234', 'https://172.10.1.11/ws/all', 'tcp+tls://10.1.2.3:2222', 'stdin:'"`
}

func NewCommand() *Command {
	return &Command{}
}

func (s *Command) CertManager() cert.TlsConfig {
	return &s.ClientConfig
}

//noinspection GoUnusedParameter
func (s *Command) Execute(args []string) error {
	logging.SetupLogging()

	var errs error

	if err := s.ListenList.StartListening(&s.Upstream, s); err != nil {
		return errors.Wrapf(err, "Could not listen on some of the addresses: %s", errs)
	}

	var e error
	interrupted := make(chan os.Signal, 1)
	signal.Notify(interrupted, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	select {
	case <-interrupted:
		log.Infof("Graceful shutdown...")
		s.Upstream.Shutdown()
		for _, srv := range s.ListenList {
			srvType := reflect.TypeOf(reflect.Indirect(reflect.ValueOf(srv)).Interface())
			log.Debugf("Shutting down: %v", srv.String())
			if err := srv.Shutdown(); err != nil {
				e = multierror.Append(e, errors.Wrapf(err, "Could not shutdown %v: %v", srvType, srv))
			}
		}
	}

	return e
}
