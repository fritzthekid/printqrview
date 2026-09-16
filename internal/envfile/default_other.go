//go:build !windows

package envfile

// LoadDefault ist unter Linux/macOS ein No-op: dort übernimmt systemds
// "EnvironmentFile=" (siehe deploy/*.service) das Einlesen der
// Konfigurationsdatei, Go-Code muss dafür nichts tun.
func LoadDefault() error {
	return nil
}
