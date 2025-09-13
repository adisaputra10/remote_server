package main

import (
"encoding/json"
"flag"
"net/http"
"os"

"github.com/gorilla/websocket"
"remote-tunnel/internal/logger"
)

var upgrader = websocket.Upgrader{
CheckOrigin: func(r *http.Request) bool {
return true
},
}

func handleAgent(w http.ResponseWriter, r *http.Request) {
log := logger.New("RELAY")
log.Info(" Agent connection from %s", r.RemoteAddr)

conn, err := upgrader.Upgrade(w, r, nil)
if err != nil {
log.Error(" Failed to upgrade: %v", err)
return
}
defer conn.Close()

log.Info(" Agent WebSocket connected")

for {
var msg map[string]interface{}
if err := conn.ReadJSON(&msg); err != nil {
log.Error(" Agent disconnected: %v", err)
break
}
log.Info(" Agent message: %v", msg)
}
}

func handleClient(w http.ResponseWriter, r *http.Request) {
log := logger.New("RELAY")
log.Info(" Client connection from %s", r.RemoteAddr)

conn, err := upgrader.Upgrade(w, r, nil)
if err != nil {
log.Error(" Failed to upgrade: %v", err)
return
}
defer conn.Close()

log.Info(" Client WebSocket connected")

for {
var msg map[string]interface{}
if err := conn.ReadJSON(&msg); err != nil {
log.Error(" Client disconnected: %v", err)
break
}
log.Info(" Client message: %v", msg)
}
}

func main() {
addr := flag.String("addr", ":8080", "Server address")
flag.Parse()

log := logger.New("RELAY")
log.Info(" Starting relay server on %s", *addr)

http.HandleFunc("/ws/agent", handleAgent)
http.HandleFunc("/ws/client", handleClient)
http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
w.Header().Set("Content-Type", "application/json")
json.NewEncoder(w).Encode(map[string]string{"status": "ok", "service": "tunnel-relay"})
})

log.Info(" Agent endpoint: ws://localhost%s/ws/agent", *addr)
log.Info(" Client endpoint: ws://localhost%s/ws/client", *addr)

if err := http.ListenAndServe(*addr, nil); err != nil {
log.Error(" Server error: %v", err)
os.Exit(1)
}
}
