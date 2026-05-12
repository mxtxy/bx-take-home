package main

import "testing"

func TestSessionSecretFromEnv_AllowsDevelopmentDefault(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	t.Setenv("JWT_SECRET", "")

	secret, err := sessionSecretFromEnv()

	if err != nil {
		t.Fatalf("sessionSecretFromEnv: %v", err)
	}
	if secret != "dev_only_change_me" {
		t.Fatalf("secret = %q", secret)
	}
}

func TestSessionSecretFromEnv_RejectsMissingProductionSecret(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("JWT_SECRET", "")

	if _, err := sessionSecretFromEnv(); err == nil {
		t.Fatal("expected missing production secret to fail")
	}
}

func TestSessionSecretFromEnv_RejectsDefaultProductionSecret(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("JWT_SECRET", "dev_only_change_me")

	if _, err := sessionSecretFromEnv(); err == nil {
		t.Fatal("expected default production secret to fail")
	}
}

func TestSessionSecretFromEnv_RejectsShortProductionSecret(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("JWT_SECRET", "short-secret")

	if _, err := sessionSecretFromEnv(); err == nil {
		t.Fatal("expected short production secret to fail")
	}
}
