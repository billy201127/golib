package db

import (
	"database/sql/driver"
	"testing"
)

func TestBuildCompleteSQL(t *testing.T) {
	tests := []struct {
		name     string
		query    string
		args     []driver.NamedValue
		expected string
	}{
		{
			name:     "No args",
			query:    "SELECT * FROM users",
			args:     []driver.NamedValue{},
			expected: "SELECT * FROM users",
		},
		{
			name:     "String parameter",
			query:    "SELECT * FROM users WHERE name = ?",
			args:     []driver.NamedValue{{Value: "John"}},
			expected: "SELECT * FROM users WHERE name = 'John'",
		},
		{
			name:     "Multiple parameters",
			query:    "SELECT * FROM users WHERE age > ? AND status = ?",
			args:     []driver.NamedValue{{Value: 18}, {Value: "active"}},
			expected: "SELECT * FROM users WHERE age > 18 AND status = 'active'",
		},
		{
			name:     "Null parameter",
			query:    "SELECT * FROM users WHERE deleted_at = ?",
			args:     []driver.NamedValue{{Value: nil}},
			expected: "SELECT * FROM users WHERE deleted_at = NULL",
		},
		{
			name:     "String with single quote",
			query:    "SELECT * FROM users WHERE name = ?",
			args:     []driver.NamedValue{{Value: "O'Connor"}},
			expected: "SELECT * FROM users WHERE name = 'O''Connor'",
		},
		{
			name:     "Integer parameter",
			query:    "SELECT * FROM users WHERE id = ?",
			args:     []driver.NamedValue{{Value: 123}},
			expected: "SELECT * FROM users WHERE id = 123",
		},
		{
			name:     "Float parameter",
			query:    "SELECT * FROM products WHERE price > ?",
			args:     []driver.NamedValue{{Value: 99.99}},
			expected: "SELECT * FROM products WHERE price > 99.99",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildCompleteSQL(tt.query, tt.args)
			if result != tt.expected {
				t.Errorf("buildCompleteSQL() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestBuildCompleteSQLMismatch(t *testing.T) {
	// Test case where placeholder count doesn't match argument count
	query := "SELECT * FROM users WHERE id = ? AND name = ?"
	args := []driver.NamedValue{{Value: 1}} // Only one arg for two placeholders

	result := buildCompleteSQL(query, args)
	if result != query {
		t.Errorf("buildCompleteSQL() should return original query when placeholder count doesn't match, got %v", result)
	}
}
