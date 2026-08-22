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

	mux := http.NewServeMux()

	// Register Statsviz handlers on the mux.
	_ = statsviz.Register(mux)

	fmt.Printf("Point your browser to %s\n", example.URL("http", 8091, "/debug/statsviz/"))
	log.Fatal(example.HTTPServer(8091, mux).ListenAndServe())
}
