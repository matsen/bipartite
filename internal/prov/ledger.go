// Package prov checks a manuscript's provenance ledger (provenance.yaml)
// against the %PROV[id] tags in its TeX source and the git objects of the
// code repos the ledger points to.
package prov

import (
	"bytes"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Ledger is the parsed provenance.yaml.
type Ledger struct {
	AuditedThrough string           `yaml:"audited_through"`
	Runs           map[string]Run   `yaml:"runs"`
	Entries        map[string]Entry `yaml:"entries"`
}

// Run is something that produced a number or is described by a code claim.
type Run struct {
	Facts       *Pin     `yaml:"facts"`
	Launches    []string `yaml:"launches"`
	CommittedAt string   `yaml:"committed_at"`
}

// Pin names a file at a commit of a repo.
type Pin struct {
	Repo string `yaml:"repo"`
	SHA  string `yaml:"sha"`
	Path string `yaml:"path"`
}

// Entry is one sourced claim. It has exactly one extractor: Key, Pattern,
// Absent, Blob, From, or Unsourced.
type Entry struct {
	Run       string   `yaml:"run"`
	Repo      string   `yaml:"repo"`
	SHA       string   `yaml:"sha"`
	Path      string   `yaml:"path"`
	Key       string   `yaml:"key"`
	Value     any      `yaml:"value"`
	Pattern   string   `yaml:"pattern"`
	Absent    string   `yaml:"absent"`
	Blob      string   `yaml:"blob"`
	Unsourced string   `yaml:"unsourced"`
	From      []string `yaml:"from"` // ids this derived value is computed from
	Token     string   `yaml:"token"`
	Scope     string   `yaml:"scope"`
}

// LoadLedger reads a ledger, rejecting unknown fields and duplicate ids.
func LoadLedger(path string) (*Ledger, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	var l Ledger
	if err := dec.Decode(&l); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return &l, nil
}
