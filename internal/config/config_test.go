package config

import (
	"os"
	"path/filepath"
	"testing"
)

func setValidDBEnv(t *testing.T) {
	t.Helper()

	t.Setenv("POSTGRES_HOST", "localhost")
	t.Setenv("POSTGRES_PORT", "5432")
	t.Setenv("POSTGRES_USER", "test_user")
	t.Setenv("POSTGRES_PASSWORD", "test_password")
	t.Setenv("POSTGRES_DB", "test_db")
}

func setValidBotEnv(t *testing.T) {
	t.Helper()

	t.Setenv("TGBOTAPI_TOKEN", "test-token")
	t.Setenv("BOT_MODE", "polling")
}

func TestNewDBConfigSuccess(t *testing.T) {
	setValidDBEnv(t)

	cfg, err := NewDBConfig()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if cfg.Host != "localhost" {
		t.Errorf("expected host localhost, got %s", cfg.Host)
	}

	if cfg.Port != 5432 {
		t.Errorf("expected port 5432, got %d", cfg.Port)
	}

	if cfg.Username != "test_user" {
		t.Errorf("expected username test_user, got %s", cfg.Username)
	}

	if cfg.DBName != "test_db" {
		t.Errorf("expected db name test_db, got %s", cfg.DBName)
	}
}

func TestNewDBConfigMissingRequiredVariable(t *testing.T) {
	requiredVariables := []string{
		"POSTGRES_HOST",
		"POSTGRES_PORT",
		"POSTGRES_USER",
		"POSTGRES_PASSWORD",
		"POSTGRES_DB",
	}

	for _, variable := range requiredVariables {
		t.Run(variable, func(t *testing.T) {
			setValidDBEnv(t)
			t.Setenv(variable, "")

			_, err := NewDBConfig()
			if err == nil {
				t.Fatalf("expected error for missing %s", variable)
			}
		})
	}
}

func TestNewDBConfigInvalidPort(t *testing.T) {
	testCases := []string{
		"abc",
		"12.5",
		"-10",
	}

	for _, port := range testCases {
		t.Run(port, func(t *testing.T) {
			setValidDBEnv(t)
			t.Setenv("POSTGRES_PORT", port)

			_, err := NewDBConfig()
			if err == nil {
				t.Fatalf("expected error for port %s", port)
			}
		})
	}
}

func TestNewDBConfigPortOutOfRange(t *testing.T) {
	testCases := []string{
		"0",
		"65536",
	}

	for _, port := range testCases {
		t.Run(port, func(t *testing.T) {
			setValidDBEnv(t)
			t.Setenv("POSTGRES_PORT", port)

			_, err := NewDBConfig()
			if err == nil {
				t.Fatalf("expected error for port %s", port)
			}
		})
	}
}

func TestNewBotConfigSuccess(t *testing.T) {
	setValidBotEnv(t)

	cfg, err := NewBotConfig()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if cfg.Token != "test-token" {
		t.Errorf("expected token test-token, got %s", cfg.Token)
	}

	if cfg.BotMode != "polling" {
		t.Errorf("expected bot mode polling, got %s", cfg.BotMode)
	}
}

func TestNewBotConfigMissingValues(t *testing.T) {
	testCases := []struct {
		name  string
		token string
		mode  string
	}{
		{
			name:  "missing token",
			token: "",
			mode:  "polling",
		},
		{
			name:  "missing mode",
			token: "test-token",
			mode:  "",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Setenv("TGBOTAPI_TOKEN", testCase.token)
			t.Setenv("BOT_MODE", testCase.mode)

			_, err := NewBotConfig()
			if err == nil {
				t.Fatalf("expected error")
			}
		})
	}
}

func TestNewBotConfigInvalidMode(t *testing.T) {
	t.Setenv("TGBOTAPI_TOKEN", "test-token")
	t.Setenv("BOT_MODE", "webhook")

	_, err := NewBotConfig()
	if err == nil {
		t.Fatal("expected error for invalid bot mode")
	}
}

func TestLoadSuccess(t *testing.T) {
	tempDir := t.TempDir()

	envContent := `TGBOTAPI_TOKEN=test-token
BOT_MODE=polling
POSTGRES_HOST=localhost
POSTGRES_PORT=5432
POSTGRES_USER=test_user
POSTGRES_PASSWORD=test_password
POSTGRES_DB=test_db
`

	err := os.WriteFile(
		filepath.Join(tempDir, ".env"),
		[]byte(envContent),
		0600,
	)
	if err != nil {
		t.Fatalf("failed to create .env: %v", err)
	}

	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current directory: %v", err)
	}

	err = os.Chdir(tempDir)
	if err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}

	t.Cleanup(func() {
		_ = os.Chdir(oldDir)
	})

	for _, variable := range []string{
		"TGBOTAPI_TOKEN",
		"BOT_MODE",
		"POSTGRES_HOST",
		"POSTGRES_PORT",
		"POSTGRES_USER",
		"POSTGRES_PASSWORD",
		"POSTGRES_DB",
	} {
		_ = os.Unsetenv(variable)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if cfg.Bot.BotMode != "polling" {
		t.Errorf("expected polling mode, got %s", cfg.Bot.BotMode)
	}

	if cfg.DB.Port != 5432 {
		t.Errorf("expected port 5432, got %d", cfg.DB.Port)
	}
}

func TestLoadWithoutEnvFile(t *testing.T) {
	tempDir := t.TempDir()

	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current directory: %v", err)
	}

	err = os.Chdir(tempDir)
	if err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}

	t.Cleanup(func() {
		_ = os.Chdir(oldDir)
	})

	_, err = Load()
	if err == nil {
		t.Fatal("expected error when .env file is missing")
	}
}
