package sharepassword

import (
	"strings"
	"testing"
)

func TestDeriveMatchesKnownVector(t *testing.T) {
	// Testvektor per Python bestätigt:
	//   hashlib.sha256(b"password") -> base32(...)[:16] == "L2EERGG2FACHCUOQ"
	// In Fünfer-Gruppen: "L2EER-GG2FA-CHCUO-Q" (siehe FormatGroups).
	crypt, password := Derive("password", "password", 16)
	if crypt != "L2EER-GG2FA-CHCUO-Q" {
		t.Errorf("crypt = %q, want %q", crypt, "L2EER-GG2FA-CHCUO-Q")
	}
	if password != "passwordL2EER-GG2FA-CHCUO-Q" {
		t.Errorf("password = %q, want %q", password, "passwordL2EER-GG2FA-CHCUO-Q")
	}
}

func TestDeriveUsesDefaultLengthWhenZeroOrNegative(t *testing.T) {
	cryptZero, _ := Derive("password", "password", 0)
	cryptDefault, _ := Derive("password", "password", DefaultLength)
	if cryptZero != cryptDefault {
		t.Errorf("Derive(..., 0) = %q, want dasselbe wie Derive(..., DefaultLength) = %q", cryptZero, cryptDefault)
	}
	cryptNegative, _ := Derive("password", "password", -1)
	if cryptNegative != cryptDefault {
		t.Errorf("Derive(..., -1) = %q, want dasselbe wie Derive(..., DefaultLength) = %q", cryptNegative, cryptDefault)
	}
}

func TestDeriveIsDeterministic(t *testing.T) {
	crypt1, password1 := Derive("username", "username+hardening", 16)
	crypt2, password2 := Derive("username", "username+hardening", 16)
	if crypt1 != crypt2 || password1 != password2 {
		t.Errorf("Derive() nicht deterministisch: (%q,%q) != (%q,%q)", crypt1, password1, crypt2, password2)
	}
}

func TestDifferentHardenedSeedsProduceDifferentCrypts(t *testing.T) {
	crypt1, _ := Derive("alice", "alice+seed1", 16)
	crypt2, _ := Derive("alice", "alice+seed2", 16)
	if crypt1 == crypt2 {
		t.Errorf("unterschiedliche hardenedSeed lieferten denselben crypt: %q", crypt1)
	}
}

func TestPasswordIsNamePlusCryptNotHardenedSeedPlusCrypt(t *testing.T) {
	// Der Empfänger kennt nur name (mündlich) und crypt (angezeigt) - das
	// tatsächliche Passwort darf deshalb NICHT den (dem Empfänger
	// unbekannten) hardenedSeed enthalten, sonst kann er es nie korrekt
	// eintippen (genau dieser Bug wurde live beobachtet und hier fixiert).
	crypt, password := Derive("alice", "alice+mac-adresse+remote-path", 16)
	if password != "alice"+crypt {
		t.Errorf("password = %q, want name+crypt = %q", password, "alice"+crypt)
	}
}

func TestFormatGroupsSplitsIntoFives(t *testing.T) {
	got := FormatGroups("SJBB3ZMD7PR4HIQ")
	want := "SJBB3-ZMD7P-R4HIQ"
	if got != want {
		t.Errorf("FormatGroups() = %q, want %q", got, want)
	}
}

func TestFormatGroupsHandlesRemainder(t *testing.T) {
	got := FormatGroups("L2EERGG2FACHCUOQ") // 16 Zeichen, nicht durch 5 teilbar
	want := "L2EER-GG2FA-CHCUO-Q"
	if got != want {
		t.Errorf("FormatGroups() = %q, want %q", got, want)
	}
}

func TestFormatGroupsLeavesShortCodeUnchanged(t *testing.T) {
	got := FormatGroups("ABCDE")
	if got != "ABCDE" {
		t.Errorf("FormatGroups() = %q, want unverändert %q", got, "ABCDE")
	}
}

func TestLocalMACReturnsPlausibleAddressAndIsStable(t *testing.T) {
	mac1, err := LocalMAC()
	if err != nil {
		t.Fatalf("LocalMAC() error = %v (Testumgebung ohne Netzwerkschnittstelle mit MAC?)", err)
	}
	if !strings.Contains(mac1, ":") {
		t.Errorf("LocalMAC() = %q, sieht nicht wie eine MAC-Adresse aus", mac1)
	}

	mac2, err := LocalMAC()
	if err != nil {
		t.Fatalf("LocalMAC() (zweiter Aufruf) error = %v", err)
	}
	if mac1 != mac2 {
		t.Errorf("LocalMAC() nicht stabil: %q != %q", mac1, mac2)
	}
}
