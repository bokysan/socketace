package server

import (
	"context"
	"crypto/tls"
	"fmt"
	"github.com/bokysan/socketace/v2/internal/streams"
	"github.com/bokysan/socketace/v2/internal/util/addr"
	"github.com/bokysan/socketace/v2/internal/util/cert"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/gorilla/websocket"
	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"net/http"
	"time"
)

type HttpServer struct {
	cert.ServerConfig

	Address   *addr.ProtoAddress    `json:"address"`
	Endpoints WebsocketEndpointList `json:"endpoints"`

	secure        bool
	server        *http.Server
	couldNotStart chan struct{}
}

func NewHttpServer() *HttpServer {
	return &HttpServer{
		couldNotStart: make(chan struct{}),
	}
}

func (ws *HttpServer) String() string {
	var addr addr.ProtoAddress
	addr = *ws.Address
	return fmt.Sprintf("%s", addr.String())
}

func (ws *HttpServer) EndpointHandler(ep *HttpEndpoint, upstreams Channels) (http.HandlerFunc, error) {
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
		var conn streams.Connection
		conn = streams.NewWebsocketTunnelConnection(c)
		conn = streams.NewNamedConnection(conn, "websocket")

		if err = AcceptConnection(conn, &ws.ServerConfig, ws.secure, upstreams); err != nil {
			log.WithError(err).Errorf("Error accepting connection: %v", err)
		}
	}, nil
}

//noinspection GoUnusedParameter
func (ws *HttpServer) Startup(channels Channels) error {
	var errs error

	address, err := addr.ResolveHostAddress(ws.Address.Host)
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
		upstreams, err := channels.Filter(endpoint.Channels)
		if err != nil {
			errs = multierror.Append(errs, errors.WithStack(err))
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
		Addr:    ws.Address.Host,
		Handler: router,
	}

	if streams.HasTls.MatchString(ws.Address.Scheme) {
		ws.Address.Scheme = ws.Address.Scheme[:len(ws.Address.Scheme)-4]
		ws.secure = true
	} else if ws.Address.Scheme == "https" || ws.Address.Scheme == "wss" {
		ws.Address.Scheme = "https"
		ws.secure = true
	} else {
		ws.Address.Scheme = "http"
		ws.secure = false
	}

	startupErrors := make(chan error, 1)
	go func() {
		if ws.secure {
			var tlsConfig *tls.Config
			if tlsConfig, err = ws.ServerConfig.GetTlsConfig(); err != nil {
				startupErrors <- errors.Wrapf(err, "Could not configure TLS")
				return
			}

			ws.server.TLSConfig = tlsConfig
			log.Infof("Starting HTTPS server at %v", ws)
			if err := ws.server.ListenAndServeTLS("", ""); err != http.ErrServerClosed {
				startupErrors <- errors.WithStack(err)
				return
			}

		} else {
			log.Infof("Starting HTTP server at %v", ws)
			if err := ws.server.ListenAndServe(); err != http.ErrServerClosed {
				startupErrors <- errors.WithStack(err)
				return
			}
		}
	}()

	select {
	case <-time.After(3 * time.Second):
		return nil
	case err := <-startupErrors:
		return err
	}

}

func (ws *HttpServer) Shutdown() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer func() {
		cancel()
	}()
	return ws.server.Shutdown(ctx)

}
