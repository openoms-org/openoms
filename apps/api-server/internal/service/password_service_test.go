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
	if err := svc.ValidateStrength("longpassword"); err != nil {
		t.Errorf("expected no error: %v", err)
	}
}
