package store

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseSchema(t *testing.T) {
	// Create a temp schema file
	dir := t.TempDir()
	schemaPath := filepath.Join(dir, "test_schema.json")

	schemaJSON := `{
		"name": "test_store",
		"fields": {
			"id": {"type": "string", "primary": true},
			"name": {"type": "string", "fts": true},
			"count": {"type": "integer", "index": true},
			"active": {"type": "boolean"},
			"score": {"type": "float"},
			"created": {"type": "date"},
			"updated": {"type": "datetime"},
			"metadata": {"type": "json"},
			"status": {"type": "string", "enum": ["pending", "active", "done"]}
		}
	}`

	if err := os.WriteFile(schemaPath, []byte(schemaJSON), 0644); err != nil {
		t.Fatalf("writing schema file: %v", err)
	}

	schema, err := ParseSchema(schemaPath)
	if err != nil {
		t.Fatalf("ParseSchema: %v", err)
	}

	if schema.Name != "test_store" {
		t.Errorf("Name = %q, want %q", schema.Name, "test_store")
	}

	if len(schema.Fields) != 9 {
		t.Errorf("len(Fields) = %d, want 9", len(schema.Fields))
	}

	// Check field types
	if schema.Fields["id"].Type != FieldTypeString {
		t.Errorf("id.Type = %q, want %q", schema.Fields["id"].Type, FieldTypeString)
	}
	if !schema.Fields["id"].Primary {
		t.Error("id.Primary should be true")
	}
	if !schema.Fields["name"].FTS {
		t.Error("name.FTS should be true")
	}
	if !schema.Fields["count"].Index {
		t.Error("count.Index should be true")
	}
	if len(schema.Fields["status"].Enum) != 3 {
		t.Errorf("status.Enum length = %d, want 3", len(schema.Fields["status"].Enum))
	}
}

