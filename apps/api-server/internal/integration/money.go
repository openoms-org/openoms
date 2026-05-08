package integration

import (
	"fmt"
	"strconv"
)

// ParseMoneyString parses a decimal money string and returns a contextual error naming the source field.
func ParseMoneyString(field, value string) (float64, error) {
	amount, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0, fmt.Errorf("%s: parse money value %q: %w", field, value, err)
	}
	return amount, nil
}
