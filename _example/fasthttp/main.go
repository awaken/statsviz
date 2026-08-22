package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/fasthttp/router"
	"github.com/soheilhy/cmux"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/fasthttpadaptor"

	"github.com/arl/statsviz"
	example "github.com/arl/statsviz/_example"
)

func main() {
	// Force the GC to work to make the plots "move".
	go example.Work()

	// Create the main listener and mux
	l, err := example.Listen(8083)
	if err != nil {
		log.Fatalf("failed to create listener: %s", err)
	}
	m := cmux.New(l)
	m.SetReadTimeout(example.ServerReadHeaderTimeout)
	ws := http.NewServeMux()

	// fasthttp routers
	r := router.New()
	r.GET("/", func(ctx *fasthttp.RequestCtx) {
		fmt.Fprintf(ctx, "Hello, world!")
	})

	// Create statsviz server.
	srv, _ := statsviz.NewServer()

	// Register Statsviz server on the fasthttp router.
	r.GET("/debug/statsviz/{filepath:*}", fasthttpadaptor.NewFastHTTPHandler(srv.Index()))
	ws.HandleFunc("/debug/statsviz/ws", srv.Ws())

	// Server start
	wsListener := m.Match(cmux.HTTP1HeaderField("Upgrade", "websocket"))
	httpListener := m.Match(cmux.Any())
	wsServer := example.HTTPServer(8083, ws)
	fastHTTPServer := &fasthttp.Server{
		Handler:      r.Handler,
		ReadTimeout:  example.ServerReadTimeout,
		WriteTimeout: example.ServerWriteTimeout,
		IdleTimeout:  example.ServerIdleTimeout,
	}
	go func() {
		if err := wsServer.Serve(wsListener); err != nil {
			log.Fatalf("failed to serve WebSocket connections: %s", err)
		}
	}()
	go func() {
		if err := fastHTTPServer.Serve(httpListener); err != nil {
			log.Fatalf("failed to serve HTTP connections: %s", err)
		}
	}()
	fmt.Printf("Point your browser to %s\n", example.URL("http", 8083, "/debug/statsviz/"))
	if err := m.Serve(); err != nil {
		log.Fatalf("failed to serve multiplexed connections: %s", err)
	}
}
