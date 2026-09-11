package statsviz

import (
	"bytes"
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"github.com/arl/statsviz/internal/static"
)

func TestWebSocketContinuesAfterNonFiniteUserSample(t *testing.T) {
	var value atomic.Uint64
	value.Store(math.Float64bits(math.NaN()))
	userPlot, err := (TimeSeriesPlotConfig{
		Name: "finite-recovery",
		Series: []TimeSeries{{
			Name:     "value",
			GetValue: func() float64 { return math.Float64frombits(value.Load()) },
		}},
	}).Build()
	if err != nil {
		t.Fatal(err)
	}
	srv := newServer(t, SendFrequency(5*time.Millisecond), TimeseriesPlot(userPlot))

	httpServer := httptest.NewServer(srv.Ws())
	defer httpServer.Close()
	u, err := url.Parse(httpServer.URL)
	if err != nil {
		t.Fatal(err)
	}
	u.Scheme = "ws"
	ws, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer ws.Close()
	if err := ws.SetReadDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatal(err)
	}

	var cfg struct {
		Event string `json:"event"`
	}
	if err := ws.ReadJSON(&cfg); err != nil {
		t.Fatal(err)
	}
	if cfg.Event != "config" {
		t.Fatalf("first WebSocket event = %q; want config", cfg.Event)
	}

	type metricsEvent struct {
		Event string `json:"event"`
		Data  struct {
			Series struct {
				User []*float64 `json:"finite-recovery"`
			} `json:"series"`
		} `json:"data"`
	}
	var msg metricsEvent
	if err := ws.ReadJSON(&msg); err != nil {
		t.Fatal(err)
	}
	if msg.Event != "metrics" || len(msg.Data.Series.User) != 1 || msg.Data.Series.User[0] != nil {
		t.Fatalf("non-finite metrics event = %#v; want one null point", msg)
	}

	value.Store(math.Float64bits(42))
	for {
		if err := ws.ReadJSON(&msg); err != nil {
			t.Fatal(err)
		}
		if len(msg.Data.Series.User) == 1 && msg.Data.Series.User[0] != nil {
			if *msg.Data.Series.User[0] != 42 {
				t.Fatalf("recovered user value = %v; want 42", *msg.Data.Series.User[0])
			}
			break
		}
	}
}

func testIndex(t *testing.T, f http.Handler, url string) {
	t.Helper()

	req := httptest.NewRequest("GET", url, nil)
	w := httptest.NewRecorder()
	f.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()
	httpindex, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("couldn't read index response: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("http status %v, want %v", resp.StatusCode, http.StatusOK)
	}

	if resp.Header.Get("Content-Type") != "text/html; charset=utf-8" {
		t.Errorf("header[Content-Type] %s, want %s", resp.Header.Get("Content-Type"), "text/html; charset=utf-8")
	}

	fhtml, err := static.Assets().Open("index.html")
	if err != nil {
		t.Fatalf("couldn't read index.html from assets Fs: %v", err)
	}
	defer fhtml.Close()
	fsindex, err := io.ReadAll(fhtml)
	if err != nil {
		t.Fatalf("couldn't read index.html from assets Fs: %v", err)
	}

	if !bytes.Equal(fsindex, httpindex) {
		t.Errorf("read body is not that of index.html from assets")
	}

	if !bytes.Contains(httpindex, []byte("Flower Statsviz")) {
		t.Errorf("read body does not contain Flower branding")
	}

	if bytes.Contains(httpindex, []byte("github.com/arl/statsviz")) {
		t.Errorf("read body contains upstream GitHub link")
	}
}

func newServer(tb testing.TB, opts ...Option) *Server {
	tb.Helper()

	srv, err := NewServer(opts...)
	if err != nil {
		tb.Fatal(err)
	}
	tb.Cleanup(func() {
		if err := srv.Close(); err != nil {
			tb.Errorf("Server.Close() error = %v", err)
		}
	})
	return srv
}

func TestIndex(t *testing.T) {
	t.Parallel()

	srv := newServer(t)
	testIndex(t, srv.Index(), "http://example.com/debug/statsviz/")
}

