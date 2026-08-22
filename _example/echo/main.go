package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/labstack/echo/v5"

	"github.com/arl/statsviz"
	example "github.com/arl/statsviz/_example"
)

func main() {
	// Force the GC to work to make the plots "move".
	go example.Work()

	// Echo instance
	e := echo.New()

	mux := http.NewServeMux()

	// Register statsviz handlerson the mux.
	statsviz.Register(mux)

	// Use echo WrapHandler to wrap statsviz ServeMux as echo HandleFunc
	e.GET("/debug/statsviz/", echo.WrapHandler(mux))
	// Serve static content for statsviz UI
	e.GET("/debug/statsviz/*", echo.WrapHandler(mux))

	// Start server
	fmt.Printf("Point your browser to %s\n", example.URL("http", 8082, "/debug/statsviz/"))
	if err := example.HTTPServer(8082, e).ListenAndServe(); err != nil {
		log.Fatalf("failed to start server: %s", err)
	}
}
