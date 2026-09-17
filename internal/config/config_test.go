package config

import (
	"strings"
	"testing"
)

func TestFromEnvReadsAllFields(t *testing.T) {
	env := map[string]string{
		"NC_BASE_URL":   "https://cloud.example.com/",
		"NC_USERNAME":   "change-cloud-user",
		"NC_PASSWORD":   "app-token",
		"NC_TARGET_DIR": "/Drucke",
	}
	cfg, err := FromEnv(env)
	if err != nil {
		t.Fatalf("FromEnv() error = %v", err)
	}
	if cfg.BaseURL != "https://cloud.example.com" {
		t.Errorf("BaseURL = %q, want trailing slash removed", cfg.BaseURL)
	}
	if cfg.Username != "change-cloud-user" || cfg.Password != "app-token" {
		t.Errorf("unexpected credentials: %+v", cfg)
	}
	if cfg.TargetDir != "/Drucke" {
		t.Errorf("TargetDir = %q, want /Drucke", cfg.TargetDir)
	}
}

func TestFromEnvUsesDefaultTargetDir(t *testing.T) {
	env := map[string]string{
		"NC_BASE_URL": "https://cloud.example.com",
		"NC_USERNAME": "change-cloud-user",
		"NC_PASSWORD": "app-token",
	}
	cfg, err := FromEnv(env)
	if err != nil {
		t.Fatalf("FromEnv() error = %v", err)
	}
	if cfg.TargetDir != "/PrinterUploads" {
		t.Errorf("TargetDir = %q, want default", cfg.TargetDir)
	}
}

func TestFromEnvUsesDefaultLinkExpireDays(t *testing.T) {
	env := map[string]string{
		"NC_BASE_URL": "https://cloud.example.com",
		"NC_USERNAME": "change-cloud-user",
		"NC_PASSWORD": "app-token",
	}
	cfg, err := FromEnv(env)
	if err != nil {
		t.Fatalf("FromEnv() error = %v", err)
	}
	if cfg.LinkExpireDays != 1 {
		t.Errorf("LinkExpireDays = %d, want 1", cfg.LinkExpireDays)
	}
}

func TestFromEnvReadsCustomLinkExpireDays(t *testing.T) {
	env := map[string]string{
		"NC_BASE_URL":         "https://cloud.example.com",
		"NC_USERNAME":         "change-cloud-user",
		"NC_PASSWORD":         "app-token",
		"NC_LINK_EXPIRE_DAYS": "7",
	}
	cfg, err := FromEnv(env)
	if err != nil {
		t.Fatalf("FromEnv() error = %v", err)
	}
	if cfg.LinkExpireDays != 7 {
		t.Errorf("LinkExpireDays = %d, want 7", cfg.LinkExpireDays)
	}
}

func TestFromEnvUsesDefaultLenCode(t *testing.T) {
	env := map[string]string{
		"NC_BASE_URL": "https://cloud.example.com",
		"NC_USERNAME": "change-cloud-user",
		"NC_PASSWORD": "app-token",
	}
	cfg, err := FromEnv(env)
	if err != nil {
		t.Fatalf("FromEnv() error = %v", err)
	}
	if cfg.LenCode != 15 {
		t.Errorf("LenCode = %d, want 15 (sharepassword.DefaultLength)", cfg.LenCode)
	}
}

func TestFromEnvReadsCustomLenCode(t *testing.T) {
	env := map[string]string{
		"NC_BASE_URL": "https://cloud.example.com",
		"NC_USERNAME": "change-cloud-user",
		"NC_PASSWORD": "app-token",
		"NC_LEN_CODE": "20",
	}
	cfg, err := FromEnv(env)
	if err != nil {
		t.Fatalf("FromEnv() error = %v", err)
	}
	if cfg.LenCode != 20 {
		t.Errorf("LenCode = %d, want 20", cfg.LenCode)
	}
}

func TestFromEnvRejectsInvalidLenCode(t *testing.T) {
	env := map[string]string{
		"NC_BASE_URL": "https://cloud.example.com",
		"NC_USERNAME": "change-cloud-user",
		"NC_PASSWORD": "app-token",
		"NC_LEN_CODE": "nicht-numerisch",
	}
	if _, err := FromEnv(env); err == nil {
		t.Fatal("FromEnv() mit ungültigem NC_LEN_CODE: erwarteter Fehler blieb aus")
	}
}

func TestFromEnvRaisesOnMissingRequiredVar(t *testing.T) {
	base := map[string]string{
		"NC_BASE_URL": "https://cloud.example.com",
		"NC_USERNAME": "change-cloud-user",
		"NC_PASSWORD": "app-token",
	}
	for _, missingKey := range []string{"NC_BASE_URL", "NC_USERNAME", "NC_PASSWORD"} {
		env := map[string]string{}
		for k, v := range base {
			env[k] = v
		}
		delete(env, missingKey)

		_, err := FromEnv(env)
		if err == nil {
			t.Fatalf("FromEnv() with missing %s: expected error", missingKey)
		}
		if !strings.Contains(err.Error(), missingKey) {
			t.Errorf("error %q does not mention %s", err.Error(), missingKey)
		}
	}
}
