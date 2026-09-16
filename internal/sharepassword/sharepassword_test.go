package sharepassword

import (
	"strings"
	"testing"
)

func TestDeriveMatchesKnownVector(t *testing.T) {
	// Testvektor per Python bestätigt:
	//   hashlib.sha256(b"password") -> base32(...)[:16] == "L2EERGG2FACHCUOQ"
	crypt, password := Derive("password", "password")
	if crypt != "L2EERGG2FACHCUOQ" {
		t.Errorf("crypt = %q, want %q", crypt, "L2EERGG2FACHCUOQ")
	}
	if password != "passwordL2EERGG2FACHCUOQ" {
		t.Errorf("password = %q, want %q", password, "passwordL2EERGG2FACHCUOQ")
	}
}

func TestDeriveIsDeterministic(t *testing.T) {
	crypt1, password1 := Derive("username", "username+hardening")
	crypt2, password2 := Derive("username", "username+hardening")
	if crypt1 != crypt2 || password1 != password2 {
		t.Errorf("Derive() nicht deterministisch: (%q,%q) != (%q,%q)", crypt1, password1, crypt2, password2)
	}
}

func TestDifferentHardenedSeedsProduceDifferentCrypts(t *testing.T) {
	crypt1, _ := Derive("alice", "alice+seed1")
	crypt2, _ := Derive("alice", "alice+seed2")
	if crypt1 == crypt2 {
		t.Errorf("unterschiedliche hardenedSeed lieferten denselben crypt: %q", crypt1)
	}
}

func TestPasswordIsNamePlusCryptNotHardenedSeedPlusCrypt(t *testing.T) {
	// Der Empfänger kennt nur name (mündlich) und crypt (angezeigt) - das
	// tatsächliche Passwort darf deshalb NICHT den (dem Empfänger
	// unbekannten) hardenedSeed enthalten, sonst kann er es nie korrekt
	// eintippen (genau dieser Bug wurde live beobachtet und hier fixiert).
	crypt, password := Derive("alice", "alice+mac-adresse+remote-path")
	if password != "alice"+crypt {
		t.Errorf("password = %q, want name+crypt = %q", password, "alice"+crypt)
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
