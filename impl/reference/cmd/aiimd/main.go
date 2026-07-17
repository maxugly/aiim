// Command aiimd is the AIIM reference daemon.
// It listens for WebSocket connections on /aiim/v1 and performs the
// AIIM handshake (HELLO → ACK → READY with challenge-response auth).
//
// Usage: go run ./cmd/aiimd/ [--addr :9191]
package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/maxugly/aiim/pkg/identity"
	"github.com/maxugly/aiim/pkg/protocol"

	"golang.org/x/net/websocket"
)

var (
	addr      = flag.String("addr", ":9191", "listen address")
	agentID   = flag.String("agent-id", "agent:aiimd@localhost", "this agent's identity")
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

	// Register WebSocket handler
	http.Handle("/aiim/v1", websocket.Handler(func(ws *websocket.Conn) {
		handleConnection(ws, kp, trust)
	}))

	// Health check
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status":"ok","agent":"%s","version":"0.1.0"}`, *agentID)
	})

	log.Printf("Listening on %s", *addr)
	if err := http.ListenAndServe(*addr, nil); err != nil {
		log.Fatalf("server error: %v", err)
		os.Exit(1)
	}
}

func handleConnection(ws *websocket.Conn, kp *identity.KeyPair, trust *identity.TrustStore) {
	defer ws.Close()

	remoteAddr := ws.Request().RemoteAddr
	log.Printf("[%s] new connection", remoteAddr)

	result, err := protocol.HandleHandshake(ws, *agentID, kp, trust)
	if err != nil {
		log.Printf("[%s] handshake failed: %v", remoteAddr, err)
		// HandleHandshake already sent the ERROR frame
		return
	}

	log.Printf("[%s] handshake complete — agent=%s session=%s version=%s capabilities=%v",
		remoteAddr, result.AgentID, result.SessionID, result.Version, result.Capabilities)

	// Channel is ACTIVE. For the reference impl, echo session info and wait.
	ack := map[string]interface{}{
		"status":     "active",
		"session_id": result.SessionID,
		"agent_id":   result.AgentID,
		"version":    result.Version,
		"message":    "handshake complete — channel active",
	}
	if err := protocol.WriteFrame(ws, ack); err != nil {
		log.Printf("[%s] write error: %v", remoteAddr, err)
	}
}
