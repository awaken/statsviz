package statsviz

import (
	"context"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"github.com/arl/statsviz/internal/plot"
)

type clients struct {
	cfg *plot.Config
	ctx context.Context
	writeTimeout time.Duration

	mu sync.RWMutex
	m  map[*websocket.Conn]chan []byte
}

func newClients(ctx context.Context, cfg *plot.Config, writeTimeout time.Duration) *clients {
	return &clients{
		m:            make(map[*websocket.Conn]chan []byte),
		cfg:          cfg,
		ctx:          ctx,
		writeTimeout: writeTimeout,
	}
}

type wsmsg struct {
	Event string `json:"event"`
	Data  any    `json:"data"`
}

func (c *clients) add(conn *websocket.Conn) {
	dbglog("adding client")
	stopClose := context.AfterFunc(c.ctx, func() {
		_ = conn.Close()
	})

	// Send config first.
	err := c.writeJSON(conn, wsmsg{Event: "config", Data: c.cfg})
	if err != nil {
		stopClose()
		_ = conn.Close()
		dbglog("failed to send config: %v", err)
		return
	}

	ch := make(chan []byte)
	c.mu.Lock()
	c.m[conn] = ch
	c.mu.Unlock()

	go func() {
		defer func() {
			stopClose()

			c.mu.Lock()
			delete(c.m, conn)
			c.mu.Unlock()
			_ = conn.Close()

			dbglog("removed client")
		}()

		for {
			select {
			case <-c.ctx.Done():
				return
			case msg := <-ch:
				if err := c.sendbuf(conn, msg); err != nil {
					dbglog("failed to send data: %v", err)
					return
				}
			}
		}
	}()
}

func (c *clients) setWriteDeadline(conn *websocket.Conn) error {
	return conn.SetWriteDeadline(time.Now().Add(c.writeTimeout))
}

func (c *clients) writeJSON(conn *websocket.Conn, value any) error {
	if err := c.setWriteDeadline(conn); err != nil {
		return err
	}
	return conn.WriteJSON(value)
}

func (c *clients) sendbuf(conn *websocket.Conn, buf []byte) error {
	if err := c.setWriteDeadline(conn); err != nil {
		return err
	}
	w, err := conn.NextWriter(websocket.TextMessage)
	if err != nil {
		return err
	}
	_, err1 := w.Write(buf)
	err2 := w.Close()
	if err1 != nil {
		return err1
	}
	return err2
}

func (c *clients) broadcast(buf []byte) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	for _, ch := range c.m {
		select {
		case ch <- buf:
		default:
			// if a client is not keeping up, we
			// drop the message for that client.
			dbglog("dropping message to client")
		}
	}
}