func TestParseSchema_FileNotFound(t *testing.T) {
	_, err := ParseSchema("/nonexistent/path/schema.json")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestParseSchema_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	schemaPath := filepath.Join(dir, "invalid.json")

	if err := os.WriteFile(schemaPath, []byte("not valid json"), 0644); err != nil {
		t.Fatalf("writing file: %v", err)
	}

	_, err := ParseSchema(schemaPath)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestSchemaValidate(t *testing.T) {
	tests := []struct {
		name    string
		schema  Schema
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid schema",
			schema: Schema{
				Name: "test",
				Fields: map[string]*Field{
					"id": {Type: FieldTypeString, Primary: true},
				},
			},
			wantErr: false,
		},
		{
			name: "missing name",
			schema: Schema{
				Fields: map[string]*Field{
					"id": {Type: FieldTypeString, Primary: true},
				},
			},
			wantErr: true,
			errMsg:  "schema name is required",
		},
		{
			name: "invalid name",
			schema: Schema{
				Name: "123invalid",
				Fields: map[string]*Field{
					"id": {Type: FieldTypeString, Primary: true},
				},
			},
			wantErr: true,
			errMsg:  "not a valid identifier",
		},
		{
			name: "no fields",
			schema: Schema{
				Name:   "test",
				Fields: map[string]*Field{},
			},
			wantErr: true,
			errMsg:  "at least one field",
		},
		{
			name: "no primary key",
			schema: Schema{
				Name: "test",
				Fields: map[string]*Field{
					"name": {Type: FieldTypeString},
				},
			},
			wantErr: true,
			errMsg:  "exactly one primary key",
		},
		{
			name: "multiple primary keys",
			schema: Schema{
				Name: "test",
				Fields: map[string]*Field{
					"id":   {Type: FieldTypeString, Primary: true},
					"uuid": {Type: FieldTypeString, Primary: true},
				},
			},
			wantErr: true,
			errMsg:  "multiple primary keys",
		},
		{
			name: "invalid field type",
			schema: Schema{
				Name: "test",
				Fields: map[string]*Field{
					"id": {Type: "invalid", Primary: true},
				},
			},
			wantErr: true,
			errMsg:  "invalid type",
		},
		{
			name: "fts on non-string",
			schema: Schema{
				Name: "test",
				Fields: map[string]*Field{
					"id":    {Type: FieldTypeString, Primary: true},
					"count": {Type: FieldTypeInteger, FTS: true},
				},
			},
			wantErr: true,
			errMsg:  "fts only valid for string",
		},
		{
			name: "enum on non-string",
			schema: Schema{
				Name: "test",
				Fields: map[string]*Field{
					"id":    {Type: FieldTypeString, Primary: true},
					"count": {Type: FieldTypeInteger, Enum: []string{"a", "b"}},
				},
			},
			wantErr: true,
			errMsg:  "enum only valid for string",
		},
		{
			name: "empty enum value",
			schema: Schema{
				Name: "test",
				Fields: map[string]*Field{
					"id":     {Type: FieldTypeString, Primary: true},
					"status": {Type: FieldTypeString, Enum: []string{"a", ""}},
				},
			},
			wantErr: true,
			errMsg:  "empty enum value",
		},
		{
			name: "invalid field name",
			schema: Schema{
				Name: "test",
				Fields: map[string]*Field{
					"id":       {Type: FieldTypeString, Primary: true},
					"123field": {Type: FieldTypeString},
				},
			},
			wantErr: true,
			errMsg:  "not a valid identifier",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.schema.Validate()
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error containing %q", tt.errMsg)
				} else if tt.errMsg != "" && !contains(err.Error(), tt.errMsg) {
					t.Errorf("error %q should contain %q", err.Error(), tt.errMsg)
				}
			} else if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestSchemaPrimaryKeyField(t *testing.T) {
	schema := &Schema{
		Name: "test",
		Fields: map[string]*Field{
			"id":   {Type: FieldTypeString, Primary: true},
			"name": {Type: FieldTypeString},
		},
	}

	pk := schema.PrimaryKeyField()
	if pk != "id" {
		t.Errorf("PrimaryKeyField() = %q, want %q", pk, "id")
	}
}

func TestSchemaValidateRecord(t *testing.T) {
	schema := &Schema{
		Name: "test",
		Fields: map[string]*Field{
			"id":     {Type: FieldTypeString, Primary: true},
			"name":   {Type: FieldTypeString},
			"count":  {Type: FieldTypeInteger},
			"score":  {Type: FieldTypeFloat},
			"active": {Type: FieldTypeBoolean},
			"date":   {Type: FieldTypeDate},
			"status": {Type: FieldTypeString, Enum: []string{"pending", "active"}},
		},
	}

	tests := []struct {
		name    string
		record  Record
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid record",
			record: Record{
				"id":     "123",
				"name":   "test",
				"count":  float64(42), // JSON numbers are float64
				"score":  3.14,
				"active": true,
				"date":   "2026-01-27",
				"status": "pending",
			},
			wantErr: false,
		},
		{
			name: "missing primary key",
			record: Record{
				"name": "test",
			},
			wantErr: true,
			errMsg:  "missing primary key",
		},
		{
			name: "null primary key",
			record: Record{
				"id": nil,
			},
			wantErr: true,
			errMsg:  "primary key field",
		},
		{
			name: "invalid string type",
			record: Record{
				"id":   "123",
				"name": 42,
			},
			wantErr: true,
			errMsg:  "expected string",
		},
		{
			name: "invalid integer type",
			record: Record{
				"id":    "123",
				"count": "not a number",
			},
			wantErr: true,
			errMsg:  "expected integer",
		},
		{
			name: "float where integer expected",
			record: Record{
				"id":    "123",
				"count": 3.14, // not an integer
			},
			wantErr: true,
			errMsg:  "expected integer",
		},
		{
			name: "invalid boolean type",
			record: Record{
				"id":     "123",
				"active": "yes",
			},
			wantErr: true,
			errMsg:  "expected boolean",
		},
		{
			name: "invalid enum value",
			record: Record{
				"id":     "123",
				"status": "unknown",
			},
			wantErr: true,
			errMsg:  "not in enum",
		},
		{
			name: "extra fields allowed",
			record: Record{
				"id":          "123",
				"extra_field": "allowed",
			},
			wantErr: false,
		},
		{
			name: "null non-primary field allowed",
			record: Record{
				"id":   "123",
				"name": nil,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := schema.ValidateRecord(tt.record)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error containing %q", tt.errMsg)
				} else if tt.errMsg != "" && !contains(err.Error(), tt.errMsg) {
					t.Errorf("error %q should contain %q", err.Error(), tt.errMsg)
				}
			} else if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

// contains checks if s contains substr (case-insensitive would be better but this is simpler)
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && containsAt(s, substr)))
}

func containsAt(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
