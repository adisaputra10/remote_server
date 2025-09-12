package main

import (
"encoding/json"
"flag"
"net/http"
"os"
"remote-tunnel/internal/logger"
)

func main() {
addr := flag.String("addr", ":8080", "Server address")
flag.Parse()

log := logger.New("RELAY")
log.Info("Starting relay server on %s", *addr)

http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
w.Header().Set("Content-Type", "application/json")
json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
})

if err := http.ListenAndServe(*addr, nil); err != nil {
log.Error("Server error: %v", err)
os.Exit(1)
}
}
