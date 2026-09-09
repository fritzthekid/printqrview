// Package config liest die Treiber-Konfiguration ausschließlich aus
// Umgebungsvariablen (R6).
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	BaseURL        string
	Username       string
	Password       string
	TargetDir      string
	LinkExpireDays int
}

// FromEnv baut die Config aus env. Ist env nil, wird os.Environ verwendet.
func FromEnv(env map[string]string) (*Config, error) {
	lookup := func(key string) (string, bool) {
		if env != nil {
			v, ok := env[key]
			return v, ok
		}
		return os.LookupEnv(key)
	}

	values := map[string]string{}
	var missing []string
	for _, key := range []string{"NC_BASE_URL", "NC_USERNAME", "NC_PASSWORD"} {
		v, ok := lookup(key)
		if !ok {
			missing = append(missing, key)
			continue
		}
		values[key] = v
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("Fehlende Umgebungsvariable(n): %s", strings.Join(missing, ", "))
	}

	targetDir := "/PrinterUploads"
	if v, ok := lookup("NC_TARGET_DIR"); ok {
		targetDir = v
	}

	expireDays := 1
	if v, ok := lookup("NC_LINK_EXPIRE_DAYS"); ok {
		n, err := strconv.Atoi(v)
		if err != nil {
			return nil, fmt.Errorf("Ungültiger Wert für NC_LINK_EXPIRE_DAYS: %q", v)
		}
		expireDays = n
	}

	return &Config{
		BaseURL:        strings.TrimRight(values["NC_BASE_URL"], "/"),
		Username:       values["NC_USERNAME"],
		Password:       values["NC_PASSWORD"],
		TargetDir:      targetDir,
		LinkExpireDays: expireDays,
	}, nil
}