func TestRoot(t *testing.T) {
	t.Parallel()

	testIndex(t, newServer(t, Root("/debug/")).Index(), "http://example.com/debug/")
	testIndex(t, newServer(t, Root("/debug")).Index(), "http://example.com/debug/")
	testIndex(t, newServer(t, Root("/")).Index(), "http://example.com/")
	testIndex(t, newServer(t, Root("/test/")).Index(), "http://example.com/test/")
}

func testWs(t *testing.T, f http.Handler, URL string) {
	t.Helper()

	s := httptest.NewServer(f)
	defer s.Close()

	// Build a "ws://" url using the httptest server URL and the URL argument.
	u1, err := url.Parse(s.URL)
	if err != nil {
		t.Fatal(err)
	}
	u2, err := url.Parse(URL)
	if err != nil {
		t.Fatal(err)
	}

	u1.Scheme = "ws"
	u1.Path = u2.Path

	// Connect to the server
	ws, _, err := websocket.DefaultDialer.Dial(u1.String(), nil)
	if err != nil {
		t.Fatalf("%v", err)
	}
	defer ws.Close()

	// First message is the plots configuration.
	var cfg struct {
		Event string `json:"event"`
	}
	if err := ws.ReadJSON(&cfg); err != nil {
		t.Fatalf("failed reading json from websocket: %v", err)
	}
	if cfg.Event != "config" {
		t.Fatalf("first WebSocket event = %q, want %q", cfg.Event, "config")
	}

	// Check the content of 2 consecutive payloads.
	for range 2 {
		// Verifies that we've received:
		// - 1 time series (cgo), when cgo is enabled
		// - 1 heatmap (sizeClasses).
		var msg struct {
			Event string `json:"event"`
			Data  struct {
				Series struct {
					CGo         []uint64 `json:"cgo"`
					SizeClasses []uint64 `json:"size-classes"`
				} `json:"series"`
			} `json:"data"`
		}

		if err := ws.ReadJSON(&msg); err != nil {
			t.Fatalf("failed reading json from websocket: %v", err)
		}
		if msg.Event != "metrics" {
			t.Errorf("WebSocket event = %q, want %q", msg.Event, "metrics")
		}

		wantCGoLen := 0
		if cgoEnabled {
			wantCGoLen = 1
		}
		if len(msg.Data.Series.CGo) != wantCGoLen {
			t.Errorf("len(cgo) = %d, want %d", len(msg.Data.Series.CGo), wantCGoLen)
		}
		// Heatmaps should have many elements, check that there's more than one.
		if len(msg.Data.Series.SizeClasses) <= 1 {
			t.Errorf("len(sizeClasses) = %d, want > 1", len(msg.Data.Series.SizeClasses))
		}
	}
}

func TestWsCantUpgrade(t *testing.T) {
	url := "http://example.com/debug/statsviz/ws"

	req := httptest.NewRequest("GET", url, nil)
	w := httptest.NewRecorder()
	newServer(t).Ws()(w, req)

	resp := w.Result()
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("responded %v to %q with non-websocket-upgradable conn, want %v", resp.StatusCode, url, http.StatusBadRequest)
	}
}

func testRegister(t *testing.T, f http.Handler, baseURL string) {
	testIndex(t, f, baseURL)
	ws := strings.TrimRight(baseURL, "/") + "/ws"
	testWs(t, f, ws)
}

