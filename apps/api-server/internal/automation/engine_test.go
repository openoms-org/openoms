package automation

import (
	"testing"
)

func TestEvaluateConditions_EmptyConditions(t *testing.T) {
	result := EvaluateConditions(nil, map[string]any{"status": "new"})
	if !result {
		t.Error("expected true for empty conditions")
	}
}

func TestEvaluateConditions_Eq(t *testing.T) {
	conds := []Condition{{Field: "status", Operator: "eq", Value: "new"}}
	if !EvaluateConditions(conds, map[string]any{"status": "new"}) {
		t.Error("expected true for eq match")
	}
	if EvaluateConditions(conds, map[string]any{"status": "shipped"}) {
		t.Error("expected false for eq mismatch")
	}
}

func TestEvaluateConditions_Neq(t *testing.T) {
	conds := []Condition{{Field: "status", Operator: "neq", Value: "cancelled"}}
	if !EvaluateConditions(conds, map[string]any{"status": "new"}) {
		t.Error("expected true for neq match")
	}
	if EvaluateConditions(conds, map[string]any{"status": "cancelled"}) {
		t.Error("expected false for neq mismatch")
	}
}

func TestEvaluateConditions_In(t *testing.T) {
	conds := []Condition{{Field: "status", Operator: "in", Value: []any{"new", "confirmed"}}}
	if !EvaluateConditions(conds, map[string]any{"status": "new"}) {
		t.Error("expected true for in match")
	}
	if !EvaluateConditions(conds, map[string]any{"status": "confirmed"}) {
		t.Error("expected true for in match")
	}
	if EvaluateConditions(conds, map[string]any{"status": "shipped"}) {
		t.Error("expected false for in mismatch")
	}
}

func TestEvaluateConditions_NotIn(t *testing.T) {
	conds := []Condition{{Field: "status", Operator: "not_in", Value: []any{"cancelled", "refunded"}}}
	if !EvaluateConditions(conds, map[string]any{"status": "new"}) {
		t.Error("expected true for not_in match")
	}
	if EvaluateConditions(conds, map[string]any{"status": "cancelled"}) {
		t.Error("expected false for not_in mismatch")
	}
}

func TestEvaluateConditions_Gt(t *testing.T) {
	conds := []Condition{{Field: "total_amount", Operator: "gt", Value: 100.0}}
	if !EvaluateConditions(conds, map[string]any{"total_amount": 150.0}) {
		t.Error("expected true for gt match")
	}
	if EvaluateConditions(conds, map[string]any{"total_amount": 50.0}) {
		t.Error("expected false for gt mismatch")
	}
	if EvaluateConditions(conds, map[string]any{"total_amount": 100.0}) {
		t.Error("expected false for gt equal")
	}
}

func TestEvaluateConditions_Gte(t *testing.T) {
	conds := []Condition{{Field: "total_amount", Operator: "gte", Value: 100.0}}
	if !EvaluateConditions(conds, map[string]any{"total_amount": 100.0}) {
		t.Error("expected true for gte equal")
	}
	if !EvaluateConditions(conds, map[string]any{"total_amount": 150.0}) {
		t.Error("expected true for gte greater")
	}
	if EvaluateConditions(conds, map[string]any{"total_amount": 99.0}) {
		t.Error("expected false for gte less")
	}
}

func TestEvaluateConditions_Lt(t *testing.T) {
	conds := []Condition{{Field: "total_amount", Operator: "lt", Value: 100.0}}
	if !EvaluateConditions(conds, map[string]any{"total_amount": 50.0}) {
		t.Error("expected true for lt match")
	}
	if EvaluateConditions(conds, map[string]any{"total_amount": 150.0}) {
		t.Error("expected false for lt mismatch")
	}
}

func TestEvaluateConditions_Lte(t *testing.T) {
	conds := []Condition{{Field: "total_amount", Operator: "lte", Value: 100.0}}
	if !EvaluateConditions(conds, map[string]any{"total_amount": 100.0}) {
		t.Error("expected true for lte equal")
	}
	if !EvaluateConditions(conds, map[string]any{"total_amount": 50.0}) {
		t.Error("expected true for lte less")
	}
	if EvaluateConditions(conds, map[string]any{"total_amount": 101.0}) {
		t.Error("expected false for lte greater")
	}
}

