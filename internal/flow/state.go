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

// ReadLastCheckin reads the last checkin timestamp from StateFile in the given nexus directory.
// Returns zero time if the file doesn't exist or can't be parsed.
func ReadLastCheckin(nexusPath string) time.Time {
	data, err := os.ReadFile(StatePath(nexusPath))
	if err != nil {
		return time.Time{}
	}

	var state checkinState
	if err := json.Unmarshal(data, &state); err != nil {
		return time.Time{}
	}

	return state.LastCheckin
}

// WriteLastCheckin writes the given timestamp to StateFile in the given nexus directory.
func WriteLastCheckin(nexusPath string, t time.Time) error {
	state := checkinState{LastCheckin: t}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling state: %w", err)
	}
	data = append(data, '\n')

	statePath := StatePath(nexusPath)
	if err := os.WriteFile(statePath, data, 0644); err != nil {
		return fmt.Errorf("writing %s: %w", statePath, err)
	}

	return nil
}