func TestRegister(t *testing.T) {
	t.Run("defaultmux", func(t *testing.T) {
		previousMux := http.DefaultServeMux
		http.DefaultServeMux = http.NewServeMux()
		t.Cleanup(func() {
			http.DefaultServeMux = previousMux
		})

		if err := Register(http.DefaultServeMux); err != nil {
			t.Fatalf("Register() failed: %v", err)
		}
		testRegister(t, http.DefaultServeMux, "http://example.com/debug/statsviz/")
	})

	t.Run("default", func(t *testing.T) {
		t.Parallel()

		mux := http.NewServeMux()

		newServer(t).Register(mux)
		testRegister(t, mux, "http://example.com/debug/statsviz/")
	})

	t.Run("zero-value", func(t *testing.T) {
		t.Parallel()

		mux := http.NewServeMux()

		var srv Server
		srv.Register(mux)
		t.Cleanup(func() { _ = srv.Close() })
		testRegister(t, mux, "http://example.com/debug/statsviz/")
	})

	t.Run("root", func(t *testing.T) {
		t.Parallel()

		mux := http.NewServeMux()

		srv := newServer(t, Root(""))
		srv.Register(mux)
		testRegister(t, mux, "http://example.com/")
	})

	t.Run("slash", func(t *testing.T) {
		t.Parallel()

		mux := http.NewServeMux()

		srv := newServer(t, Root("/"))
		srv.Register(mux)
		testRegister(t, mux, "http://example.com/")
	})

	t.Run("root2", func(t *testing.T) {
		t.Parallel()

		mux := http.NewServeMux()

		srv := newServer(t, Root("/path/to/statsviz"))
		srv.Register(mux)
		testRegister(t, mux, "http://example.com/path/to/statsviz/")
	})

	t.Run("root+frequency", func(t *testing.T) {
		t.Parallel()

		mux := http.NewServeMux()

		srv := newServer(t,
			Root("/path/to/statsviz"),
			SendFrequency(100*time.Millisecond),
		)
		srv.Register(mux)
		testRegister(t, mux, "http://example.com/path/to/statsviz/")
	})

	t.Run("non-positive frequency", func(t *testing.T) {
		t.Parallel()

		_, err := NewServer(
			Root("/path/to/statsviz"),
			SendFrequency(-1),
		)
		if err == nil {
			t.Errorf("NewServer() should have errored")
		}
	})

	t.Run("WebSocket write timeout", func(t *testing.T) {
		t.Parallel()

		srv := newServer(t, WebSocketWriteTimeout(250*time.Millisecond))
		if srv.webSocketWriteTimeout != 250*time.Millisecond {
			t.Errorf("WebSocket write timeout = %s; want %s", srv.webSocketWriteTimeout, 250*time.Millisecond)
		}
	})

	t.Run("non-positive WebSocket write timeout", func(t *testing.T) {
		t.Parallel()

		if _, err := NewServer(WebSocketWriteTimeout(0)); err == nil {
			t.Error("NewServer() accepted a non-positive WebSocket write timeout")
		}
	})
}

func TestDebugStateConcurrentAccess(t *testing.T) {
	t.Setenv("STATSVIZ_DEBUG", "false")

	enabled := newDebugEnabled()
	upgrader := sync.OnceValue(func() websocket.Upgrader {
		return newWsUpgrader(enabled)
	})

	const readers = 8
	start := make(chan struct{})
	started := make(chan struct{}, readers)
	stop := make(chan struct{})
	var wg sync.WaitGroup
	for range readers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			_ = enabled()
			started <- struct{}{}
			for {
				select {
				case <-stop:
					return
				default:
					_ = enabled()
				}
			}
		}()
	}

	close(start)
	for range readers {
		<-started
	}
	got := upgrader()
	close(stop)
	wg.Wait()
	if enabled() {
		t.Fatal("debug mode enabled for STATSVIZ_DEBUG=false")
	}
	if got.CheckOrigin != nil {
		t.Fatal("debug origin bypass enabled for STATSVIZ_DEBUG=false")
	}
}

func TestZeroValueServerClose(t *testing.T) {
	var srv Server
	if err := srv.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
}

func TestZeroValueServerWebSocket(t *testing.T) {
	var srv Server
	t.Cleanup(func() { _ = srv.Close() })

	httpServer := httptest.NewServer(srv.Ws())
	defer httpServer.Close()
	u, err := url.Parse(httpServer.URL)
	if err != nil {
		t.Fatal(err)
	}
	u.Scheme = "ws"
	dialer := websocket.Dialer{HandshakeTimeout: time.Second}
	ws, _, err := dialer.Dial(u.String(), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer ws.Close()
	if err := ws.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
		t.Fatal(err)
	}

	var cfg struct {
		Event string `json:"event"`
	}
	if err := ws.ReadJSON(&cfg); err != nil {
		t.Fatal(err)
	}
	if cfg.Event != "config" {
		t.Fatalf("first WebSocket event = %q; want config", cfg.Event)
	}
}

func TestNewServerRejectsNilOption(t *testing.T) {
	var option Option
	server, err := NewServer(option)
	if err == nil {
		if server != nil {
			_ = server.Close()
		}
		t.Fatal("NewServer accepted a nil option")
	}
	if server != nil {
		t.Fatalf("NewServer returned server %#v with error %v", server, err)
	}
}
