package storage

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/matsen/bipartite/internal/project"
)

// ReadAllProjects reads all projects from a JSONL file.
// Returns an error if any project fails structural validation (fail-fast).
func ReadAllProjects(path string) ([]project.Project, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // Empty file returns empty slice
		}
		return nil, fmt.Errorf("opening projects file: %w", err)
	}
	defer f.Close()

	var projects []project.Project
	scanner := bufio.NewScanner(f)

	// Increase buffer size for long lines
	buf := make([]byte, MaxJSONLLineCapacity)
	scanner.Buffer(buf, MaxJSONLLineCapacity)

	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Bytes()
		if len(line) == 0 {
			continue // Skip empty lines
		}

		var p project.Project
		if err := json.Unmarshal(line, &p); err != nil {
			return nil, fmt.Errorf("parsing line %d: %w", lineNum, err)
		}

		// Fail fast: validate project structure before adding to collection
		if err := p.ValidateForCreate(); err != nil {
			return nil, fmt.Errorf("invalid project at line %d: %w", lineNum, err)
		}

		projects = append(projects, p)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading projects file: %w", err)
	}

	return projects, nil
}

// writeProjectJSONL marshals a project to JSON and writes it as a JSONL line.
func writeProjectJSONL(w io.Writer, p project.Project) error {
	data, err := json.Marshal(p)
	if err != nil {
		return fmt.Errorf("encoding project: %w", err)
	}
	if _, err := w.Write(data); err != nil {
		return fmt.Errorf("writing project: %w", err)
	}
	if _, err := w.Write([]byte("\n")); err != nil {
		return fmt.Errorf("writing newline: %w", err)
	}
	return nil
}

// AppendProject adds a project to the end of a JSONL file.
func AppendProject(path string, p project.Project) error {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("opening projects file for append: %w", err)
	}
	defer f.Close()

	return writeProjectJSONL(f, p)
}

// WriteAllProjects writes all projects to a JSONL file, replacing existing content.
func WriteAllProjects(path string, projects []project.Project) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("creating projects file: %w", err)
	}
	defer f.Close()

	for _, p := range projects {
		if err := writeProjectJSONL(f, p); err != nil {
			return err
		}
	}

	return nil
}

// FindProjectByID searches for a project by its ID in an in-memory slice.
// Returns the index and true if found, -1 and false otherwise.
func FindProjectByID(projects []project.Project, id string) (int, bool) {
	for i, p := range projects {
		if p.ID == id {
			return i, true
		}
	}
	return -1, false
}

// UpsertProjectInSlice adds or updates a project in an in-memory slice.
// Returns the updated slice and true if the project was updated, false if added.
func UpsertProjectInSlice(projects []project.Project, newProject project.Project) ([]project.Project, bool) {
	idx, found := FindProjectByID(projects, newProject.ID)
	if found {
		projects[idx] = newProject
		return projects, true
	}
	return append(projects, newProject), false
}

// DeleteProjectFromSlice removes a project from an in-memory slice.
// Returns the updated slice and true if the project was found and removed, false otherwise.
// Note: This operation does not preserve slice order; it swaps the deleted element with
// the last element for O(1) performance. If ordering matters, callers should sort after deletion.
func DeleteProjectFromSlice(projects []project.Project, id string) ([]project.Project, bool) {
	idx, found := FindProjectByID(projects, id)
	if !found {
		return projects, false
	}
	// Remove by replacing with last element and truncating (O(1) but changes order)
	projects[idx] = projects[len(projects)-1]
	return projects[:len(projects)-1], true
}

// LoadProjectIDSet loads all project IDs and returns them as a set for O(1) lookup.
func LoadProjectIDSet(path string) (map[string]bool, error) {
	projects, err := ReadAllProjects(path)
	if err != nil {
		return nil, err
	}

	idSet := make(map[string]bool, len(projects))
	for _, p := range projects {
		idSet[p.ID] = true
	}
	return idSet, nil
}
