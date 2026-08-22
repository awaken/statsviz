//go:build unix

package statsviz_test

import (
	"bytes"
	"context"
	"crypto/tls"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	example "github.com/arl/statsviz/_example"

	"github.com/gorilla/websocket"
	"github.com/rogpeppe/go-internal/gotooltest"
	"github.com/rogpeppe/go-internal/testscript"
)

func TestExampleServerDefaults(t *testing.T) {
	t.Setenv(example.HostEnv, "")
	if got, want := example.Address(8080), "127.0.0.1:8080"; got != want {
		t.Fatalf("default example address = %q; want %q", got, want)
	}
	server := example.HTTPServer(8080, http.NotFoundHandler())
	if server.ReadHeaderTimeout != example.ServerReadHeaderTimeout ||
		server.ReadTimeout != example.ServerReadTimeout ||
		server.WriteTimeout != example.ServerWriteTimeout ||
		server.IdleTimeout != example.ServerIdleTimeout {
		t.Fatalf("example server timeouts = %#v", server)
	}

	t.Setenv(example.HostEnv, "0.0.0.0")
	if got, want := example.Address(8080), "0.0.0.0:8080"; got != want {
		t.Fatalf("opt-in example address = %q; want %q", got, want)
	}
}

func TestExamples(t *testing.T) {
	if testing.Short() {
		t.Skipf("TestExamples skipped in short mode")
	}

	p := testscript.Params{
		Dir: "testdata",
		Setup: func(env *testscript.Env) error {
			// We want to run scripts with the local version of Statsviz.
			// Provide scripts with statsviz root dir so we can use a
			// 'go mod -edit replace' directive.
			wd, err := os.Getwd()
			if err != nil {
				return err
			}
			env.Setenv("STATSVIZ_ROOT", wd)
			return nil
		},
		Cmds: map[string]func(ts *testscript.TestScript, neg bool, args []string){
			"checkui": checkui,
		},
	}

	if err := gotooltest.Setup(&p); err != nil {
		t.Fatal(err)
	}
	testscript.Run(t, p)
}

// checkui requests statsviz url from a script.
// In a script, run it with:
//
//	checkui url [basic_auth_user basic_auth_pwd]
func checkui(ts *testscript.TestScript, neg bool, args []string) {
	if len(args) != 1 && len(args) != 3 {
		ts.Fatalf(`checkui: wrong number of arguments. Call with "checkui URL [BASIC_USER BASIC_PWD]`)
	}
	u := args[0]
	ts.Logf("checkui: loading web page %s", args[0])

	const startupTimeout = 5 * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), startupTimeout)
	defer cancel()

	tlsConfig := &tls.Config{InsecureSkipVerify: true}
	transport := &http.Transport{TLSClientConfig: tlsConfig}
	defer transport.CloseIdleConnections()
	client := http.Client{
		Transport: transport,
		Timeout:   5 * time.Second,
	}

	var resp *http.Response
	for {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
		if err != nil {
			ts.Fatalf("checkui: bad request: %v", err)
		}
		if len(args) == 3 {
			req.SetBasicAuth(args[1], args[2])
		}

		resp, err = client.Do(req)
		if err == nil {
			break
		}
		select {
		case <-ctx.Done():
			ts.Fatalf("checkui: server did not become ready within %s: %v", startupTimeout, err)
		case <-time.After(50 * time.Millisecond):
		}
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		ts.Fatalf("checkui: HTTP status %s, want %s", resp.Status, http.StatusText(http.StatusOK))
	}

	body, err := io.ReadAll(resp.Body)
	ts.Check(err)

	const want = `id="plots"`
	if !bytes.Contains(body, []byte(want)) {
		ts.Fatalf("checkui: response body doesn't contain %s\n\nbody;\n\n%s", want, body)
	}

	checkWebSocket(ts, u, tlsConfig, args)
}

func checkWebSocket(ts *testscript.TestScript, uiURL string, tlsConfig *tls.Config, args []string) {
	u, err := url.Parse(uiURL)
	if err != nil {
		ts.Fatalf("checkui: bad WebSocket URL: %v", err)
	}
	switch u.Scheme {
	case "http":
		u.Scheme = "ws"
	case "https":
		u.Scheme = "wss"
	default:
		ts.Fatalf("checkui: unsupported URL scheme %q", u.Scheme)
	}
	u.Path = strings.TrimSuffix(u.Path, "/") + "/ws"

	header := http.Header{}
	if len(args) == 3 {
		req := http.Request{Header: header}
		req.SetBasicAuth(args[1], args[2])
	}

	const websocketTimeout = 5 * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), websocketTimeout)
	defer cancel()
	dialer := websocket.Dialer{
		HandshakeTimeout: websocketTimeout,
		TLSClientConfig:  tlsConfig,
	}
	ws, resp, err := dialer.DialContext(ctx, u.String(), header)
	if err != nil {
		if resp != nil {
			_ = resp.Body.Close()
		}
		ts.Fatalf("checkui: WebSocket handshake failed: %v", err)
	}
	defer ws.Close()
	if err := ws.SetReadDeadline(time.Now().Add(websocketTimeout)); err != nil {
		ts.Fatalf("checkui: setting WebSocket read deadline: %v", err)
	}

	var msg struct {
		Event string `json:"event"`
	}
	if err := ws.ReadJSON(&msg); err != nil {
		ts.Fatalf("checkui: reading WebSocket config: %v", err)
	}
	if msg.Event != "config" {
		ts.Fatalf("checkui: first WebSocket event is %q, want %q", msg.Event, "config")
	}
}
