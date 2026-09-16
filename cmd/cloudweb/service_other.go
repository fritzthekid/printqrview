//go:build !windows

package main

import (
	"log"
	"net/http"
)

// runService startet den HTTP-Server blockierend. Unter Linux/macOS läuft
// cloudweb als gewöhnlicher systemd-Dienst (siehe deploy/cloudweb.service)
// - kein Service-Control-Manager-Handshake und kein Ordner-Watcher nötig,
// das übernimmt dort das CUPS-Backend (cmd/backend) per Push an /push.
func runService(st *state, mux http.Handler, addr string) {
	log.Printf("cloudweb hört auf %s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}
