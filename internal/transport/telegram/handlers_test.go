package telegram

import (
	"testing"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func TestParseCallbackData(t *testing.T) {
	tests := []struct {
		name     string
		data     string
		action   string
		tgID     int64
		id       int
		scenario string
		wantErr  bool
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
			data:   "role:8281761514:2",
			action: "role",
			tgID:   8281761514,
			id:     2,
		},
		{
			name:     "group",
			data:     "group:8281761514:5:sv",
			action:   "group",
			tgID:     8281761514,
			id:       5,
			scenario: "sv",
		},
		{
			name:     "territory",
			data:     "territory:8281761514:10:sr",
			action:   "territory",
			tgID:     8281761514,
			id:       10,
			scenario: "sr",
		},
		{
			name:   "code",
			data:   "code:8281761514:25",
			action: "code",
			tgID:   8281761514,
			id:     25,
		},
		{
			name:    "empty data",
			data:    "",
			wantErr: true,
		},
		{
			name:    "invalid telegram id",
			data:    "approve:abc",
			wantErr: true,
		},
		{
			name:    "missing id",
			data:    "role:8281761514",
			wantErr: true,
		},
		{
			name:    "invalid scenario",
			data:    "group:8281761514:5:unknown",
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
			result, err := parseCallbackData(tt.data)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result.Action != tt.action {
				t.Errorf("action = %q, want %q", result.Action, tt.action)
			}

			if result.TgID != tt.tgID {
				t.Errorf("tgID = %d, want %d", result.TgID, tt.tgID)
			}

			if result.ID != tt.id {
				t.Errorf("id = %d, want %d", result.ID, tt.id)
			}

			if result.Scenario != tt.scenario {
				t.Errorf(
					"scenario = %q, want %q",
					result.Scenario,
					tt.scenario,
				)
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
