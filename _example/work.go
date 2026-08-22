package example

import (
	"math/rand"
	"net"
	"net/http"
	"os"
	"reflect"
	"runtime"
	"strconv"
	"time"
)

const (
	// HostEnv selects the host used by the runnable examples. It defaults to
	// the IPv4 loopback address; set it explicitly to opt in to remote access.
	HostEnv = "STATSVIZ_EXAMPLE_HOST"

	// ServerReadHeaderTimeout limits how long an example accepts request headers.
	ServerReadHeaderTimeout = 5 * time.Second
	// ServerReadTimeout limits how long an example reads a complete request.
	ServerReadTimeout = 15 * time.Second
	// ServerWriteTimeout limits how long an example writes a response.
	ServerWriteTimeout = 15 * time.Second
	// ServerIdleTimeout limits how long an example keeps an idle connection.
	ServerIdleTimeout = 60 * time.Second
)

// Address returns the TCP address used by a runnable example. Examples bind to
// 127.0.0.1 unless HostEnv explicitly selects another host.
func Address(port int) string {
	host := os.Getenv(HostEnv)
	if host == "" {
		host = "127.0.0.1"
	}
	return net.JoinHostPort(host, strconv.Itoa(port))
}

// URL returns a URL for a runnable example endpoint.
func URL(scheme string, port int, path string) string {
	return scheme + "://" + Address(port) + path
}

// HTTPServer returns an HTTP server with the example address and conservative
// request, response, and idle timeouts.
func HTTPServer(port int, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              Address(port),
		Handler:           handler,
		ReadHeaderTimeout: ServerReadHeaderTimeout,
		ReadTimeout:       ServerReadTimeout,
		WriteTimeout:      ServerWriteTimeout,
		IdleTimeout:       ServerIdleTimeout,
	}
}

// Listen creates a TCP listener on the example address.
func Listen(port int) (net.Listener, error) {
	return net.Listen("tcp", Address(port))
}

// Work loops forever, generating allocations of various sizes, in order to
// create artificial work for a nice 'demo effect'.
func Work() {
	m := make(map[int64]any)
	tick := time.NewTicker(30 * time.Millisecond)
	clearTick := time.NewTicker(1 * time.Second)
	for {
		select {
		case <-clearTick.C:
			if rand.Intn(100) < 5 {
				runtime.GC()
			}
			if rand.Intn(100) < 2 {
				m = make(map[int64]any)
			}
		case ts := <-tick.C:
			m[ts.UnixNano()] = newStruct()
		}
	}
}

// create a randomly sized struct (to create 'motion' on size classes plot).
func newStruct() any {
	nfields := rand.Intn(32)
	var fields []reflect.StructField
	for i := 0; i < nfields; i++ {
		fields = append(fields, reflect.StructField{
			Name:    "f" + strconv.Itoa(i),
			PkgPath: "main",
			Type:    reflect.TypeOf(""),
		})
	}
	return reflect.New(reflect.StructOf(fields)).Interface()
}
