package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openoms-org/openoms/apps/api-server/internal/model"
)

// --- Create Customer Tests ---

func TestCustomerService_Create_ValidationError_EmptyName(t *testing.T) {
	svc := NewCustomerService(nil, nil, nil, nil, nil)

	_, err := svc.Create(context.Background(), uuid.New(), model.CreateCustomerRequest{
		Name: "",
	}, uuid.New(), "127.0.0.1")

	require.Error(t, err)
	var ve *ValidationError
	assert.True(t, errors.As(err, &ve))
	assert.Contains(t, err.Error(), "name")
}

func TestCustomerService_Create_ValidationError_WhitespaceName(t *testing.T) {
	svc := NewCustomerService(nil, nil, nil, nil, nil)

	_, err := svc.Create(context.Background(), uuid.New(), model.CreateCustomerRequest{
		Name: "   ",
	}, uuid.New(), "127.0.0.1")

	require.Error(t, err)
	var ve *ValidationError
	assert.True(t, errors.As(err, &ve))
	assert.Contains(t, err.Error(), "name")
}

func TestCustomerService_Create_ValidationError_NameTooLong(t *testing.T) {
	svc := NewCustomerService(nil, nil, nil, nil, nil)

	longName := strings.Repeat("a", 501)
	_, err := svc.Create(context.Background(), uuid.New(), model.CreateCustomerRequest{
		Name: longName,
	}, uuid.New(), "127.0.0.1")

	require.Error(t, err)
	var ve *ValidationError
	assert.True(t, errors.As(err, &ve))
	assert.Contains(t, err.Error(), "name")
}

func TestCustomerService_Create_ValidationError_EmailTooLong(t *testing.T) {
	svc := NewCustomerService(nil, nil, nil, nil, nil)

	longEmail := strings.Repeat("a", 256)
	_, err := svc.Create(context.Background(), uuid.New(), model.CreateCustomerRequest{
		Name:  "Valid Name",
		Email: &longEmail,
	}, uuid.New(), "127.0.0.1")

	require.Error(t, err)
	var ve *ValidationError
	assert.True(t, errors.As(err, &ve))
	assert.Contains(t, err.Error(), "email")
}

func TestCustomerService_Create_ValidationError_InvalidEmailFormat(t *testing.T) {
	svc := NewCustomerService(nil, nil, nil, nil, nil)

	invalidEmail := "not-an-email"
	_, err := svc.Create(context.Background(), uuid.New(), model.CreateCustomerRequest{
		Name:  "Valid Name",
		Email: &invalidEmail,
	}, uuid.New(), "127.0.0.1")

	require.Error(t, err)
	var ve *ValidationError
	assert.True(t, errors.As(err, &ve))
	assert.Contains(t, err.Error(), "email")
}

func TestCustomerService_Create_ValidationError_PhoneTooLong(t *testing.T) {
	svc := NewCustomerService(nil, nil, nil, nil, nil)

	longPhone := strings.Repeat("1", 51)
	_, err := svc.Create(context.Background(), uuid.New(), model.CreateCustomerRequest{
		Name:  "Valid Name",
		Phone: &longPhone,
	}, uuid.New(), "127.0.0.1")

	require.Error(t, err)
	var ve *ValidationError
	assert.True(t, errors.As(err, &ve))
	assert.Contains(t, err.Error(), "phone")
}

func TestCustomerService_Create_ValidationError_CompanyNameTooLong(t *testing.T) {
	svc := NewCustomerService(nil, nil, nil, nil, nil)

	longCompany := strings.Repeat("c", 501)
	_, err := svc.Create(context.Background(), uuid.New(), model.CreateCustomerRequest{
		Name:        "Valid Name",
		CompanyName: &longCompany,
	}, uuid.New(), "127.0.0.1")

	require.Error(t, err)
	var ve *ValidationError
	assert.True(t, errors.As(err, &ve))
	assert.Contains(t, err.Error(), "company_name")
}

func TestCustomerService_Create_ValidationError_NIPTooLong(t *testing.T) {
	svc := NewCustomerService(nil, nil, nil, nil, nil)

	longNIP := strings.Repeat("1", 21)
	_, err := svc.Create(context.Background(), uuid.New(), model.CreateCustomerRequest{
		Name: "Valid Name",
		NIP:  &longNIP,
	}, uuid.New(), "127.0.0.1")

	require.Error(t, err)
	var ve *ValidationError
	assert.True(t, errors.As(err, &ve))
	assert.Contains(t, err.Error(), "nip")
}

func TestCustomerService_Create_ValidationError_NotesTooLong(t *testing.T) {
	svc := NewCustomerService(nil, nil, nil, nil, nil)

	longNotes := strings.Repeat("n", 5001)
	_, err := svc.Create(context.Background(), uuid.New(), model.CreateCustomerRequest{
		Name:  "Valid Name",
		Notes: &longNotes,
	}, uuid.New(), "127.0.0.1")

	require.Error(t, err)
	var ve *ValidationError
	assert.True(t, errors.As(err, &ve))
	assert.Contains(t, err.Error(), "notes")
}

// --- Update Customer Tests ---

