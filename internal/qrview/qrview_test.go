package qrview

import (
	"os"
	"path/filepath"
	"testing"
)

var pngMagic = []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}

func TestWritePNGCreatesValidPNG(t *testing.T) {
	out := filepath.Join(t.TempDir(), "code.png")

	if err := WritePNG("https://cloud.orthos.selfhost.eu/s/abc123", out); err != nil {
		t.Fatalf("WritePNG() error = %v", err)
	}

	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("output file missing: %v", err)
	}
	if len(data) < len(pngMagic) {
		t.Fatalf("file too short: %d bytes", len(data))
	}
	for i, b := range pngMagic {
		if data[i] != b {
			t.Fatalf("not a PNG file, got header %v", data[:len(pngMagic)])
		}
	}
}

func TestPNGBytesReturnsValidPNG(t *testing.T) {
	data, err := PNGBytes("https://cloud.orthos.selfhost.eu/s/abc123")
	if err != nil {
		t.Fatalf("PNGBytes() error = %v", err)
	}
	if len(data) < len(pngMagic) {
		t.Fatalf("data too short: %d bytes", len(data))
	}
	for i, b := range pngMagic {
		if data[i] != b {
			t.Fatalf("not a PNG file, got header %v", data[:len(pngMagic)])
		}
	}
}
