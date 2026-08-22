package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/arl/statsviz"
	example "github.com/arl/statsviz/_example"
)

func main() {
	// Force the GC to work to make the plots "move".
	go example.Work()

	// Register a Statsviz server on the default mux.
	statsviz.Register(http.DefaultServeMux)

	fmt.Printf("Point your browser to %s\n", example.URL("http", 8080, "/debug/statsviz/"))
	log.Fatal(example.HTTPServer(8080, nil).ListenAndServe())
}
