package server

import (
	"context"
	"crypto/tls"
	"fmt"
	"github.com/bokysan/socketace/v2/internal/cert"
	"github.com/bokysan/socketace/v2/internal/streams"
	"github.com/bokysan/socketace/v2/internal/util"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/gorilla/websocket"
	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"net/http"
	"time"
)

type WebsocketServer struct {
	cert.Manager
	cert.ClientAuthentication

	Kind      string                `json:"kind"`
	Listen    string                `json:"listen"`
	Endpoints WebsocketEndpointList `json:"endpoints"`

	service       *Service
	server        *http.Server
	couldNotStart chan struct{}
}

func NewWebsocketServer() *WebsocketServer {
	return &WebsocketServer{
		Listen:        "127.0.0.1:9988",
		couldNotStart: make(chan struct{}),
	}
}

func (ws *WebsocketServer) String() string {
	if ws.server != nil {
		protocol := "http"
		if ws.server.TLSConfig != nil {
			protocol += "s"
		}

		return fmt.Sprintf("%s://%s", protocol, ws.Listen)
	} else {
		return ""
	}
}

func (ws *WebsocketServer) SetService(service *Service) {
	ws.service = service
}

func (ws *WebsocketServer) EndpointHandler(ep *WebsocketEndpoint, upstreams []*Channel) (http.HandlerFunc, error) {
	var upgrader = websocket.Upgrader{
		EnableCompression: ep.EnableCompression,
	} // use default options

	return func(w http.ResponseWriter, r *http.Request) {
		log.Debugf("New client request...")

		c, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.WithError(err).Errorf("Socket upgrade failed: %+v", err)
			http.Error(w, err.Error(), 500)
			return
		}
		defer func() {
			if err := c.Close(); err != nil {
				log.WithError(err).Warnf("Failed closing the downstream connection: %+v", err)
			}
		}()

		conn := streams.NewWebsocketReadWriteCloser(c)
		client, err := streams.NewProxyWrapperServer(conn)
		if err != nil {
			log.WithError(err).Errorf("Could not negotiate websocket connection: %v", err)
			http.Error(w, err.Error(), 500)
		}

		if err := MultiplexToUpstream(client, upstreams); err != nil {
			log.WithError(err).Errorf("Socket upgrade failed: %+v", err)
			http.Error(w, err.Error(), 500)
			return
		}

	}, nil
}

//noinspection GoUnusedParameter
func (ws *WebsocketServer) Execute(args []string) error {
	var errs error

	address, err := util.ResolveHostAddress(ws.Listen)
	if err != nil {
		return errors.WithStack(err)
	}

	router := chi.NewRouter()
	router.Use(
		middleware.RequestID, // Set Request Id on all requests
		middleware.RealIP,    // Extract actual IP if running behind reverse proxy
		GetRequestLogger(address),
		middleware.RedirectSlashes, // Redirect slashes to no slash URLs
		middleware.Recoverer,       // Recover from panics without crashing the server
		// middleware.Timeout(60*time.Second),
	)

	debugData := make([]string, 0)

	for _, endpoint := range ws.Endpoints {
		upstreams := make(ChannelList, 0)
		if endpoint.Channels == nil || len(endpoint.Channels) == 0 {
			upstreams = ws.service.Channels
		} else {
			for _, ch := range endpoint.Channels {
				upstream, err := ws.service.Channels.Find(ch)
				if err != nil {
					errs = multierror.Append(errs, errors.WithStack(err))
					continue
				}
				upstreams = append(upstreams, upstream)
			}
		}

		if len(upstreams) == 0 {
			errs = multierror.Append(errs, errors.Errorf("No upstreams defined for endpoint %v", endpoint))
			continue
		}
		debugData = append(debugData, fmt.Sprintf("%v -> %v", endpoint.Endpoint, upstreams))

		handler, err := ws.EndpointHandler(&endpoint, upstreams)
		if err != nil {
			errs = multierror.Append(errs, errors.WithStack(err))
			continue
		}
		router.HandleFunc(endpoint.Endpoint, handler)
	}

	if errs != nil {
		return errs
	}

	ws.server = &http.Server{
		Addr:    ws.Listen,
		Handler: router,
	}

	if crt, err := ws.GetX509KeyPair(); err != nil {
		return errors.WithStack(err)
	} else if crt != nil {
		ws.server.TLSConfig = ws.MakeTlsConfig(crt)
		if ws.RequireClientCert {
			ws.server.TLSConfig.ClientAuth = tls.RequireAndVerifyClientCert
		}
		ws.AddCaCertificates(ws.server.TLSConfig)
		log.Infof("Starting websocket server at https://%s: %s", ws.Listen, debugData)
		if err := ws.server.ListenAndServeTLS("", ""); err != http.ErrServerClosed {
			return errors.WithStack(err)
		}
	} else {
		log.Infof("Starting websocket server at http://%s: %s", ws.Listen, debugData)
		if err := ws.server.ListenAndServe(); err != http.ErrServerClosed {
			return errors.WithStack(err)
		}
	}

	return nil
}

func (ws *WebsocketServer) Shutdown() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer func() {
		cancel()
	}()
	return ws.server.Shutdown(ctx)

}
