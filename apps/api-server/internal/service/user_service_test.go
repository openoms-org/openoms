package service

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openoms-org/openoms/apps/api-server/internal/model"
)

func TestUserService_CreateUser_ValidationError_MissingEmail(t *testing.T) {
	svc := NewUserService(nil, nil, nil, nil)

	_, err := svc.CreateUser(context.Background(), uuid.New(), model.CreateUserRequest{
		Name:     "User",
		Role:     "admin",
		Password: "Password123",
	}, uuid.New(), "127.0.0.1")

	require.Error(t, err)
	var ve *ValidationError
	assert.True(t, errors.As(err, &ve))
}

func TestUserService_CreateUser_ValidationError_MissingName(t *testing.T) {
	svc := NewUserService(nil, nil, nil, nil)

	_, err := svc.CreateUser(context.Background(), uuid.New(), model.CreateUserRequest{
		Email:    "u@e.com",
		Role:     "admin",
		Password: "Password123",
	}, uuid.New(), "127.0.0.1")

	require.Error(t, err)
	var ve *ValidationError
	assert.True(t, errors.As(err, &ve))
}

func TestUserService_CreateUser_ValidationError_InvalidRole(t *testing.T) {
	svc := NewUserService(nil, nil, nil, nil)

	_, err := svc.CreateUser(context.Background(), uuid.New(), model.CreateUserRequest{
		Email:    "u@e.com",
		Name:     "User",
		Role:     "superadmin",
		Password: "Password123",
	}, uuid.New(), "127.0.0.1")

	require.Error(t, err)
	var ve *ValidationError
	assert.True(t, errors.As(err, &ve))
	assert.Contains(t, err.Error(), "role")
}

func TestUserService_CreateUser_ValidationError_MissingPassword(t *testing.T) {
	svc := NewUserService(nil, nil, nil, nil)

	_, err := svc.CreateUser(context.Background(), uuid.New(), model.CreateUserRequest{
		Email: "u@e.com",
		Name:  "User",
		Role:  "admin",
	}, uuid.New(), "127.0.0.1")

	require.Error(t, err)
	var ve *ValidationError
	assert.True(t, errors.As(err, &ve))
	assert.Contains(t, err.Error(), "password is required")
}

func TestUserService_CreateUser_ValidationError_WeakPassword(t *testing.T) {
	svc := NewUserService(nil, nil, NewPasswordService(), nil)

	_, err := svc.CreateUser(context.Background(), uuid.New(), model.CreateUserRequest{
		Email:    "u@e.com",
		Name:     "User",
		Role:     "admin",
		Password: "password",
	}, uuid.New(), "127.0.0.1")

	require.Error(t, err)
	var ve *ValidationError
	assert.True(t, errors.As(err, &ve))
	assert.Contains(t, err.Error(), "password")
}

func TestUserService_ChangePassword_ValidationError_MissingCurrentPassword(t *testing.T) {
	svc := NewUserService(nil, nil, nil, nil)

	err := svc.ChangePassword(context.Background(), uuid.New(), uuid.New(), model.ChangePasswordRequest{
		NewPassword: "NewPassword123",
	}, uuid.New(), "127.0.0.1")

	require.Error(t, err)
	var ve *ValidationError
	assert.True(t, errors.As(err, &ve))
	assert.Contains(t, err.Error(), "current_password is required")
}

func TestUserService_ChangePassword_ValidationError_WeakNewPassword(t *testing.T) {
	svc := NewUserService(nil, nil, NewPasswordService(), nil)

	err := svc.ChangePassword(context.Background(), uuid.New(), uuid.New(), model.ChangePasswordRequest{
		CurrentPassword: "OldPassword123",
		NewPassword:     "password",
	}, uuid.New(), "127.0.0.1")

	require.Error(t, err)
	var ve *ValidationError
	assert.True(t, errors.As(err, &ve))
	assert.Contains(t, err.Error(), "password")
}

