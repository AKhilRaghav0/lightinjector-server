package main

import (
	"io"
	"log"
	"net"
	"net/http"
	"os"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true }, // allow all origins
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("LightInjector SSH Server\n"))
	})

	// WebSocket → SSH tunnel
	http.HandleFunc("/ssh", handleWebSocket)

	// HTTP CONNECT fallback (works on non-Cloudflare paths)
	http.HandleFunc("/connect", handleConnect)

	log.Printf("LightInjector listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("WS upgrade:", err)
		return
	}
	defer ws.Close()

	ssh, err := net.Dial("tcp", "127.0.0.1:2222")
	if err != nil {
		log.Println("SSH dial:", err)
		return
	}
	defer ssh.Close()

	log.Printf("WS tunnel: %s → SSH", r.RemoteAddr)

	// ws → ssh
	go func() {
		for {
			_, msg, err := ws.ReadMessage()
			if err != nil {
				return
			}
			if _, err := ssh.Write(msg); err != nil {
				return
			}
		}
	}()

	// ssh → ws
	buf := make([]byte, 32*1024)
	for {
		n, err := ssh.Read(buf)
		if n > 0 {
			if err2 := ws.WriteMessage(websocket.BinaryMessage, buf[:n]); err2 != nil {
				return
			}
		}
		if err != nil {
			return
		}
	}
}

func handleConnect(w http.ResponseWriter, r *http.Request) {
	ssh, err := net.Dial("tcp", "127.0.0.1:2222")
	if err != nil {
		http.Error(w, "SSH unreachable", 502)
		return
	}
	defer ssh.Close()

	hijacker, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "not supported", 500)
		return
	}
	client, _, err := hijacker.Hijack()
	if err != nil {
		return
	}
	defer client.Close()

	client.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))
	done := make(chan struct{}, 2)
	go func() { io.Copy(ssh, client); done <- struct{}{} }()
	go func() { io.Copy(client, ssh); done <- struct{}{} }()
	<-done
}
