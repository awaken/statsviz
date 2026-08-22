package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gofiber/fiber/v3"
	"github.com/soheilhy/cmux"

	"github.com/arl/statsviz"
	example "github.com/arl/statsviz/_example"
)

func main() {
	// Force the GC to work to make the plots "move".
	go example.Work()

	// Create the main listener and mux
	l, err := example.Listen(8093)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	m := cmux.New(l)
	m.SetReadTimeout(example.ServerReadHeaderTimeout)
	ws := http.NewServeMux()

	// Fiber instance
	app := fiber.New(fiber.Config{
		ReadTimeout:  example.ServerReadTimeout,
		WriteTimeout: example.ServerWriteTimeout,
		IdleTimeout:  example.ServerIdleTimeout,
	})

	// Create statsviz server.
	srv, err := statsviz.NewServer()
	if err != nil {
		panic(err)
	}

	app.Use("/debug/statsviz/", srv.Index())
	ws.HandleFunc("/debug/statsviz/ws", srv.Ws())

	fmt.Printf("Point your browser to %s\n", example.URL("http", 8093, "/debug/statsviz/"))

	// Server start
	wsListener := m.Match(cmux.HTTP1HeaderField("Upgrade", "websocket"))
	fiberListener := m.Match(cmux.Any())
	wsServer := example.HTTPServer(8093, ws)
	go func() {
		if err := wsServer.Serve(wsListener); err != nil {
			log.Fatalf("failed to serve WebSocket connections: %s", err)
		}
	}()
	go func() {
		if err := app.Listener(fiberListener); err != nil {
			log.Fatalf("failed to serve Fiber connections: %s", err)
		}
	}()
	if err := m.Serve(); err != nil {
		log.Fatalf("failed to serve multiplexed connections: %s", err)
	}
}
