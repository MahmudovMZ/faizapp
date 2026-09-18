package telegram

import "testing"

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
