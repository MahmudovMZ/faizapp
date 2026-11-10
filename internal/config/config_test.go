package config

import (
	"testing"
	"time"
)

func TestNewBotConfig(t *testing.T) {
	tests := []struct {
		name      string
		token     string
		mode      string
		adminID   string
		wantErr   bool
		wantAdmin int64
	}{
		{
			name:      "valid config",
			token:     "test-token",
			mode:      "polling",
			adminID:   "8281761514",
			wantAdmin: 8281761514,
		},
		{
			name:    "missing token",
			mode:    "polling",
			adminID: "8281761514",
			wantErr: true,
		},
		{
			name:    "missing admin id",
			token:   "test-token",
			mode:    "polling",
			wantErr: true,
		},
		{
			name:    "invalid admin id",
			token:   "test-token",
			mode:    "polling",
			adminID: "abc",
			wantErr: true,
		},
		{
			name:    "zero admin id",
			token:   "test-token",
			mode:    "polling",
			adminID: "0",
			wantErr: true,
		},
		{
			name:    "negative admin id",
			token:   "test-token",
			mode:    "polling",
			adminID: "-10",
			wantErr: true,
		},
		{
			name:    "unsupported bot mode",
			token:   "test-token",
			mode:    "webhook",
			adminID: "8281761514",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("TGBOTAPI_TOKEN", tt.token)
			t.Setenv("BOT_MODE", tt.mode)
			t.Setenv("ADMIN_TELEGRAM_ID", tt.adminID)
			t.Setenv("PHOTO_TIMEOUT", "")
			t.Setenv("PHOTO_WORKERS", "")

			cfg, err := NewBotConfig()

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if cfg.AdminTGID != tt.wantAdmin {
				t.Errorf(
					"AdminTGID = %d, want %d",
					cfg.AdminTGID,
					tt.wantAdmin,
				)
			}

			if cfg.Token != tt.token {
				t.Errorf("Token = %q, want %q", cfg.Token, tt.token)
			}

			if cfg.BotMode != tt.mode {
				t.Errorf("BotMode = %q, want %q", cfg.BotMode, tt.mode)
			}
		})
	}
}

func TestNewBotConfigPhotoTimeout(t *testing.T) {
	tests := []struct {
		name        string
		value       string
		wantErr     bool
		wantTimeout time.Duration
	}{
		{
			name:        "not set uses default",
			value:       "",
			wantTimeout: 60 * time.Second,
		},
		{
			name:        "custom seconds",
			value:       "90s",
			wantTimeout: 90 * time.Second,
		},
		{
			name:        "custom minutes",
			value:       "2m",
			wantTimeout: 2 * time.Minute,
		},
		{
			name:    "not a duration",
			value:   "abc",
			wantErr: true,
		},
		{
			name:    "number without unit",
			value:   "60",
			wantErr: true,
		},
		{
			name:    "zero",
			value:   "0s",
			wantErr: true,
		},
		{
			name:    "negative",
			value:   "-5s",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("TGBOTAPI_TOKEN", "test-token")
			t.Setenv("BOT_MODE", "polling")
			t.Setenv("ADMIN_TELEGRAM_ID", "8281761514")
			t.Setenv("PHOTO_TIMEOUT", tt.value)
			t.Setenv("PHOTO_WORKERS", "")

			cfg, err := NewBotConfig()

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if cfg.PhotoTimeout != tt.wantTimeout {
				t.Errorf(
					"PhotoTimeout = %v, want %v",
					cfg.PhotoTimeout,
					tt.wantTimeout,
				)
			}
		})
	}
}

func TestNewBotConfigPhotoWorkers(t *testing.T) {
	tests := []struct {
		name        string
		value       string
		wantErr     bool
		wantWorkers int
	}{
		{
			name:        "not set uses default",
			value:       "",
			wantWorkers: 4,
		},
		{
			name:        "minimum",
			value:       "1",
			wantWorkers: 1,
		},
		{
			name:        "custom value",
			value:       "6",
			wantWorkers: 6,
		},
		{
			name:        "maximum",
			value:       "8",
			wantWorkers: 8,
		},
		{
			name:    "not a number",
			value:   "abc",
			wantErr: true,
		},
		{
			name:    "zero",
			value:   "0",
			wantErr: true,
		},
		{
			name:    "negative",
			value:   "-2",
			wantErr: true,
		},
		{
			name:    "above maximum",
			value:   "9",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("TGBOTAPI_TOKEN", "test-token")
			t.Setenv("BOT_MODE", "polling")
			t.Setenv("ADMIN_TELEGRAM_ID", "8281761514")
			t.Setenv("PHOTO_TIMEOUT", "")
			t.Setenv("PHOTO_WORKERS", tt.value)

			cfg, err := NewBotConfig()

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if cfg.PhotoWorkers != tt.wantWorkers {
				t.Errorf(
					"PhotoWorkers = %d, want %d",
					cfg.PhotoWorkers,
					tt.wantWorkers,
				)
			}
		})
	}
}
