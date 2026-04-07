// HTTP CONNECT proxy + WebSocket-to-SSH bridge
// Runs on $PORT (Render's HTTPS port), forwards to SSH on :2222
package main

import (
	"io"
	"log"
	"net"
	"net/http"
	"os"

	"golang.org/x/net/websocket"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	mux := http.NewServeMux()

	// Health check — Render pings this
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "CONNECT" {
			handleConnect(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(200)
		w.Write([]byte("LightInjector SSH Server\n"))
	})

	// WebSocket tunnel — LightInjector connects here for SSH
	mux.Handle("/ssh", websocket.Handler(handleWebSocket))

	log.Printf("LightInjector proxy listening on :%s", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatal(err)
	}
}

// handleConnect handles HTTP CONNECT requests (proxy mode)
func handleConnect(w http.ResponseWriter, r *http.Request) {
	ssh, err := net.Dial("tcp", "127.0.0.1:2222")
	if err != nil {
		http.Error(w, "SSH unreachable", http.StatusBadGateway)
		return
	}
	defer ssh.Close()

	hijacker, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "hijack not supported", http.StatusInternalServerError)
		return
	}
	client, _, err := hijacker.Hijack()
	if err != nil {
		return
	}
	defer client.Close()

	client.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))
	pipe(client, ssh)
}

// handleWebSocket tunnels WebSocket frames to the SSH server
func handleWebSocket(ws *websocket.Conn) {
	ws.PayloadType = websocket.BinaryFrame
	ssh, err := net.Dial("tcp", "127.0.0.1:2222")
	if err != nil {
		log.Println("SSH dial failed:", err)
		return
	}
	defer ssh.Close()
	defer ws.Close()
	pipe(ws, ssh)
}

func pipe(a, b io.ReadWriter) {
	done := make(chan struct{}, 2)
	go func() { io.Copy(a, b); done <- struct{}{} }()
	go func() { io.Copy(b, a); done <- struct{}{} }()
	<-done
}
