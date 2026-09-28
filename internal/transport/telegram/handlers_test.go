package telegram

import (
	"testing"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func TestParseCallbackData(t *testing.T) {
	tests := []struct {
		name    string
		data    string
		action  string
		tgID    int64
		role    string
		wantErr bool
	}{
		{
			name:   "approve",
			data:   "approve:8281761514",
			action: "approve",
			tgID:   8281761514,
		},
		{
			name:   "reject",
			data:   "reject:8281761514",
			action: "reject",
			tgID:   8281761514,
		},
		{
			name:   "role",
			data:   "role:8281761514:Dispatcher",
			action: "role",
			tgID:   8281761514,
			role:   "Dispatcher",
		},
		{
			name:    "empty data",
			data:    "",
			wantErr: true,
		},
		{
			name:    "invalid id",
			data:    "approve:abc",
			wantErr: true,
		},
		{
			name:    "missing role",
			data:    "role:8281761514",
			wantErr: true,
		},
		{
			name:    "unknown action",
			data:    "delete:8281761514",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			action, tgID, role, err := parseCallbackData(tt.data)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if action != tt.action {
				t.Errorf("action = %q, want %q", action, tt.action)
			}

			if tgID != tt.tgID {
				t.Errorf("tgID = %d, want %d", tgID, tt.tgID)
			}

			if role != tt.role {
				t.Errorf("role = %q, want %q", role, tt.role)
			}
		})
	}
}

func TestIsOwnTelegramContact(t *testing.T) {
	const tgID int64 = 8281761514

	tests := []struct {
		name    string
		contact *tgbotapi.Contact
		want    bool
	}{
		{
			name: "own contact",
			contact: &tgbotapi.Contact{
				PhoneNumber: "+992900000000",
				UserID:      tgID,
			},
			want: true,
		},
		{
			name: "foreign contact",
			contact: &tgbotapi.Contact{
				PhoneNumber: "+992900000001",
				UserID:      1111111111,
			},
			want: false,
		},
		{
			name: "empty phone",
			contact: &tgbotapi.Contact{
				UserID: tgID,
			},
			want: false,
		},
		{
			name:    "nil contact",
			contact: nil,
			want:    false,
		},
		{
			name: "contact without user id",
			contact: &tgbotapi.Contact{
				PhoneNumber: "+992900000000",
				UserID:      0,
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isOwnTelegramContact(tt.contact, tgID)

			if got != tt.want {
				t.Fatalf("isOwnTelegramContact() = %v, want %v", got, tt.want)
			}
		})
	}
}
