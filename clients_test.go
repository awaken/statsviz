package statsviz

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"github.com/arl/statsviz/internal/plot"
)

// Control frames must work even when no metrics update is due.
func TestAuditWebSocketControls(t *testing.T) {
	for _, kind := range []int{websocket.PingMessage, websocket.CloseMessage} {
		t.Run(map[int]string{websocket.PingMessage: "ping", websocket.CloseMessage: "close"}[kind], func(t *testing.T) {
			srv := newServer(t, SendFrequency(time.Hour))
			httpSrv := httptest.NewServer(srv.Ws())
			defer httpSrv.Close()
			ws, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(httpSrv.URL, "http"), nil)
			if err != nil {
				t.Fatal(err)
			}
			defer ws.Close()
			if err := ws.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
				t.Fatal(err)
			}
			var config wsmsg
			if err := ws.ReadJSON(&config); err != nil || config.Event != "config" {
				t.Fatalf("initial config = %q, %v", config.Event, err)
			}

			payload := []byte("owned-test")
			pongReceived := errors.New("pong received")
			ws.SetPongHandler(func(got string) error {
				if got != string(payload) {
					t.Errorf("pong payload = %q, want %q", got, payload)
				}
				return pongReceived
			})
			if kind == websocket.CloseMessage {
				payload = websocket.FormatCloseMessage(websocket.CloseNormalClosure, "done")
			}
			if err := ws.WriteControl(kind, payload, time.Now().Add(time.Second)); err != nil {
				t.Fatal(err)
			}
			_, _, err = ws.ReadMessage()
			if kind == websocket.PingMessage && !errors.Is(err, pongReceived) {
				t.Errorf("ping response = %v, want matching pong", err)
			}
			if kind == websocket.CloseMessage && !websocket.IsCloseError(err, websocket.CloseNormalClosure) {
				t.Errorf("close response = %v, want normal close acknowledgement", err)
			}
		})
	}
}

func auditWebSocket(t *testing.T) (*websocket.Conn, *websocket.Conn) {
	t.Helper()
	conns := make(chan *websocket.Conn, 1)
	httpSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Error(err)
		}
		conns <- conn
	}))
	t.Cleanup(httpSrv.Close)
	peer, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(httpSrv.URL, "http"), nil)
	if err != nil {
		t.Fatal(err)
	}
	conn := <-conns
	if conn == nil {
		t.Fatal("upgrade failed")
	}
	t.Cleanup(func() { conn.Close(); peer.Close() })
	return conn, peer
}

func TestAuditClientFailures(t *testing.T) {
	t.Run("invalid config releases connection", func(t *testing.T) {
		conn, peer := auditWebSocket(t)
		c := newClients(context.Background(), &plot.Config{Series: []any{func() {}}}, time.Second)
		c.add(conn)
		if len(c.m) != 0 {
			t.Fatal("failed config retained client")
		}
		peer.SetReadDeadline(time.Now().Add(time.Second))
		if _, data, err := peer.ReadMessage(); err == nil {
			// Gorilla may flush an empty frame while closing a failed JSON writer.
			if len(data) != 0 {
				t.Fatalf("failed config sent data %q", data)
			}
			if _, _, err := peer.ReadMessage(); err == nil {
				t.Fatal("failed config left connection open")
			}
		}
	})
	for _, size := range []int{1, 1 << 16} {
		t.Run(strings.Repeat("large", size>>16)+"closed write", func(t *testing.T) {
			conn, _ := auditWebSocket(t)
			conn.Close()
			c := newClients(context.Background(), nil, time.Second)
			for range 2 {
				if err := c.sendbuf(conn, bytes.Repeat([]byte("x"), size)); err == nil {
					t.Fatal("closed connection accepted data")
				}
			}
		})
	}
	t.Run("slow client cannot block ready client", func(t *testing.T) {
		conn, _ := auditWebSocket(t)
		c := newClients(context.Background(), nil, time.Second)
		ready := make(chan []byte, 1)
		c.m[nil] = make(chan []byte)
		c.m[conn] = ready
		c.broadcast([]byte("sample"))
		if got := string(<-ready); got != "sample" {
			t.Fatalf("ready client got %q", got)
		}
	})
}
