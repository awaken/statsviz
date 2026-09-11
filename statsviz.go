// Package statsviz allows visualizing Go [runtime/metrics] data in real time.
//
// Register Statsviz HTTP handlers with your server's [http.ServeMux]
// (preferred method):
//
//	mux := http.NewServeMux()
//	statsviz.Register(mux)
//
// Alternatively, you can register with [http.DefaultServeMux]
// though you shouldn't do that in production:
//
//	var ss statsviz.Server
//	ss.Register(http.DefaultServeMux)
//
// By default, Statsviz is served at http://host:port/debug/statsviz/. This, and
// other settings, can be changed by passing some [Option] to [NewServer].
//
// If your application is not already running an HTTP server, you need to start
// one. Add "net/http" and "log" to your imports, and use the following code in
// your main function:
//
//	go func() {
//	    log.Println(http.ListenAndServe("localhost:8080", nil))
//	}()
//
// Then open your browser and visit http://localhost:8080/debug/statsviz/.
//
// # Advanced usage:
//
// If you want more control over Statsviz HTTP handlers, for example if:
//   - you're using some HTTP framework
//   - you want to place Statsviz handler behind some middleware
//
// then use [NewServer] to obtain a [Server] instance. Both the [Server.Index] and
// [Server.Ws]() methods return [http.HandlerFunc].
//
//	srv, err := statsviz.NewServer(); // Create server or handle error
//	srv.Index()                       // UI (dashboard) http.HandlerFunc
//	srv.Ws()                          // Websocket http.HandlerFunc
package statsviz

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"github.com/arl/statsviz/internal/plot"
	"github.com/arl/statsviz/internal/static"
)

const (
	defaultRoot                  = "/debug/statsviz"
	defaultSendInterval          = time.Second
	defaultWebSocketWriteTimeout = 10 * time.Second
)

// RegisterDefault registers the Statsviz HTTP handlers on [http.DefaultServeMux].
//
// RegisterDefault should not be used in production.
func RegisterDefault(opts ...Option) error {
	return Register(http.DefaultServeMux, opts...)
}

// Register registers the Statsviz HTTP handlers on the provided mux.
//
// Register must be called once per application.
func Register(mux *http.ServeMux, opts ...Option) error {
	srv, err := NewServer(opts...)
	if err != nil {
		return err
	}
	srv.Register(mux)
	return nil
}

// Server is the core component of Statsviz. It collects and periodically
// updates metrics data and provides two essential HTTP handlers:
//   - the Index handler serves Statsviz user interface, allowing you to
//     visualize runtime metrics on your browser.
//   - The Ws handler establishes a WebSocket connection allowing the connected
//     browser to receive metrics updates from the server.
//
// The zero value is a valid Server, with default options.
//
// NOTE: Having more than one Server in the same program is not supported (and
// is not useful anyway).
type Server struct {
	cancel  context.CancelFunc // terminate goroutines
	clients *clients           // connected websocket clients

	interval              time.Duration // interval between consecutive metrics emission
	root                  string        // HTTP path root
	webSocketWriteTimeout time.Duration // maximum duration of a WebSocket frame write
	plots                 *plot.List    // plots shown on the user interface
	userPlots             []plot.UserPlot
}

// NewServer constructs a new Statsviz Server with the provided options, or the
// default settings.
//
// Note that once the server is created, its HTTP handlers need to be registered
// with some HTTP server. You can either use the Register method or register yourself
// the Index and Ws handlers.
func NewServer(opts ...Option) (*Server, error) {
	var s Server
	if err := s.init(opts...); err != nil {
		return nil, err
	}
	return &s, nil
}

func (s *Server) init(opts ...Option) error {
	*s = Server{
		interval:              defaultSendInterval,
		root:                  defaultRoot,
		webSocketWriteTimeout: defaultWebSocketWriteTimeout,
	}

	for i, opt := range opts {
		if opt == nil {
			return fmt.Errorf("option %d is nil", i)
		}
		if err := opt(s); err != nil {
			return err
		}
	}

	pl, err := plot.NewList(s.userPlots)
	if err != nil {
		return err
	}
	s.plots = pl

	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel
	s.clients = newClients(ctx, s.plots.Config(), s.webSocketWriteTimeout)

	// Collect metrics.
	go func() {
		tick := time.NewTicker(s.interval)
		defer tick.Stop()
		defer cancel()

		for {
			select {
			case <-ctx.Done():
				return
			case <-tick.C:
				buf := bytes.Buffer{}
				if _, err := s.plots.WriteTo(&buf); err != nil {
					dbglog("failed to collect metrics: %v", err)
					return
				}
				s.clients.broadcast(buf.Bytes())
			}
		}
	}()

	return nil
}

