// Command aiimd is the AIIM reference daemon.
// It listens for WebSocket connections on /aiim/v1, serves the presence
// dashboard at /presence.html, and .presence.json for the dashboard to poll.
//
// Usage: go run ./cmd/aiimd/ [--addr :9191] [--project-dir ../../..]
package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/maxugly/aiim/pkg/identity"
	"github.com/maxugly/aiim/pkg/protocol"

	"golang.org/x/net/websocket"
)

var (
	addr       = flag.String("addr", ":9191", "listen address")
	agentID    = flag.String("agent-id", "agent:aiimd@localhost", "this agent's identity")
	projectDir = flag.String("project-dir", ".", "AIIM project root (serves presence.html + .presence.json)")
)

func main() {
	flag.Parse()

	// Generate server identity
	kp, err := identity.GenerateKeyPair()
	if err != nil {
		log.Fatalf("generating keypair: %v", err)
	}
	log.Printf("AIIM daemon starting as %s", *agentID)
	log.Printf("Public key: %s", kp.PublicKeyBase64())

	trust := identity.NewTrustStore()

	absDir, err := filepath.Abs(*projectDir)
	if err != nil {
		log.Fatalf("resolving project dir: %v", err)
	}

	// Register WebSocket handler
	http.Handle("/aiim/v1", websocket.Handler(func(ws *websocket.Conn) {
		handleConnection(ws, trust)
	}))

	// Health check
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status":"ok","agent":"%s","version":"0.1.0"}`, *agentID)
	})

	// Serve presence dashboard and .presence.json from project root
	fs := http.FileServer(http.Dir(absDir))
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Don't interfere with WebSocket or health routes
		if r.URL.Path == "/aiim/v1" || r.URL.Path == "/health" {
			return
		}
		log.Printf("[http] %s %s", r.Method, r.URL.Path)
		fs.ServeHTTP(w, r)
	})

	log.Printf("Project dir: %s", absDir)
	log.Printf("Dashboard: http://localhost%s/presence.html", *addr)
	log.Printf("Listening on %s", *addr)
	if err := http.ListenAndServe(*addr, nil); err != nil {
		log.Fatalf("server error: %v", err)
		os.Exit(1)
	}
}

func handleConnection(ws *websocket.Conn, trust *identity.TrustStore) {
	defer ws.Close()

	remoteAddr := ws.Request().RemoteAddr
	log.Printf("[%s] new connection", remoteAddr)

	result, err := protocol.HandleHandshake(ws, *agentID, trust)
	if err != nil {
		log.Printf("[%s] handshake failed: %v", remoteAddr, err)
		return
	}

	log.Printf("[%s] handshake complete — agent=%s session=%s version=%s capabilities=%v",
		remoteAddr, result.AgentID, result.SessionID, result.Version, result.Capabilities)

	log.Printf("[%s] channel ACTIVE — session=%s", remoteAddr, result.SessionID)
}
