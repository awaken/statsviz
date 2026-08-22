package main

import (
	"log"
	"net/http"
	"time"

	"github.com/arl/statsviz"
	example "github.com/arl/statsviz/_example"
)

func main() {
	// Force the GC to work to make the plots "move".
	go example.Work()

	mux := http.NewServeMux()

	// Register Statsviz server on the mux, serving the user interface from
	// /foo/bar instead of /debug/statsviz and send metrics every 250
	// milliseconds instead of the default of once per second.
	_ = statsviz.Register(mux,
		statsviz.Root("/foo/bar"),
		statsviz.SendFrequency(250*time.Millisecond),
	)

	log.Printf("Point your browser to %s", example.URL("http", 8092, "/foo/bar"))
	log.Fatal(example.HTTPServer(8092, mux).ListenAndServe())
}
