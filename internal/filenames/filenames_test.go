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
