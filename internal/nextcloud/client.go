// Package nextcloud kümmert sich um den Upload per WebDAV (R4) und die
// Erzeugung eines öffentlichen, ablaufenden Links per OCS-API (R5).
package nextcloud

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/fritzthekid/printqrview/internal/config"
)

const httpTimeout = 30 * time.Second

// Error wird bei fehlgeschlagenem Upload oder fehlgeschlagener
// Share-Erstellung zurückgegeben.
type Error struct {
	Msg string
}

func (e *Error) Error() string { return e.Msg }

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

func davURL(cfg *config.Config, remotePath string) string {
	return fmt.Sprintf("%s/remote.php/dav/files/%s%s", cfg.BaseURL, cfg.Username, remotePath)
}

// ensureRemoteDir legt das Zielverzeichnis (inkl. Zwischenebenen) per WebDAV
// MKCOL an. Nextcloud legt bei einem PUT keine fehlenden Elternordner an,
// deshalb muss das Zielverzeichnis vorher existieren. Ein bereits
// vorhandener Ordner meldet sich mit 405 (Method Not Allowed) und wird
// bewusst ignoriert.
func ensureRemoteDir(cfg *config.Config) error {
	client := &http.Client{Timeout: httpTimeout}
	path := ""
	for _, segment := range strings.Split(strings.Trim(cfg.TargetDir, "/"), "/") {
		if segment == "" {
			continue
		}
		path += "/" + segment
		req, err := http.NewRequest("MKCOL", davURL(cfg, path), nil)
		if err != nil {
			return err
		}
		req.SetBasicAuth(cfg.Username, cfg.Password)

		resp, err := client.Do(req)
		if err != nil {
			return &Error{Msg: fmt.Sprintf("Anlegen des Zielordners fehlgeschlagen: %v", err)}
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusMethodNotAllowed {
			return &Error{Msg: fmt.Sprintf("Anlegen des Zielordners fehlgeschlagen (%d): %s", resp.StatusCode, truncate(string(body), 200))}
		}
	}
	return nil
}

// UploadFile lädt data per WebDAV PUT unter filename in cfg.TargetDir hoch
// und liefert den entstandenen Remote-Pfad zurück.
func UploadFile(data []byte, filename string, cfg *config.Config) (string, error) {
	if err := ensureRemoteDir(cfg); err != nil {
		return "", err
	}

	remotePath := strings.TrimRight(cfg.TargetDir, "/") + "/" + filename

	req, err := http.NewRequest(http.MethodPut, davURL(cfg, remotePath), bytes.NewReader(data))
	if err != nil {
		return "", err
	}
	req.SetBasicAuth(cfg.Username, cfg.Password)

	client := &http.Client{Timeout: httpTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return "", &Error{Msg: fmt.Sprintf("Upload fehlgeschlagen: %v", err)}
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK, http.StatusCreated, http.StatusNoContent:
		return remotePath, nil
	default:
		return "", &Error{Msg: fmt.Sprintf("Upload fehlgeschlagen (%d): %s", resp.StatusCode, truncate(string(body), 200))}
	}
}

type shareResponse struct {
	OCS struct {
		Data struct {
			URL string `json:"url"`
		} `json:"data"`
	} `json:"ocs"`
}

// CreatePublicLink erzeugt für remotePath über die OCS-Share-API
// (shareType=3, öffentlicher Link) einen Freigabelink, der nach
// cfg.LinkExpireDays Tagen abläuft.
//
// Der reine Share-Link (".../s/TOKEN") liefert die HTML-Vorschauseite der
// Nextcloud-Weboberfläche. Mit angehängtem "/download" antwortet der Server
// stattdessen mit "Content-Disposition: attachment", sodass Browser die
// Datei direkt herunterladen bzw. im Handy-PDF-Viewer öffnen, statt sie
// zuerst im Web-Cloud-Viewer anzuzeigen.
func CreatePublicLink(remotePath string, cfg *config.Config) (string, error) {
	shareURL := fmt.Sprintf("%s/ocs/v2.php/apps/files_sharing/api/v1/shares", cfg.BaseURL)
	expireDate := time.Now().AddDate(0, 0, cfg.LinkExpireDays).Format("2006-01-02")

	form := url.Values{}
	form.Set("path", remotePath)
	form.Set("shareType", "3")
	form.Set("expireDate", expireDate)

	req, err := http.NewRequest(http.MethodPost, shareURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("OCS-APIRequest", "true")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(cfg.Username, cfg.Password)

	client := &http.Client{Timeout: httpTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return "", &Error{Msg: fmt.Sprintf("Share-Erstellung fehlgeschlagen: %v", err)}
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return "", &Error{Msg: fmt.Sprintf("Share-Erstellung fehlgeschlagen (%d): %s", resp.StatusCode, truncate(string(body), 200))}
	}

	var parsed shareResponse
	if err := json.Unmarshal(body, &parsed); err != nil || parsed.OCS.Data.URL == "" {
		return "", &Error{Msg: fmt.Sprintf("Unerwartete Antwort der Share-API: %s", truncate(string(body), 200))}
	}

	return strings.TrimRight(parsed.OCS.Data.URL, "/") + "/download", nil
}
