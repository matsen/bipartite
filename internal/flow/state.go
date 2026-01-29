package flow

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// checkinState represents the contents of .last-checkin.json.
type checkinState struct {
	LastCheckin time.Time `json:"last_checkin"`
}

// ReadLastCheckin reads the last checkin timestamp from StateFile.
// Returns zero time if the file doesn't exist or can't be parsed.
func ReadLastCheckin() time.Time {
	data, err := os.ReadFile(StateFile)
	if err != nil {
		return time.Time{}
	}

	var state checkinState
	if err := json.Unmarshal(data, &state); err != nil {
		return time.Time{}
	}

	return state.LastCheckin
}

// WriteLastCheckin writes the given timestamp to StateFile.
func WriteLastCheckin(t time.Time) error {
	state := checkinState{LastCheckin: t}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling state: %w", err)
	}
	data = append(data, '\n')

	if err := os.WriteFile(StateFile, data, 0644); err != nil {
		return fmt.Errorf("writing %s: %w", StateFile, err)
	}

	return nil
}
