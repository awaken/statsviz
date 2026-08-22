package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/kataras/iris/v12"

	"github.com/arl/statsviz"
	example "github.com/arl/statsviz/_example"
)

func main() {
	// Force the GC to work to make the plots "move".
	go example.Work()

	app := iris.New()

	// Need to run iris in a separate goroutine so we can start the dedicated
	// http server for Statsviz.
	go func() {
		if err := app.Run(iris.Server(example.HTTPServer(8089, nil))); err != nil {
			log.Fatalf("failed to start Iris server: %s", err)
		}
	}()

	mux := http.NewServeMux()

	// Register Statsviz handlers on the mux.
	_ = statsviz.Register(mux)

	fmt.Printf("Point your browser to %s\n", example.URL("http", 8088, "/debug/statsviz"))

	// NewHost puts the http server for statsviz under the control of iris but
	// iris won't touch its handlers.
	if err := app.NewHost(example.HTTPServer(8088, mux)).ListenAndServe(); err != nil {
		log.Fatalf("failed to start Statsviz server: %s", err)
	}
}
