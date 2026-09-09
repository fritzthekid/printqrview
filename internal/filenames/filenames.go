// Package filenames erzeugt eindeutige, dateisystemsichere Dateinamen aus
// Jobtitel + Zeitstempel (R3).
package filenames

import (
	"regexp"
	"strings"
	"time"
)

var unsafe = regexp.MustCompile(`[^A-Za-z0-9_.-]+`)

// Generate baut "<timestamp>_<sicherer-titel><ext>". ext ohne führenden Punkt
// wird automatisch normalisiert; ein leerer safeTitle fällt auf "print" zurück.
func Generate(jobTitle string, ext string, timestamp time.Time) string {
	if ext != "" && !strings.HasPrefix(ext, ".") {
		ext = "." + ext
	}
	ts := timestamp.Format("20060102-150405")
	safeTitle := strings.Trim(unsafe.ReplaceAllString(strings.TrimSpace(jobTitle), "_"), "_")
	if safeTitle == "" {
		safeTitle = "print"
	}
	return ts + "_" + safeTitle + ext
}