// Register registers the Statsviz HTTP handlers on the provided mux.
//
// Register must be called once per application.
func (s *Server) Register(mux *http.ServeMux) {
	if s.plots == nil {
		s.init()
	}

	mux.Handle(s.root+"/", s.Index())
	mux.HandleFunc(s.root+"/ws", s.Ws())
}

// Close releases all resources used by the Server.
func (s *Server) Close() error {
	if s.cancel != nil {
		s.cancel()
	}
	return nil
}

// Option is a configuration option for the Server.
type Option func(*Server) error

// SendFrequency changes the interval between successive acquisitions of metrics
// and their sending to the user interface. The default interval is one second.
func SendFrequency(intv time.Duration) Option {
	return func(s *Server) error {
		if intv <= 0 {
			return fmt.Errorf("send interval must be positive")
		}
		s.interval = intv
		return nil
	}
}

// WebSocketWriteTimeout changes the maximum duration allowed for writing one
// WebSocket configuration or metrics frame. The default is ten seconds.
func WebSocketWriteTimeout(timeout time.Duration) Option {
	return func(s *Server) error {
		if timeout <= 0 {
			return fmt.Errorf("WebSocket write timeout must be positive")
		}
		s.webSocketWriteTimeout = timeout
		return nil
	}
}

// Root changes the root path of the Statsviz user interface.
// The default is "/debug/statsviz".
func Root(path string) Option {
	return func(s *Server) error {
		s.root = strings.TrimSuffix(path, "/")
		return nil
	}
}

// TimeseriesPlot adds a new time series plot to Statsviz. This option can
// be added multiple times.
func TimeseriesPlot(tsp TimeSeriesPlot) Option {
	return func(s *Server) error {
		if tsp.timeseries == nil {
			return ErrInvalidTimeSeriesPlot
		}
		s.userPlots = append(s.userPlots, plot.UserPlot{Scatter: tsp.timeseries})
		return nil
	}
}

// Index returns the index handler, which responds with the Statsviz user
// interface HTML page. By default, the handler is served at the path specified
// by the root. Use [Root] to change the path.
func (s *Server) Index() http.HandlerFunc {
	prefix := s.root + "/"
	dist := http.FileServerFS(static.Assets())
	return http.StripPrefix(prefix, dist).ServeHTTP
}

func parseBoolEnv(name string) bool {
	env := os.Getenv(name)
	val, err := strconv.ParseBool(env)
	if err != nil {
		if env != "" {
			fmt.Fprintf(os.Stderr, "statsviz: malformed %s %v\n", name, err)
		}
	}
	return val
}

func newDebugEnabled() func() bool {
	return sync.OnceValue(func() bool {
		return parseBoolEnv("STATSVIZ_DEBUG")
	})
}

var debugEnabled = newDebugEnabled()

func dbglog(format string, args ...any) {
	if debugEnabled() {
		fmt.Fprintf(os.Stderr, "statsviz: "+format+"\n", args...)
	}
}

func newWsUpgrader(debugEnabled func() bool) websocket.Upgrader {
	var checkOrigin func(r *http.Request) bool

	// Allow all origins for testing.
	if debugEnabled() {
		// passthrough
		checkOrigin = func(r *http.Request) bool { return true }
	}

	return websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 2048,
		CheckOrigin:     checkOrigin,
	}
}

var wsUpgrader = sync.OnceValue(func() websocket.Upgrader {
	return newWsUpgrader(debugEnabled)
})

// Ws returns the WebSocket handler used by Statsviz to send application
// metrics. The underlying net.Conn is used to upgrade the HTTP server
// connection to the WebSocket protocol.
func (s *Server) Ws() http.HandlerFunc {
	if s.clients == nil {
		if err := s.init(); err != nil {
			return func(w http.ResponseWriter, _ *http.Request) {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		}
	}

	return func(w http.ResponseWriter, r *http.Request) {
		upgrader := wsUpgrader()
		ws, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			dbglog("failed to upgrade connection: %v", err)
			return
		}

		s.clients.add(ws)
	}
}