func TestUserService_ChangePasswordInTx_RejectsWrongCurrentPassword(t *testing.T) {
	passwordSvc := NewPasswordService()
	oldHash, err := passwordSvc.Hash("OldPassword123")
	require.NoError(t, err)

	userRepo := &passwordChangeUserRepo{passwordHash: oldHash}
	svc := NewUserService(userRepo, &passwordChangeAuditRepo{}, passwordSvc, nil)

	err = svc.changePasswordInTx(context.Background(), nil, uuid.New(), uuid.New(), model.ChangePasswordRequest{
		CurrentPassword: "WrongPassword123",
		NewPassword:     "NewPassword123",
	}, uuid.New(), "127.0.0.1")

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidCurrentPassword)
	assert.False(t, userRepo.updateCalled)
}

func TestUserService_ChangePasswordInTx_UpdatesHashAndAudits(t *testing.T) {
	passwordSvc := NewPasswordService()
	oldHash, err := passwordSvc.Hash("OldPassword123")
	require.NoError(t, err)

	userID := uuid.New()
	actorID := uuid.New()
	userRepo := &passwordChangeUserRepo{passwordHash: oldHash}
	auditRepo := &passwordChangeAuditRepo{}
	svc := NewUserService(userRepo, auditRepo, passwordSvc, nil)

	err = svc.changePasswordInTx(context.Background(), nil, uuid.New(), userID, model.ChangePasswordRequest{
		CurrentPassword: "OldPassword123",
		NewPassword:     "NewPassword123",
	}, actorID, "127.0.0.1")

	require.NoError(t, err)
	require.True(t, userRepo.updateCalled)
	assert.Equal(t, userID, userRepo.updatedUserID)
	assert.NoError(t, passwordSvc.Compare(userRepo.updatedHash, "NewPassword123"))
	assert.Error(t, passwordSvc.Compare(userRepo.updatedHash, "OldPassword123"))
	assert.Equal(t, "user.password_changed", auditRepo.entry.Action)
	assert.Equal(t, actorID, auditRepo.entry.UserID)
	assert.Equal(t, userID, auditRepo.entry.EntityID)
}

type passwordChangeUserRepo struct {
	loginTimingUserRepo
	passwordHash  string
	updateCalled  bool
	updatedUserID uuid.UUID
	updatedHash   string
}

func (r *passwordChangeUserRepo) FindPasswordHashByID(context.Context, pgx.Tx, uuid.UUID) (*string, error) {
	return &r.passwordHash, nil
}

func (r *passwordChangeUserRepo) UpdatePassword(_ context.Context, _ pgx.Tx, id uuid.UUID, hash string) error {
	r.updateCalled = true
	r.updatedUserID = id
	r.updatedHash = hash
	return nil
}

type passwordChangeAuditRepo struct {
	entry model.AuditEntry
}

func (r *passwordChangeAuditRepo) Log(_ context.Context, _ pgx.Tx, entry model.AuditEntry) error {
	r.entry = entry
	return nil
}

func (r *passwordChangeAuditRepo) ListByEntity(context.Context, pgx.Tx, string, uuid.UUID) ([]model.AuditLogEntry, error) {
	return nil, nil
}

func (r *passwordChangeAuditRepo) List(context.Context, pgx.Tx, model.AuditListFilter) ([]model.AuditLogEntry, int, error) {
	return nil, 0, nil
}

func TestUserService_UpdateUser_ValidationError_NoFields(t *testing.T) {
	svc := NewUserService(nil, nil, nil, nil)

	_, err := svc.UpdateUser(context.Background(), uuid.New(), uuid.New(), model.UpdateUserRequest{}, uuid.New(), "admin", "127.0.0.1")

	require.Error(t, err)
	var ve *ValidationError
	assert.True(t, errors.As(err, &ve))
}

func TestUserService_UpdateUser_ValidationError_InvalidRole(t *testing.T) {
	svc := NewUserService(nil, nil, nil, nil)
	badRole := "superadmin"

	_, err := svc.UpdateUser(context.Background(), uuid.New(), uuid.New(), model.UpdateUserRequest{
		Role: &badRole,
	}, uuid.New(), "admin", "127.0.0.1")

	require.Error(t, err)
	var ve *ValidationError
	assert.True(t, errors.As(err, &ve))
}

func TestUserService_DeleteUser_CannotDeleteSelf(t *testing.T) {
	svc := NewUserService(nil, nil, nil, nil)
	actorID := uuid.New()

	err := svc.DeleteUser(context.Background(), uuid.New(), actorID, actorID, "127.0.0.1")

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrCannotDeleteSelf)
}
