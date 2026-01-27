// Package store provides a generic JSONL + SQLite store abstraction.
package store

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
)

// FieldType represents the data type of a field.
type FieldType string

const (
	FieldTypeString   FieldType = "string"
	FieldTypeInteger  FieldType = "integer"
	FieldTypeFloat    FieldType = "float"
	FieldTypeBoolean  FieldType = "boolean"
	FieldTypeDate     FieldType = "date"     // ISO 8601: YYYY-MM-DD
	FieldTypeDatetime FieldType = "datetime" // ISO 8601: YYYY-MM-DDTHH:MM:SSZ
	FieldTypeJSON     FieldType = "json"     // Stored as TEXT, queryable via JSON functions
)

// validFieldTypes is the set of recognized field types.
var validFieldTypes = map[FieldType]bool{
	FieldTypeString:   true,
	FieldTypeInteger:  true,
	FieldTypeFloat:    true,
	FieldTypeBoolean:  true,
	FieldTypeDate:     true,
	FieldTypeDatetime: true,
	FieldTypeJSON:     true,
}

// validIdentifier matches valid SQLite identifiers (alphanumeric + underscore, must start with letter or underscore).
var validIdentifier = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

// Field defines a single field in a schema.
type Field struct {
	Type    FieldType `json:"type"`
	Primary bool      `json:"primary,omitempty"`
	Index   bool      `json:"index,omitempty"`
	FTS     bool      `json:"fts,omitempty"`
	Enum    []string  `json:"enum,omitempty"`
}

// Schema defines the structure of a store.
type Schema struct {
	Name   string            `json:"name"`
	Fields map[string]*Field `json:"fields"`
}

// ParseSchema loads and parses a JSON schema file.
func ParseSchema(path string) (*Schema, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading schema file: %w", err)
	}

	var schema Schema
	if err := json.Unmarshal(data, &schema); err != nil {
		return nil, fmt.Errorf("parsing schema JSON: %w", err)
	}

	return &schema, nil
}

// Validate checks that the schema is valid.
// It returns an error describing any validation failures.
func (s *Schema) Validate() error {
	if s.Name == "" {
		return fmt.Errorf("schema name is required")
	}

	if !validIdentifier.MatchString(s.Name) {
		return fmt.Errorf("schema name %q is not a valid identifier", s.Name)
	}

	if len(s.Fields) == 0 {
		return fmt.Errorf("schema must have at least one field")
	}

	var primaryFields []string
	for name, field := range s.Fields {
		if !validIdentifier.MatchString(name) {
			return fmt.Errorf("field name %q is not a valid identifier", name)
		}

		if !validFieldTypes[field.Type] {
			return fmt.Errorf("field %q has invalid type %q", name, field.Type)
		}

		if field.Primary {
			primaryFields = append(primaryFields, name)
		}

		// FTS only valid for string fields
		if field.FTS && field.Type != FieldTypeString {
			return fmt.Errorf("field %q has fts:true but type %q (fts only valid for string)", name, field.Type)
		}

		// Validate enum values
		if len(field.Enum) > 0 {
			if field.Type != FieldTypeString {
				return fmt.Errorf("field %q has enum but type %q (enum only valid for string)", name, field.Type)
			}
			for _, v := range field.Enum {
				if v == "" {
					return fmt.Errorf("field %q has empty enum value", name)
				}
			}
		}
	}

	// Check for exactly one primary key
	if len(primaryFields) == 0 {
		return fmt.Errorf("schema must have exactly one primary key field")
	}
	if len(primaryFields) > 1 {
		return fmt.Errorf("schema has multiple primary keys: %s", strings.Join(primaryFields, ", "))
	}

	return nil
}

// PrimaryKeyField returns the name of the primary key field.
// It panics if the schema is invalid (no primary key).
func (s *Schema) PrimaryKeyField() string {
	for name, field := range s.Fields {
		if field.Primary {
			return name
		}
	}
	panic("schema has no primary key field")
}

// ValidateRecord validates a record against this schema.
func (s *Schema) ValidateRecord(record Record) error {
	pkField := s.PrimaryKeyField()

	// Check primary key is present
	pkValue, ok := record[pkField]
	if !ok {
		return fmt.Errorf("missing primary key field %q", pkField)
	}
	if pkValue == nil {
		return fmt.Errorf("primary key field %q is null", pkField)
	}

	// Validate each field in the record
	for name, value := range record {
		field, ok := s.Fields[name]
		if !ok {
			// Extra fields are allowed (flexible schema)
			continue
		}

		if value == nil {
			// Null values are allowed for non-primary fields
			continue
		}

		if err := s.validateFieldValue(name, field, value); err != nil {
			return err
		}
	}

	return nil
}

// validateFieldValue validates a single field value.
func (s *Schema) validateFieldValue(name string, field *Field, value any) error {
	switch field.Type {
	case FieldTypeString:
		str, ok := value.(string)
		if !ok {
			return fmt.Errorf("field %q: expected string, got %T", name, value)
		}
		// Check enum constraint
		if len(field.Enum) > 0 {
			valid := false
			for _, e := range field.Enum {
				if str == e {
					valid = true
					break
				}
			}
			if !valid {
				return fmt.Errorf("field %q: value %q not in enum %v", name, str, field.Enum)
			}
		}

	case FieldTypeInteger:
		switch v := value.(type) {
		case float64:
			// JSON numbers are float64, check if it's actually an integer
			if v != float64(int64(v)) {
				return fmt.Errorf("field %q: expected integer, got float %v", name, v)
			}
		case int, int64:
			// OK
		default:
			return fmt.Errorf("field %q: expected integer, got %T", name, value)
		}

	case FieldTypeFloat:
		switch value.(type) {
		case float64, float32, int, int64:
			// OK (int can coerce to float)
		default:
			return fmt.Errorf("field %q: expected float, got %T", name, value)
		}

	case FieldTypeBoolean:
		if _, ok := value.(bool); !ok {
			return fmt.Errorf("field %q: expected boolean, got %T", name, value)
		}

	case FieldTypeDate, FieldTypeDatetime:
		if _, ok := value.(string); !ok {
			return fmt.Errorf("field %q: expected string (ISO date), got %T", name, value)
		}
		// TODO: Validate date format more strictly if needed

	case FieldTypeJSON:
		// Any JSON value is valid
	}

	return nil
}

// Record represents a single record in a store.
// Stored as map[string]any since schema is dynamic.
type Record map[string]any