func TestEvaluateConditions_Contains(t *testing.T) {
	conds := []Condition{{Field: "customer_name", Operator: "contains", Value: "Jan"}}
	if !EvaluateConditions(conds, map[string]any{"customer_name": "Jan Kowalski"}) {
		t.Error("expected true for contains match")
	}
	if EvaluateConditions(conds, map[string]any{"customer_name": "Anna Nowak"}) {
		t.Error("expected false for contains mismatch")
	}
}

func TestEvaluateConditions_NotContains(t *testing.T) {
	conds := []Condition{{Field: "customer_name", Operator: "not_contains", Value: "Test"}}
	if !EvaluateConditions(conds, map[string]any{"customer_name": "Jan Kowalski"}) {
		t.Error("expected true for not_contains match")
	}
	if EvaluateConditions(conds, map[string]any{"customer_name": "Test User"}) {
		t.Error("expected false for not_contains mismatch")
	}
}

func TestEvaluateConditions_DottedField(t *testing.T) {
	conds := []Condition{{Field: "order.status", Operator: "eq", Value: "confirmed"}}
	data := map[string]any{
		"order": map[string]any{
			"status": "confirmed",
		},
	}
	if !EvaluateConditions(conds, data) {
		t.Error("expected true for dotted field match")
	}
}

func TestEvaluateConditions_MultipleConditions(t *testing.T) {
	conds := []Condition{
		{Field: "status", Operator: "eq", Value: "confirmed"},
		{Field: "total_amount", Operator: "gt", Value: 100.0},
	}
	data := map[string]any{
		"status":       "confirmed",
		"total_amount": 150.0,
	}
	if !EvaluateConditions(conds, data) {
		t.Error("expected true for all conditions met")
	}

	data2 := map[string]any{
		"status":       "confirmed",
		"total_amount": 50.0,
	}
	if EvaluateConditions(conds, data2) {
		t.Error("expected false when not all conditions met")
	}
}

func TestEvaluateConditions_NilFieldValue(t *testing.T) {
	conds := []Condition{{Field: "missing_field", Operator: "eq", Value: "test"}}
	if EvaluateConditions(conds, map[string]any{"status": "new"}) {
		t.Error("expected false for nil field value with eq")
	}
}

func TestEvaluateConditions_InvalidOperator(t *testing.T) {
	conds := []Condition{{Field: "status", Operator: "invalid", Value: "new"}}
	if EvaluateConditions(conds, map[string]any{"status": "new"}) {
		t.Error("expected false for invalid operator")
	}
}

func TestResolveField_TopLevel(t *testing.T) {
	data := map[string]any{"status": "new"}
	val := resolveField("status", data)
	if val != "new" {
		t.Errorf("expected 'new', got %v", val)
	}
}

func TestResolveField_Nested(t *testing.T) {
	data := map[string]any{
		"order": map[string]any{
			"total_amount": 150.0,
		},
	}
	val := resolveField("order.total_amount", data)
	if val != 150.0 {
		t.Errorf("expected 150.0, got %v", val)
	}
}

func TestResolveField_DeepNested(t *testing.T) {
	data := map[string]any{
		"a": map[string]any{
			"b": map[string]any{
				"c": "deep_value",
			},
		},
	}
	val := resolveField("a.b.c", data)
	if val != "deep_value" {
		t.Errorf("expected 'deep_value', got %v", val)
	}
}

func TestResolveField_Missing(t *testing.T) {
	data := map[string]any{"status": "new"}
	val := resolveField("missing", data)
	if val != nil {
		t.Errorf("expected nil, got %v", val)
	}
}

func TestToFloat64(t *testing.T) {
	tests := []struct {
		input    any
		expected float64
	}{
		{42.5, 42.5},
		{float32(42.5), 42.5},
		{42, 42.0},
		{int64(42), 42.0},
		{int32(42), 42.0},
		{"42.5", 42.5},
		{nil, 0},
	}
	for _, tt := range tests {
		result := toFloat64(tt.input)
		if result != tt.expected {
			t.Errorf("toFloat64(%v) = %v, want %v", tt.input, result, tt.expected)
		}
	}
}
