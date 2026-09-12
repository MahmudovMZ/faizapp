package config

import "testing"

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