func TestCustomerService_Update_ValidationError_NoFieldsProvided(t *testing.T) {
	svc := NewCustomerService(nil, nil, nil, nil, nil)

	_, err := svc.Update(context.Background(), uuid.New(), uuid.New(), model.UpdateCustomerRequest{}, uuid.New(), "127.0.0.1")

	require.Error(t, err)
	var ve *ValidationError
	assert.True(t, errors.As(err, &ve))
	assert.Contains(t, err.Error(), "at least one field")
}

func TestCustomerService_Update_ValidationError_EmptyName(t *testing.T) {
	svc := NewCustomerService(nil, nil, nil, nil, nil)

	emptyName := ""
	_, err := svc.Update(context.Background(), uuid.New(), uuid.New(), model.UpdateCustomerRequest{
		Name: &emptyName,
	}, uuid.New(), "127.0.0.1")

	require.Error(t, err)
	var ve *ValidationError
	assert.True(t, errors.As(err, &ve))
	assert.Contains(t, err.Error(), "name")
}

func TestCustomerService_Update_ValidationError_WhitespaceName(t *testing.T) {
	svc := NewCustomerService(nil, nil, nil, nil, nil)

	wsName := "   "
	_, err := svc.Update(context.Background(), uuid.New(), uuid.New(), model.UpdateCustomerRequest{
		Name: &wsName,
	}, uuid.New(), "127.0.0.1")

	require.Error(t, err)
	var ve *ValidationError
	assert.True(t, errors.As(err, &ve))
	assert.Contains(t, err.Error(), "name")
}

func TestCustomerService_Update_ValidationError_NameTooLong(t *testing.T) {
	svc := NewCustomerService(nil, nil, nil, nil, nil)

	longName := strings.Repeat("a", 501)
	_, err := svc.Update(context.Background(), uuid.New(), uuid.New(), model.UpdateCustomerRequest{
		Name: &longName,
	}, uuid.New(), "127.0.0.1")

	require.Error(t, err)
	var ve *ValidationError
	assert.True(t, errors.As(err, &ve))
	assert.Contains(t, err.Error(), "name")
}

func TestCustomerService_Update_ValidationError_EmailTooLong(t *testing.T) {
	svc := NewCustomerService(nil, nil, nil, nil, nil)

	longEmail := strings.Repeat("a", 256)
	_, err := svc.Update(context.Background(), uuid.New(), uuid.New(), model.UpdateCustomerRequest{
		Email: &longEmail,
	}, uuid.New(), "127.0.0.1")

	require.Error(t, err)
	var ve *ValidationError
	assert.True(t, errors.As(err, &ve))
	assert.Contains(t, err.Error(), "email")
}

func TestCustomerService_Update_ValidationError_InvalidEmailFormat(t *testing.T) {
	svc := NewCustomerService(nil, nil, nil, nil, nil)

	badEmail := "invalid-email"
	_, err := svc.Update(context.Background(), uuid.New(), uuid.New(), model.UpdateCustomerRequest{
		Email: &badEmail,
	}, uuid.New(), "127.0.0.1")

	require.Error(t, err)
	var ve *ValidationError
	assert.True(t, errors.As(err, &ve))
	assert.Contains(t, err.Error(), "email")
}

func TestCustomerService_Update_ValidationError_PhoneTooLong(t *testing.T) {
	svc := NewCustomerService(nil, nil, nil, nil, nil)

	longPhone := strings.Repeat("1", 51)
	_, err := svc.Update(context.Background(), uuid.New(), uuid.New(), model.UpdateCustomerRequest{
		Phone: &longPhone,
	}, uuid.New(), "127.0.0.1")

	require.Error(t, err)
	var ve *ValidationError
	assert.True(t, errors.As(err, &ve))
	assert.Contains(t, err.Error(), "phone")
}

func TestCustomerService_Update_ValidationError_CompanyNameTooLong(t *testing.T) {
	svc := NewCustomerService(nil, nil, nil, nil, nil)

	longCompany := strings.Repeat("c", 501)
	_, err := svc.Update(context.Background(), uuid.New(), uuid.New(), model.UpdateCustomerRequest{
		CompanyName: &longCompany,
	}, uuid.New(), "127.0.0.1")

	require.Error(t, err)
	var ve *ValidationError
	assert.True(t, errors.As(err, &ve))
	assert.Contains(t, err.Error(), "company_name")
}

func TestCustomerService_Update_ValidationError_NIPTooLong(t *testing.T) {
	svc := NewCustomerService(nil, nil, nil, nil, nil)

	longNIP := strings.Repeat("1", 21)
	_, err := svc.Update(context.Background(), uuid.New(), uuid.New(), model.UpdateCustomerRequest{
		NIP: &longNIP,
	}, uuid.New(), "127.0.0.1")

	require.Error(t, err)
	var ve *ValidationError
	assert.True(t, errors.As(err, &ve))
	assert.Contains(t, err.Error(), "nip")
}

func TestCustomerService_Update_ValidationError_NotesTooLong(t *testing.T) {
	svc := NewCustomerService(nil, nil, nil, nil, nil)

	longNotes := strings.Repeat("n", 5001)
	_, err := svc.Update(context.Background(), uuid.New(), uuid.New(), model.UpdateCustomerRequest{
		Notes: &longNotes,
	}, uuid.New(), "127.0.0.1")

	require.Error(t, err)
	var ve *ValidationError
	assert.True(t, errors.As(err, &ve))
	assert.Contains(t, err.Error(), "notes")
}
