package service

import "testing"

func TestHashAndCompare(t *testing.T) {
	svc := NewPasswordService()
	hash, err := svc.Hash("password123")
	if err != nil {
		t.Fatalf("Hash: %v", err)
	}
	if err := svc.Compare(hash, "password123"); err != nil {
		t.Errorf("Compare should succeed: %v", err)
	}
}

func TestCompareWrongPassword(t *testing.T) {
	svc := NewPasswordService()
	hash, _ := svc.Hash("password123")
	if err := svc.Compare(hash, "wrongpassword"); err == nil {
		t.Error("Compare should fail for wrong password")
	}
}

func TestValidateStrength_TooShort(t *testing.T) {
	svc := NewPasswordService()
	if err := svc.ValidateStrength("short"); err == nil {
		t.Error("expected error for short password")
	}
}

func TestValidateStrength_OK(t *testing.T) {
	svc := NewPasswordService()
	if err := svc.ValidateStrength("longpassword1"); err != nil {
		t.Errorf("expected no error: %v", err)
	}
}

func TestValidateStrength_TooLong(t *testing.T) {
	svc := NewPasswordService()
	long := make([]byte, 73)
	for i := range long {
		long[i] = 'a'
	}
	long[0] = '1' // include a digit
	if err := svc.ValidateStrength(string(long)); err == nil {
		t.Error("expected error for password exceeding 72 characters")
	}
}

func TestValidateStrength_NoDigit(t *testing.T) {
	svc := NewPasswordService()
	if err := svc.ValidateStrength("longpassword"); err == nil {
		t.Error("expected error for password without digit")
	}
}

func TestValidateStrength_NoLetter(t *testing.T) {
	svc := NewPasswordService()
	if err := svc.ValidateStrength("12345678"); err == nil {
		t.Error("expected error for password without letter")
	}
}
