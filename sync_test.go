//go:build go1.25
// +build go1.25

package statsviz

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"testing/synctest"
	"time"

	"github.com/gorilla/websocket"
)

func TestWsConcurrent(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		srv := newServer(t)
		srv.Register(http.NewServeMux())

		li := fakeNetListen()
		s := &httptest.Server{
			Listener: li,
			Config:   &http.Server{Handler: srv.Ws()},
		}
		s.Start()
		defer s.Close()

		// Build a "ws://" url using the httptest server URL.
		u, err := url.Parse(s.URL)
		if err != nil {
			t.Fatal(err)
		}
		u.Scheme = "ws"

		const numConns = 10
		const numMessages = 200

		type connResult struct {
			id  int
			err error
		}
		resultCh := make(chan connResult, numConns)

		synctestWsDialer := websocket.Dialer{
			NetDialContext: func(ctx context.Context, network string, addr string) (net.Conn, error) {
				c := li.connect()
				return c, nil
			},
		}

		for i := range numConns {
			go func(connID int) {
				ws, _, err := synctestWsDialer.Dial(u.String(), nil)
				if err != nil {
					resultCh <- connResult{connID, err}
					return
				}
				defer ws.Close()

				// First message is the plots configuration
				var cfg struct {
					Event string `json:"event"`
				}
				if err := ws.ReadJSON(&cfg); err != nil {
					resultCh <- connResult{connID, err}
					return
				}
				if cfg.Event != "config" {
					resultCh <- connResult{connID, fmt.Errorf("first event = %q, want config", cfg.Event)}
					return
				}

				// Read multiple data messages to ensure state is being accessed
				for range numMessages {
					var msg struct {
						Event string `json:"event"`
					}
					if err := ws.ReadJSON(&msg); err != nil {
						resultCh <- connResult{connID, err}
						return
					}
					if msg.Event != "metrics" {
						resultCh <- connResult{connID, fmt.Errorf("event = %q, want metrics", msg.Event)}
						return
					}
				}

				resultCh <- connResult{id: connID}
			}(i)
		}

		for range numConns {
			result := <-resultCh
			if result.err != nil {
				t.Fatalf("connection %d failed: %v", result.id, result.err)
			}
		}

		if err := srv.Close(); err != nil {
			t.Fatal(err)
		}
		s.Close()

		synctest.Wait()
	})
}

func TestServerCloseClosesWsConnections(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		srv := newServer(t)

		li := fakeNetListen()
		s := &httptest.Server{
			Listener: li,
			Config:   &http.Server{Handler: srv.Ws()},
		}
		s.Start()
		defer s.Close()

		u, err := url.Parse(s.URL)
		if err != nil {
			t.Fatal(err)
		}
		u.Scheme = "ws"

		dialer := websocket.Dialer{
			NetDialContext: func(ctx context.Context, network string, addr string) (net.Conn, error) {
				return li.connect(), nil
			},
		}
		ws, _, err := dialer.Dial(u.String(), nil)
		if err != nil {
			t.Fatal(err)
		}
		defer ws.Close()

		var cfg struct {
			Event string `json:"event"`
		}
		if err := ws.ReadJSON(&cfg); err != nil {
			t.Fatalf("failed reading config: %v", err)
		}
		if cfg.Event != "config" {
			t.Fatalf("first WebSocket event = %q, want %q", cfg.Event, "config")
		}

		conn, ok := ws.UnderlyingConn().(*fakeNetConn)
		if !ok {
			t.Fatalf("underlying connection has type %T, want *fakeNetConn", ws.UnderlyingConn())
		}
		if conn.IsClosedByPeer() {
			t.Fatal("server closed the WebSocket before Server.Close")
		}

		if err := srv.Close(); err != nil {
			t.Fatal(err)
		}
		synctest.Wait()
		if !conn.IsClosedByPeer() {
			t.Fatal("server left the WebSocket open after Server.Close")
		}
	})
}

func TestWebSocketWriteTimeoutRemovesBlockedClient(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		srv := newServer(t,
			SendFrequency(time.Hour),
			WebSocketWriteTimeout(10*time.Second),
		)

		li := fakeNetListen()
		s := &httptest.Server{
			Listener: li,
			Config:   &http.Server{Handler: srv.Ws()},
		}
		s.Start()
		defer s.Close()

		u, err := url.Parse(s.URL)
		if err != nil {
			t.Fatal(err)
		}
		u.Scheme = "ws"

		dialer := websocket.Dialer{
			NetDialContext: func(ctx context.Context, network string, addr string) (net.Conn, error) {
				return li.connect(), nil
			},
		}
		ws, _, err := dialer.Dial(u.String(), nil)
		if err != nil {
			t.Fatal(err)
		}
		defer ws.Close()

		var cfg struct {
			Event string `json:"event"`
		}
		if err := ws.ReadJSON(&cfg); err != nil {
			t.Fatalf("failed reading config: %v", err)
		}
		if cfg.Event != "config" {
			t.Fatalf("first WebSocket event = %q; want %q", cfg.Event, "config")
		}
		synctest.Wait()

		conn, ok := ws.UnderlyingConn().(*fakeNetConn)
		if !ok {
			t.Fatalf("underlying connection has type %T, want *fakeNetConn", ws.UnderlyingConn())
		}
		conn.SetReadBufferSize(0)
		srv.clients.mu.RLock()
		var clientCh chan []byte
		for _, clientCh = range srv.clients.m {
			break
		}
		srv.clients.mu.RUnlock()
		if clientCh == nil {
			t.Fatal("WebSocket client was not registered")
		}
		clientCh <- []byte(`{"event":"metrics"}`)
		time.Sleep(11 * time.Second)
		synctest.Wait()

		srv.clients.mu.RLock()
		clientCount := len(srv.clients.m)
		srv.clients.mu.RUnlock()
		if clientCount != 0 {
			t.Fatalf("blocked WebSocket clients = %d; want 0", clientCount)
		}
		if !conn.IsClosedByPeer() {
			t.Fatal("server left the timed-out WebSocket open")
		}
	})
}
