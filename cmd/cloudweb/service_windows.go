//go:build windows

package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"golang.org/x/sys/windows/svc"
)

// serviceName muss mit dem Namen übereinstimmen, unter dem
// deploy/windows/install.ps1 den Dienst per New-Service anlegt, sonst
// erkennt svc.IsWindowsService() den Dienst-Kontext nicht zuverlässig.
const serviceName = "CloudWeb"

// runService startet den HTTP-Server und den Druck-Ordner-Watcher
// (watcher_windows.go). Läuft der Prozess unter der Kontrolle des Service
// Control Managers (installiert per New-Service), übernimmt handler den
// Start/Stop-Handshake; interaktiv gestartet (z. B. beim Testen in der VM
// per "cloudweb.exe" in einer Konsole) läuft alles blockierend direkt,
// ohne SCM-Handshake - praktisch zum Entwickeln, ohne jedes Mal den Dienst
// neu registrieren zu müssen.
func runService(st *state, mux http.Handler, addr string) {
	isService, err := svc.IsWindowsService()
	if err != nil {
		log.Fatalf("konnte Service-Kontext nicht bestimmen: %v", err)
	}
	if !isService {
		runInteractive(st, mux, addr)
		return
	}
	if err := svc.Run(serviceName, &serviceHandler{st: st, mux: mux, addr: addr}); err != nil {
		log.Fatalf("Dienst beendet mit Fehler: %v", err)
	}
}

func runInteractive(st *state, mux http.Handler, addr string) {
	stop := make(chan struct{})
	go startPrintWatcher(st, stop)

	srv := &http.Server{Addr: addr, Handler: mux}
	log.Printf("cloudweb hört auf %s (interaktiv, kein Windows-Dienst)", addr)
	log.Fatal(srv.ListenAndServe())
}

type serviceHandler struct {
	st   *state
	mux  http.Handler
	addr string
}

// Execute implementiert svc.Handler - siehe
// https://pkg.go.dev/golang.org/x/sys/windows/svc#Handler. Der Service
// Control Manager erwartet zeitnah eine erste Statusmeldung, danach
// wiederholt entweder eine neue Statusänderung oder spätestens vor Ablauf
// der (Standard-)Wartezeit ein Lebenszeichen.
func (h *serviceHandler) Execute(args []string, r <-chan svc.ChangeRequest, s chan<- svc.Status) (bool, uint32) {
	const accepted = svc.AcceptStop | svc.AcceptShutdown

	s <- svc.Status{State: svc.StartPending}

	stop := make(chan struct{})
	go startPrintWatcher(h.st, stop)

	srv := &http.Server{Addr: h.addr, Handler: h.mux}
	serveErr := make(chan error, 1)
	go func() { serveErr <- srv.ListenAndServe() }()

	s <- svc.Status{State: svc.Running, Accepts: accepted}

	for {
		select {
		case err := <-serveErr:
			log.Printf("HTTP-Server beendet: %v", err)
			close(stop)
			s <- svc.Status{State: svc.StopPending}
			return false, 1
		case req := <-r:
			switch req.Cmd {
			case svc.Interrogate:
				s <- req.CurrentStatus
			case svc.Stop, svc.Shutdown:
				s <- svc.Status{State: svc.StopPending}
				close(stop)
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				_ = srv.Shutdown(ctx)
				s <- svc.Status{State: svc.Stopped}
				return false, 0
			}
		}
	}
}
