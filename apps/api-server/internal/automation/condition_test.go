package automation

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

// ---------------------------------------------------------------------------
// resolveField
// ---------------------------------------------------------------------------

func TestResolveField(t *testing.T) {
	tests := []struct {
		name     string
		field    string
		data     map[string]any
		expected any
	}{
		{
			name:     "top-level key",
			field:    "status",
			data:     map[string]any{"status": "new"},
			expected: "new",
		},
		{
			name:  "two-level dotted path",
			field: "order.status",
			data: map[string]any{
				"order": map[string]any{"status": "confirmed"},
			},
			expected: "confirmed",
		},
		{
			name:  "three-level deeply nested",
			field: "a.b.c",
			data: map[string]any{
				"a": map[string]any{
					"b": map[string]any{
						"c": 42,
					},
				},
			},
			expected: 42,
		},
		{
			name:  "four-level deeply nested",
			field: "a.b.c.d",
			data: map[string]any{
				"a": map[string]any{
					"b": map[string]any{
						"c": map[string]any{
							"d": "deep",
						},
					},
				},
			},
			expected: "deep",
		},
		{
			name:     "missing top-level key returns nil",
			field:    "missing",
			data:     map[string]any{"status": "new"},
			expected: nil,
		},
		{
			name:  "missing nested key returns nil",
			field: "order.missing",
			data: map[string]any{
				"order": map[string]any{"status": "new"},
			},
			expected: nil,
		},
		{
			name:  "path through non-map value returns nil",
			field: "order.status.deeper",
			data: map[string]any{
				"order": map[string]any{"status": "new"},
			},
			expected: nil,
		},
		{
			name:     "nil map returns nil",
			field:    "anything",
			data:     nil,
			expected: nil,
		},
		{
			name:     "empty map returns nil",
			field:    "status",
			data:     map[string]any{},
			expected: nil,
		},
		{
			name:  "path through slice value returns nil (not a map)",
			field: "items.0.name",
			data: map[string]any{
				"items": []any{"a", "b"},
			},
			expected: nil,
		},
		{
			name:  "path through integer value returns nil",
			field: "count.sub",
			data: map[string]any{
				"count": 10,
			},
			expected: nil,
		},
		{
			name:  "value is a nested map (not a leaf scalar)",
			field: "order",
			data: map[string]any{
				"order": map[string]any{"status": "new"},
			},
			expected: map[string]any{"status": "new"},
		},
		{
			name:     "empty field string resolves from root",
			field:    "",
			data:     map[string]any{"status": "new"},
			expected: nil,
		},
		{
			name:  "field value is nil explicitly",
			field: "status",
			data: map[string]any{
				"status": nil,
			},
			expected: nil,
		},
		{
			name:  "field value is boolean",
			field: "active",
			data: map[string]any{
				"active": true,
			},
			expected: true,
		},
		{
			name:  "field value is float64",
			field: "amount",
			data: map[string]any{
				"amount": 99.99,
			},
			expected: 99.99,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := resolveField(tt.field, tt.data)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// ---------------------------------------------------------------------------
// compareEqual
// ---------------------------------------------------------------------------

func TestCompareEqual(t *testing.T) {
	tests := []struct {
		name     string
		a        any
		b        any
		expected bool
	}{
		{"both nil", nil, nil, true},
		{"a nil b non-nil", nil, "hello", false},
		{"a non-nil b nil", "hello", nil, false},
		{"same strings", "abc", "abc", true},
		{"different strings", "abc", "xyz", false},
		{"int vs string same value", 42, "42", true},
		{"float64 vs string same value", 42.0, "42", true},
		{"float64 vs int same value", 42.0, 42, true},
		{"bool true vs string", true, "true", true},
		{"bool false vs string", false, "false", true},
		{"empty strings", "", "", true},
		{"string vs empty", "abc", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := compareEqual(tt.a, tt.b)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// ---------------------------------------------------------------------------
// toFloat64
// ---------------------------------------------------------------------------

func TestToFloat64_Extended(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected float64
	}{
		{"nil", nil, 0},
		{"float64", 42.5, 42.5},
		{"float64 zero", 0.0, 0},
		{"float64 negative", -10.5, -10.5},
		{"float32", float32(3.14), 3.140000104904175}, // float32 precision
		{"int", 42, 42.0},
		{"int zero", 0, 0},
		{"int negative", -7, -7.0},
		{"int64", int64(1000000), 1000000.0},
		{"int64 negative", int64(-999), -999.0},
		{"int32", int32(32), 32.0},
		{"string integer", "100", 100.0},
		{"string float", "3.14", 3.14},
		{"string negative", "-5.5", -5.5},
		{"string zero", "0", 0},
		{"string non-numeric", "abc", 0},
		{"string empty", "", 0},
		{"bool true (via default Sprintf)", true, 0},
		{"bool false (via default Sprintf)", false, 0},
		{"json.Number integer", json.Number("42"), 42.0},
		{"json.Number float", json.Number("3.14"), 3.14},
		{"json.Number negative", json.Number("-100"), -100.0},
		{"uint via default branch", uint(55), 55.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := toFloat64(tt.input)
			assert.InDelta(t, tt.expected, result, 0.001, "toFloat64(%v)", tt.input)
		})
	}
}

// ---------------------------------------------------------------------------
// compareNumeric
// ---------------------------------------------------------------------------

func TestCompareNumeric(t *testing.T) {
	tests := []struct {
		name     string
		a        any
		b        any
		expected int
	}{
		{"equal floats", 10.0, 10.0, 0},
		{"a greater float", 20.0, 10.0, 1},
		{"a less float", 5.0, 10.0, -1},
		{"equal ints", 7, 7, 0},
		{"int vs float equal", 10, 10.0, 0},
		{"string numbers equal", "100", "100.0", 0},
		{"string number vs float", "50.5", 50.5, 0},
		{"string number a > b", "200", 100.0, 1},
		{"string number a < b", "10", 100.0, -1},
		{"nil vs zero", nil, 0, 0},
		{"nil vs positive", nil, 5.0, -1},
		{"positive vs nil", 5.0, nil, 1},
		{"negative vs positive", -1.0, 1.0, -1},
		{"both nil", nil, nil, 0},
		{"mixed int64 and float64", int64(42), 42.0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := compareNumeric(tt.a, tt.b)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// ---------------------------------------------------------------------------
// compareContains
// ---------------------------------------------------------------------------

func TestCompareContains(t *testing.T) {
	tests := []struct {
		name     string
		field    any
		search   any
		expected bool
	}{
		{"substring found", "Jan Kowalski", "Jan", true},
		{"substring not found", "Anna Nowak", "Jan", false},
		{"exact match", "hello", "hello", true},
		{"empty search in non-empty field", "hello", "", true},
		{"empty field with empty search (nil field)", nil, "", false},
		{"nil field value", nil, "test", false},
		{"case sensitive mismatch", "Hello World", "hello", false},
		{"case sensitive match", "Hello World", "Hello", true},
		{"numeric field contains digit", 12345, "234", true},
		{"numeric field does not contain", 12345, "999", false},
		{"search at beginning", "abcdef", "abc", true},
		{"search at end", "abcdef", "def", true},
		{"search in middle", "abcdef", "cde", true},
		{"field is bool", true, "true", true},
		{"field is bool false", false, "fals", true},
		{"field with special characters", "hello@world.com", "@world", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := compareContains(tt.field, tt.search)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// ---------------------------------------------------------------------------
// compareStartsWith
// ---------------------------------------------------------------------------

func TestCompareStartsWith(t *testing.T) {
	tests := []struct {
		name     string
		field    any
		prefix   any
		expected bool
	}{
		{"prefix match", "Hello World", "Hello", true},
		{"prefix mismatch", "Hello World", "World", false},
		{"exact match", "Hello", "Hello", true},
		{"empty prefix always matches", "Hello", "", true},
		{"nil field value", nil, "test", false},
		{"case sensitive mismatch", "Hello", "hello", false},
		{"numeric field starts with", 12345, "123", true},
		{"numeric field does not start with", 12345, "234", false},
		{"prefix longer than field", "Hi", "Hello", false},
		{"empty field with empty prefix (nil)", nil, "", false},
		{"field is bool true", true, "tr", true},
		{"field with spaces", "  leading", "  ", true},
		{"field with special characters", "http://example.com", "http://", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := compareStartsWith(tt.field, tt.prefix)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// ---------------------------------------------------------------------------
// compareIn
// ---------------------------------------------------------------------------

func TestCompareIn(t *testing.T) {
	tests := []struct {
		name      string
		field     any
		listValue any
		expected  bool
	}{
		{"value in slice", "new", []any{"new", "confirmed"}, true},
		{"value not in slice", "shipped", []any{"new", "confirmed"}, false},
		{"nil field always false", nil, []any{"new"}, false},
		{"empty slice", "new", []any{}, false},
		{"non-slice falls back to compareEqual", "abc", "abc", true},
		{"non-slice mismatch", "abc", "xyz", false},
		{"numeric in string slice", 42, []any{"42", "43"}, true},
		{"mixed types in slice", "100", []any{100, "200"}, true},
		{"single element slice match", "x", []any{"x"}, true},
		{"single element slice mismatch", "x", []any{"y"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := compareIn(tt.field, tt.listValue)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// ---------------------------------------------------------------------------
// EvaluateConditions — comprehensive integration tests
// ---------------------------------------------------------------------------

func TestEvaluateConditions_Comprehensive(t *testing.T) {
	tests := []struct {
		name       string
		conditions []Condition
		data       map[string]any
		expected   bool
	}{
		// Empty / nil conditions
		{
			name:       "nil conditions returns true",
			conditions: nil,
			data:       map[string]any{"status": "new"},
			expected:   true,
		},
		{
			name:       "empty slice conditions returns true",
			conditions: []Condition{},
			data:       map[string]any{"status": "new"},
			expected:   true,
		},

		// eq operator
		{
			name:       "eq match",
			conditions: []Condition{{Field: "status", Operator: "eq", Value: "new"}},
			data:       map[string]any{"status": "new"},
			expected:   true,
		},
		{
			name:       "eq mismatch",
			conditions: []Condition{{Field: "status", Operator: "eq", Value: "new"}},
			data:       map[string]any{"status": "shipped"},
			expected:   false,
		},
		{
			name:       "eq with nil field",
			conditions: []Condition{{Field: "missing", Operator: "eq", Value: "test"}},
			data:       map[string]any{},
			expected:   false,
		},
		{
			name:       "eq nil field nil value",
			conditions: []Condition{{Field: "missing", Operator: "eq", Value: nil}},
			data:       map[string]any{},
			expected:   true,
		},

		// neq operator
		{
			name:       "neq different values",
			conditions: []Condition{{Field: "status", Operator: "neq", Value: "cancelled"}},
			data:       map[string]any{"status": "new"},
			expected:   true,
		},
		{
			name:       "neq same values",
			conditions: []Condition{{Field: "status", Operator: "neq", Value: "new"}},
			data:       map[string]any{"status": "new"},
			expected:   false,
		},

		// gt operator
		{
			name:       "gt true",
			conditions: []Condition{{Field: "amount", Operator: "gt", Value: 100.0}},
			data:       map[string]any{"amount": 200.0},
			expected:   true,
		},
		{
			name:       "gt equal is false",
			conditions: []Condition{{Field: "amount", Operator: "gt", Value: 100.0}},
			data:       map[string]any{"amount": 100.0},
			expected:   false,
		},
		{
			name:       "gt less is false",
			conditions: []Condition{{Field: "amount", Operator: "gt", Value: 100.0}},
			data:       map[string]any{"amount": 50.0},
			expected:   false,
		},
		{
			name:       "gt with string numbers",
			conditions: []Condition{{Field: "amount", Operator: "gt", Value: "100"}},
			data:       map[string]any{"amount": "200"},
			expected:   true,
		},

		// gte operator
		{
			name:       "gte equal",
			conditions: []Condition{{Field: "amount", Operator: "gte", Value: 100.0}},
			data:       map[string]any{"amount": 100.0},
			expected:   true,
		},
		{
			name:       "gte greater",
			conditions: []Condition{{Field: "amount", Operator: "gte", Value: 100.0}},
			data:       map[string]any{"amount": 101.0},
			expected:   true,
		},
		{
			name:       "gte less",
			conditions: []Condition{{Field: "amount", Operator: "gte", Value: 100.0}},
			data:       map[string]any{"amount": 99.0},
			expected:   false,
		},

		// lt operator
		{
			name:       "lt true",
			conditions: []Condition{{Field: "amount", Operator: "lt", Value: 100.0}},
			data:       map[string]any{"amount": 50.0},
			expected:   true,
		},
		{
			name:       "lt equal is false",
			conditions: []Condition{{Field: "amount", Operator: "lt", Value: 100.0}},
			data:       map[string]any{"amount": 100.0},
			expected:   false,
		},
		{
			name:       "lt greater is false",
			conditions: []Condition{{Field: "amount", Operator: "lt", Value: 100.0}},
			data:       map[string]any{"amount": 200.0},
			expected:   false,
		},

		// lte operator
		{
			name:       "lte equal",
			conditions: []Condition{{Field: "amount", Operator: "lte", Value: 100.0}},
			data:       map[string]any{"amount": 100.0},
			expected:   true,
		},
		{
			name:       "lte less",
			conditions: []Condition{{Field: "amount", Operator: "lte", Value: 100.0}},
			data:       map[string]any{"amount": 50.0},
			expected:   true,
		},
		{
			name:       "lte greater",
			conditions: []Condition{{Field: "amount", Operator: "lte", Value: 100.0}},
			data:       map[string]any{"amount": 101.0},
			expected:   false,
		},

		// contains operator
		{
			name:       "contains match",
			conditions: []Condition{{Field: "name", Operator: "contains", Value: "Jan"}},
			data:       map[string]any{"name": "Jan Kowalski"},
			expected:   true,
		},
		{
			name:       "contains mismatch",
			conditions: []Condition{{Field: "name", Operator: "contains", Value: "Anna"}},
			data:       map[string]any{"name": "Jan Kowalski"},
			expected:   false,
		},
		{
			name:       "contains nil field",
			conditions: []Condition{{Field: "name", Operator: "contains", Value: "test"}},
			data:       map[string]any{},
			expected:   false,
		},

		// not_contains operator
		{
			name:       "not_contains true",
			conditions: []Condition{{Field: "name", Operator: "not_contains", Value: "Anna"}},
			data:       map[string]any{"name": "Jan Kowalski"},
			expected:   true,
		},
		{
			name:       "not_contains false",
			conditions: []Condition{{Field: "name", Operator: "not_contains", Value: "Jan"}},
			data:       map[string]any{"name": "Jan Kowalski"},
			expected:   false,
		},

		// in operator
		{
			name:       "in match",
			conditions: []Condition{{Field: "status", Operator: "in", Value: []any{"new", "confirmed"}}},
			data:       map[string]any{"status": "new"},
			expected:   true,
		},
		{
			name:       "in mismatch",
			conditions: []Condition{{Field: "status", Operator: "in", Value: []any{"new", "confirmed"}}},
			data:       map[string]any{"status": "shipped"},
			expected:   false,
		},
		{
			name:       "in with nil field",
			conditions: []Condition{{Field: "status", Operator: "in", Value: []any{"new"}}},
			data:       map[string]any{},
			expected:   false,
		},

		// not_in operator
		{
			name:       "not_in true",
			conditions: []Condition{{Field: "status", Operator: "not_in", Value: []any{"cancelled", "refunded"}}},
			data:       map[string]any{"status": "new"},
			expected:   true,
		},
		{
			name:       "not_in false",
			conditions: []Condition{{Field: "status", Operator: "not_in", Value: []any{"cancelled", "refunded"}}},
			data:       map[string]any{"status": "cancelled"},
			expected:   false,
		},

		// starts_with operator
		{
			name:       "starts_with match",
			conditions: []Condition{{Field: "sku", Operator: "starts_with", Value: "PRD-"}},
			data:       map[string]any{"sku": "PRD-001"},
			expected:   true,
		},
		{
			name:       "starts_with mismatch",
			conditions: []Condition{{Field: "sku", Operator: "starts_with", Value: "PRD-"}},
			data:       map[string]any{"sku": "SVC-001"},
			expected:   false,
		},
		{
			name:       "starts_with nil field",
			conditions: []Condition{{Field: "sku", Operator: "starts_with", Value: "PRD-"}},
			data:       map[string]any{},
			expected:   false,
		},

		// Unknown operator
		{
			name:       "unknown operator returns false",
			conditions: []Condition{{Field: "status", Operator: "regex", Value: ".*"}},
			data:       map[string]any{"status": "new"},
			expected:   false,
		},

		// Multiple conditions (AND logic)
		{
			name: "multiple conditions all true",
			conditions: []Condition{
				{Field: "status", Operator: "eq", Value: "confirmed"},
				{Field: "total_amount", Operator: "gt", Value: 100.0},
				{Field: "source", Operator: "in", Value: []any{"allegro", "amazon"}},
			},
			data: map[string]any{
				"status":       "confirmed",
				"total_amount": 250.0,
				"source":       "allegro",
			},
			expected: true,
		},
		{
			name: "multiple conditions first false short-circuits",
			conditions: []Condition{
				{Field: "status", Operator: "eq", Value: "cancelled"},
				{Field: "total_amount", Operator: "gt", Value: 100.0},
			},
			data: map[string]any{
				"status":       "new",
				"total_amount": 250.0,
			},
			expected: false,
		},
		{
			name: "multiple conditions last false",
			conditions: []Condition{
				{Field: "status", Operator: "eq", Value: "confirmed"},
				{Field: "total_amount", Operator: "gt", Value: 100.0},
				{Field: "name", Operator: "contains", Value: "URGENT"},
			},
			data: map[string]any{
				"status":       "confirmed",
				"total_amount": 250.0,
				"name":         "Regular order",
			},
			expected: false,
		},

		// Dotted field paths in EvaluateConditions
		{
			name:       "dotted field path eq",
			conditions: []Condition{{Field: "order.status", Operator: "eq", Value: "new"}},
			data: map[string]any{
				"order": map[string]any{"status": "new"},
			},
			expected: true,
		},
		{
			name:       "deeply nested dotted path gt",
			conditions: []Condition{{Field: "order.customer.total_spent", Operator: "gt", Value: 1000.0}},
			data: map[string]any{
				"order": map[string]any{
					"customer": map[string]any{
						"total_spent": 1500.0,
					},
				},
			},
			expected: true,
		},
		{
			name:       "dotted path missing intermediate returns false for eq",
			conditions: []Condition{{Field: "order.nonexistent.status", Operator: "eq", Value: "new"}},
			data: map[string]any{
				"order": map[string]any{"status": "new"},
			},
			expected: false,
		},

		// Mixed numeric types
		{
			name:       "int field vs float64 value gt",
			conditions: []Condition{{Field: "qty", Operator: "gt", Value: 5.0}},
			data:       map[string]any{"qty": 10},
			expected:   true,
		},
		{
			name:       "string number field vs float value gte",
			conditions: []Condition{{Field: "qty", Operator: "gte", Value: 10.0}},
			data:       map[string]any{"qty": "10"},
			expected:   true,
		},

		// Edge: all operators with nil data field
		{
			name:       "nil field gt is false (0 > 5)",
			conditions: []Condition{{Field: "x", Operator: "gt", Value: 5.0}},
			data:       map[string]any{},
			expected:   false,
		},
		{
			name:       "nil field lt is true (0 < 5)",
			conditions: []Condition{{Field: "x", Operator: "lt", Value: 5.0}},
			data:       map[string]any{},
			expected:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := EvaluateConditions(tt.conditions, tt.data)
			assert.Equal(t, tt.expected, result)
		})
	}
}
