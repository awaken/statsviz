package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/arl/statsviz"
	example "github.com/arl/statsviz/_example"
)

func main() {
	// Force the GC to work to make the plots "move".
	go example.Work()

	// Create a Gorilla router.
	r := mux.NewRouter()

	// Create statsviz server and register the handlers on the router.
	srv, _ := statsviz.NewServer()
	r.Methods("GET").Path("/debug/statsviz/ws").Name("GET /debug/statsviz/ws").HandlerFunc(srv.Ws())
	r.Methods("GET").PathPrefix("/debug/statsviz/").Name("GET /debug/statsviz/").Handler(srv.Index())

	mux := http.NewServeMux()
	mux.Handle("/", r)

	fmt.Printf("Point your browser to %s\n", example.URL("http", 8086, "/debug/statsviz/"))
	if err := example.HTTPServer(8086, mux).ListenAndServe(); err != nil {
		log.Fatalf("failed to start server: %s", err)
	}
}
