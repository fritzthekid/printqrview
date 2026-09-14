package filenames

import (
	"strings"
	"testing"
	"time"
)

func ts() time.Time {
	return time.Date(2026, 9, 7, 14, 30, 5, 0, time.UTC)
}

func TestGeneratesNameWithTimestampAndTitle(t *testing.T) {
	name := Generate("Rechnung September", ".pdf", ts())
	want := "20260907-143005_Rechnung_September.pdf"
	if name != want {
		t.Errorf("Generate() = %q, want %q", name, want)
	}
}

func TestSanitizesUnsafeCharacters(t *testing.T) {
	name := Generate("Bericht: Q3/Ä Ö Ü??.docx", ".pdf", ts())
	if !strings.HasPrefix(name, "20260907-143005_") {
		t.Errorf("Generate() = %q, missing timestamp prefix", name)
	}
	if !strings.HasSuffix(name, ".pdf") {
		t.Errorf("Generate() = %q, missing .pdf suffix", name)
	}
	stem := strings.TrimSuffix(name, ".pdf")
	for _, c := range "äöü/:? " {
		if strings.ContainsRune(stem, c) {
			t.Errorf("Generate() = %q, still contains unsafe rune %q", name, c)
		}
	}
}

func TestEmptyTitleFallsBackToDefault(t *testing.T) {
	name := Generate("   ", ".pdf", ts())
	want := "20260907-143005_print.pdf"
	if name != want {
		t.Errorf("Generate() = %q, want %q", name, want)
	}
}

func TestTwoCallsAtDifferentTimesAreUnique(t *testing.T) {
	name1 := Generate("Test", ".pdf", time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	name2 := Generate("Test", ".pdf", time.Date(2026, 1, 1, 0, 0, 1, 0, time.UTC))
	if name1 == name2 {
		t.Errorf("Generate() returned identical names for different timestamps: %q", name1)
	}
}

func TestCustomExtensionIsUsed(t *testing.T) {
	name := Generate("test", ".zip", ts())
	want := "20260907-143005_test.zip"
	if name != want {
		t.Errorf("Generate() = %q, want %q", name, want)
	}
}

func TestExtensionWithoutLeadingDotIsNormalized(t *testing.T) {
	name := Generate("test", "zip", ts())
	want := "20260907-143005_test.zip"
	if name != want {
		t.Errorf("Generate() = %q, want %q", name, want)
	}
}

func TestDetectExtension(t *testing.T) {
	cases := []struct {
		name string
		data []byte
		want string
	}{
		{"PDF", []byte("%PDF-1.4 ..."), ".pdf"},
		{"ZIP lokaler Dateikopf", []byte("PK\x03\x04 ..."), ".zip"},
		{"ZIP leeres Archiv", []byte("PK\x05\x06 ..."), ".zip"},
		{"ZIP mit Data-Descriptor", []byte("PK\x07\x08 ..."), ".zip"},
		{"PNG", []byte("\x89PNG\r\n\x1a\n..."), ".png"},
		{"JPEG", []byte{0xFF, 0xD8, 0xFF, 0xE0}, ".jpg"},
		{"GIF87a", []byte("GIF87a..."), ".gif"},
		{"GIF89a", []byte("GIF89a..."), ".gif"},
		{"unbekannt", []byte("irgendwelche Bytes"), ".bin"},
		{"leer", []byte{}, ".bin"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := DetectExtension(c.data)
			if got != c.want {
				t.Errorf("DetectExtension(%q) = %q, want %q", c.data, got, c.want)
			}
		})
	}
}
