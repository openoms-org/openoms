package automation

import (
	"fmt"
	"reflect"
	"strings"
)

// EvaluateConditions evaluates all conditions against the given data.
// Returns true if all conditions are met (AND logic).
// Returns true if there are no conditions.
func EvaluateConditions(conditions []Condition, data map[string]any) bool {
	if len(conditions) == 0 {
		return true
	}
	for _, cond := range conditions {
		if !evaluateCondition(cond, data) {
			return false
		}
	}
	return true
}

func evaluateCondition(cond Condition, data map[string]any) bool {
	fieldValue := resolveField(cond.Field, data)

	switch cond.Operator {
	case "eq":
		return compareEqual(fieldValue, cond.Value)
	case "neq":
		return !compareEqual(fieldValue, cond.Value)
	case "in":
		return compareIn(fieldValue, cond.Value)
	case "not_in":
		return !compareIn(fieldValue, cond.Value)
	case "gt":
		return compareNumeric(fieldValue, cond.Value) > 0
	case "gte":
		return compareNumeric(fieldValue, cond.Value) >= 0
	case "lt":
		return compareNumeric(fieldValue, cond.Value) < 0
	case "lte":
		return compareNumeric(fieldValue, cond.Value) <= 0
	case "contains":
		return compareContains(fieldValue, cond.Value)
	case "not_contains":
		return !compareContains(fieldValue, cond.Value)
	case "starts_with":
		return compareStartsWith(fieldValue, cond.Value)
	default:
		return false
	}
}

// resolveField resolves a dotted field path from data.
// E.g. "order.status" resolves data["order"]["status"].
func resolveField(field string, data map[string]any) any {
	parts := strings.Split(field, ".")
	var current any = data
	for _, part := range parts {
		switch v := current.(type) {
		case map[string]any:
			current = v[part]
		default:
			return nil
		}
	}
	return current
}

func compareEqual(a, b any) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return fmt.Sprintf("%v", a) == fmt.Sprintf("%v", b)
}

func compareIn(fieldValue, listValue any) bool {
	if fieldValue == nil {
		return false
	}
	rv := reflect.ValueOf(listValue)
	if rv.Kind() != reflect.Slice {
		return compareEqual(fieldValue, listValue)
	}
	fieldStr := fmt.Sprintf("%v", fieldValue)
	for i := 0; i < rv.Len(); i++ {
		if fmt.Sprintf("%v", rv.Index(i).Interface()) == fieldStr {
			return true
		}
	}
	return false
}

func compareNumeric(a, b any) int {
	aNum := toFloat64(a)
	bNum := toFloat64(b)
	if aNum < bNum {
		return -1
	}
	if aNum > bNum {
		return 1
	}
	return 0
}

func toFloat64(v any) float64 {
	if v == nil {
		return 0
	}
	switch n := v.(type) {
	case float64:
		return n
	case float32:
		return float64(n)
	case int:
		return float64(n)
	case int64:
		return float64(n)
	case int32:
		return float64(n)
	case string:
		var f float64
		fmt.Sscanf(n, "%f", &f)
		return f
	default:
		var f float64
		fmt.Sscanf(fmt.Sprintf("%v", v), "%f", &f)
		return f
	}
}

func compareContains(fieldValue, searchValue any) bool {
	if fieldValue == nil {
		return false
	}
	fieldStr := fmt.Sprintf("%v", fieldValue)
	searchStr := fmt.Sprintf("%v", searchValue)
	return strings.Contains(fieldStr, searchStr)
}

func compareStartsWith(fieldValue, prefixValue any) bool {
	if fieldValue == nil {
		return false
	}
	fieldStr := fmt.Sprintf("%v", fieldValue)
	prefixStr := fmt.Sprintf("%v", prefixValue)
	return strings.HasPrefix(fieldStr, prefixStr)
}
