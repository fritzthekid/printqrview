// Package envfile lädt eine einfache KEY=VALUE-Konfigurationsdatei in die
// Prozess-Umgebung - das Windows-Äquivalent zu systemds "EnvironmentFile="
// (siehe deploy/backend.env.example, deploy/*.service), das es unter
// Windows-Diensten (New-Service) nicht gibt.
package envfile

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// Load liest path zeilenweise als KEY=VALUE (leere Zeilen und mit "#"
// beginnende Kommentarzeilen werden übersprungen) und setzt jede Variable
// per os.Setenv. Eine bereits gesetzte Umgebungsvariable wird überschrieben,
// damit eine Konfigurationsdatei verlässlich das gilt, was in ihr steht.
func Load(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			return fmt.Errorf("%s:%d: keine KEY=VALUE-Zeile: %q", path, lineNo, line)
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if key == "" {
			return fmt.Errorf("%s:%d: leerer Schlüssel", path, lineNo)
		}
		if err := os.Setenv(key, value); err != nil {
			return err
		}
	}
	return scanner.Err()
}
