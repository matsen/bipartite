package board

import "testing"

func TestParseBoardKey(t *testing.T) {
	tests := []struct {
		name       string
		boardKey   string
		wantOwner  string
		wantNumber string
		wantErr    bool
	}{
		{
			name:       "valid org board",
			boardKey:   "matsengrp/30",
			wantOwner:  "matsengrp",
			wantNumber: "30",
			wantErr:    false,
		},
		{
			name:       "valid user board",
			boardKey:   "matsen/5",
			wantOwner:  "matsen",
			wantNumber: "5",
			wantErr:    false,
		},
		{
			name:     "missing slash",
			boardKey: "matsengrp30",
			wantErr:  true,
		},
		{
			name:     "too many slashes",
			boardKey: "matsengrp/boards/30",
			wantErr:  true,
		},
		{
			name:     "empty string",
			boardKey: "",
			wantErr:  true,
		},
		{
			name:     "only slash",
			boardKey: "/",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			owner, number, err := ParseBoardKey(tt.boardKey)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if owner != tt.wantOwner {
				t.Errorf("owner = %q, want %q", owner, tt.wantOwner)
			}
			if number != tt.wantNumber {
				t.Errorf("number = %q, want %q", number, tt.wantNumber)
			}
		})
	}
}
