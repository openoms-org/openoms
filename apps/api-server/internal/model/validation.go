package model

import "fmt"

func validateMaxLength(field, value string, max int) error {
	if len(value) > max {
		return fmt.Errorf("%s exceeds maximum length of %d characters", field, max)
	}
	return nil
}

func validateMaxLengthPtr(field string, value *string, max int) error {
	if value != nil && len(*value) > max {
		return fmt.Errorf("%s exceeds maximum length of %d characters", field, max)
	}
	return nil
}
