package service

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openoms-org/openoms/apps/api-server/internal/model"
)

func TestSupplierService_Create_ValidationError_MissingName(t *testing.T) {
	svc := NewSupplierService(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)

	_, err := svc.Create(context.Background(), uuid.New(), model.CreateSupplierRequest{
		FeedFormat: "iof",
	}, uuid.New(), "127.0.0.1")

	require.Error(t, err)
	var ve *ValidationError
	assert.True(t, errors.As(err, &ve))
	assert.Contains(t, err.Error(), "name")
}

func TestSupplierService_Create_ValidationError_InvalidFeedFormat(t *testing.T) {
	svc := NewSupplierService(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)

	_, err := svc.Create(context.Background(), uuid.New(), model.CreateSupplierRequest{
		Name:       "Test Supplier",
		FeedFormat: "yaml",
	}, uuid.New(), "127.0.0.1")

	require.Error(t, err)
	var ve *ValidationError
	assert.True(t, errors.As(err, &ve))
	assert.Contains(t, err.Error(), "feed_format")
}

func TestSupplierService_Update_ValidationError_NoFields(t *testing.T) {
	svc := NewSupplierService(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)

	_, err := svc.Update(context.Background(), uuid.New(), uuid.New(), model.UpdateSupplierRequest{}, uuid.New(), "127.0.0.1")

	require.Error(t, err)
	var ve *ValidationError
	assert.True(t, errors.As(err, &ve))
}

// TestValidateBTPBaseURL covers the early, DNS-free SSRF defense-in-depth check applied
// to the user-supplied BTP wizard base_url (INJECT-01). The authoritative guard is the
// SafeHTTPClient dialer; this asserts the obvious internal/scheme cases are rejected.
func TestValidateBTPBaseURL(t *testing.T) {
	tests := []struct {
		name    string
		baseURL string
		wantErr bool
	}{
		{name: "empty allowed (SDK default)", baseURL: "", wantErr: false},
		{name: "blank allowed (SDK default)", baseURL: "   ", wantErr: false},
		{name: "valid public https", baseURL: "https://ext.btp.pro", wantErr: false},
		{name: "valid public https with path", baseURL: "https://api.btp.com/v1", wantErr: false},
		{name: "http scheme rejected", baseURL: "http://ext.btp.pro", wantErr: true},
		{name: "no scheme rejected", baseURL: "ext.btp.pro", wantErr: true},
		{name: "cloud metadata IP rejected", baseURL: "https://169.254.169.254", wantErr: true},
		{name: "loopback IP rejected", baseURL: "https://127.0.0.1", wantErr: true},
		{name: "rfc1918 IP rejected", baseURL: "https://10.0.0.5", wantErr: true},
		{name: "rfc1918 192.168 IP rejected", baseURL: "https://192.168.1.1:8080", wantErr: true},
		{name: "localhost rejected", baseURL: "https://localhost", wantErr: true},
		{name: "ipv6 loopback rejected", baseURL: "https://[::1]", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateBTPBaseURL(tt.baseURL)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestSupplierService_BTPWizardSetAPIKeys_BlockedBaseURL proves the wizard entry point
// rejects a base_url pointing at an internal host before any network call or DB access,
// i.e. the SSRF guard is wired into the BTP wizard path (INJECT-01). The nil pool would
// panic if execution reached database.WithTenant, so a clean ValidationError confirms the
// check runs first.
func TestSupplierService_BTPWizardSetAPIKeys_BlockedBaseURL(t *testing.T) {
	svc := NewSupplierService(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)

	_, err := svc.BTPWizardSetAPIKeys(context.Background(), uuid.New(), uuid.New(), model.BTPWizardSetAPIKeysRequest{
		PublicKey:  "pk_test_123",
		PrivateKey: "sk_test_456",
		BaseURL:    "https://169.254.169.254/latest/meta-data",
	}, uuid.New(), "127.0.0.1")

	require.Error(t, err)
	var ve *ValidationError
	assert.True(t, errors.As(err, &ve))
	assert.Contains(t, err.Error(), "base_url")
}
