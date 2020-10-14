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

	Network   string                `json:"network"`
	Listen    string                `json:"listen"`
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

	address, err := addr.ResolveHostAddress(ws.Listen)
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
		Addr:    ws.Listen,
		Handler: router,
	}

	if streams.HasTls.MatchString(ws.Network) {
		ws.Network = ws.Network[:len(ws.Network)-4]
		ws.secure = true
	} else if ws.Network == "https" || ws.Network == "wss" {
		ws.Network = "https"
		ws.secure = true
	} else {
		ws.Network = "http"
		ws.secure = false
	}

	if ws.secure {
		var tlsConfig *tls.Config
		if tlsConfig, err = ws.ServerConfig.GetTlsConfig(); err != nil {
			return errors.Wrapf(err, "Could not configure TLS")
		}

		ws.server.TLSConfig = tlsConfig
		log.Infof("Starting HTTPS server at %s", ws)
		if err := ws.server.ListenAndServeTLS("", ""); err != http.ErrServerClosed {
			return errors.WithStack(err)
		}

	} else {
		log.Infof("Starting HTTP server at %s", ws)
		if err := ws.server.ListenAndServe(); err != http.ErrServerClosed {
			return errors.WithStack(err)
		}
	}

	return nil
}

func (ws *HttpServer) Shutdown() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer func() {
		cancel()
	}()
	return ws.server.Shutdown(ctx)

}
