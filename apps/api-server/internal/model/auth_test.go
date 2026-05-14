package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoginRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		req     LoginRequest
		wantErr string
	}{
		{
			name:    "missing email",
			req:     LoginRequest{Password: "pass", TenantSlug: "acme"},
			wantErr: "email is required",
		},
		{
			name:    "missing password",
			req:     LoginRequest{Email: "a@b.com", TenantSlug: "acme"},
			wantErr: "password is required",
		},
		{
			name:    "missing tenant_slug",
			req:     LoginRequest{Email: "a@b.com", Password: "pass"},
			wantErr: "tenant_slug is required",
		},
		{
			name: "valid",
			req:  LoginRequest{Email: "a@b.com", Password: "pass", TenantSlug: "acme"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestRegisterRequest_Validate(t *testing.T) {
	valid := RegisterRequest{
		Email:      "test@example.com",
		Password:   "Password123",
		Name:       "Jan",
		TenantName: "Acme Corp",
		TenantSlug: "acme",
	}

	t.Run("valid request", func(t *testing.T) {
		require.NoError(t, valid.Validate())
	})

	t.Run("short password", func(t *testing.T) {
		r := valid
		r.Password = "short"
		err := r.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "at least 8 characters")
	})

	t.Run("missing name", func(t *testing.T) {
		r := valid
		r.Name = ""
		err := r.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "name is required")
	})

	t.Run("missing tenant_name", func(t *testing.T) {
		r := valid
		r.TenantName = ""
		err := r.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "tenant_name is required")
	})
}

func TestCreateUserRequest_Validate(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		req := CreateUserRequest{Email: "a@b.com", Name: "Jan", Role: "admin", Password: "Password123"}
		require.NoError(t, req.Validate())
	})

	t.Run("missing email", func(t *testing.T) {
		req := CreateUserRequest{Name: "Jan", Role: "admin", Password: "Password123"}
		require.Error(t, req.Validate())
	})

	t.Run("missing password", func(t *testing.T) {
		req := CreateUserRequest{Email: "a@b.com", Name: "Jan", Role: "admin"}
		err := req.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "password is required")
	})

	t.Run("weak password", func(t *testing.T) {
		req := CreateUserRequest{Email: "a@b.com", Name: "Jan", Role: "admin", Password: "password"}
		err := req.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "password must contain uppercase, lowercase, and digit")
	})

	t.Run("invalid role", func(t *testing.T) {
		req := CreateUserRequest{Email: "a@b.com", Name: "Jan", Role: "superuser", Password: "Password123"}
		err := req.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "role must be one of")
	})

	t.Run("all valid roles", func(t *testing.T) {
		for _, role := range []string{"owner", "admin", "member"} {
			req := CreateUserRequest{Email: "a@b.com", Name: "Jan", Role: role, Password: "Password123"}
			require.NoError(t, req.Validate(), "role %s should be valid", role)
		}
	})
}

func TestChangePasswordRequest_Validate(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		req := ChangePasswordRequest{CurrentPassword: "OldPassword123", NewPassword: "NewPassword123"}
		require.NoError(t, req.Validate())
	})

	t.Run("missing current password", func(t *testing.T) {
		req := ChangePasswordRequest{NewPassword: "NewPassword123"}
		err := req.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "current_password is required")
	})

	t.Run("missing new password", func(t *testing.T) {
		req := ChangePasswordRequest{CurrentPassword: "OldPassword123"}
		err := req.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "new_password is required")
	})

	t.Run("weak new password", func(t *testing.T) {
		req := ChangePasswordRequest{CurrentPassword: "OldPassword123", NewPassword: "password"}
		err := req.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "password must contain uppercase, lowercase, and digit")
	})
}

func TestUpdateUserRequest_Validate(t *testing.T) {
	t.Run("empty update", func(t *testing.T) {
		req := UpdateUserRequest{}
		require.Error(t, req.Validate())
	})

	t.Run("invalid role", func(t *testing.T) {
		role := "superuser"
		req := UpdateUserRequest{Role: &role}
		err := req.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "role must be one of")
	})

	t.Run("valid name update", func(t *testing.T) {
		name := "New Name"
		req := UpdateUserRequest{Name: &name}
		require.NoError(t, req.Validate())
	})
}
