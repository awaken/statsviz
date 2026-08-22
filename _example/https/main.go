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

	// Use your own certificates and key files.
	const certFile = "./cert.pem"
	const keyFile = "./key.pem"

	mux := http.NewServeMux()

	// Register Statsviz handlers on the mux.
	_ = statsviz.Register(mux)

	fmt.Printf("Point your browser to %s\n", example.URL("https", 8087, "/debug/statsviz/"))
	log.Fatal(example.HTTPServer(8087, mux).ListenAndServeTLS(certFile, keyFile))
}
