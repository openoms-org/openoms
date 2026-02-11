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

func TestUserService_CreateUser_ValidationError_MissingEmail(t *testing.T) {
	svc := NewUserService(nil, nil, nil, nil)

	_, err := svc.CreateUser(context.Background(), uuid.New(), model.CreateUserRequest{
		Name: "User",
		Role: "admin",
	}, uuid.New(), "127.0.0.1")

	require.Error(t, err)
	var ve *ValidationError
	assert.True(t, errors.As(err, &ve))
}

func TestUserService_CreateUser_ValidationError_MissingName(t *testing.T) {
	svc := NewUserService(nil, nil, nil, nil)

	_, err := svc.CreateUser(context.Background(), uuid.New(), model.CreateUserRequest{
		Email: "u@e.com",
		Role:  "admin",
	}, uuid.New(), "127.0.0.1")

	require.Error(t, err)
	var ve *ValidationError
	assert.True(t, errors.As(err, &ve))
}

func TestUserService_CreateUser_ValidationError_InvalidRole(t *testing.T) {
	svc := NewUserService(nil, nil, nil, nil)

	_, err := svc.CreateUser(context.Background(), uuid.New(), model.CreateUserRequest{
		Email: "u@e.com",
		Name:  "User",
		Role:  "superadmin",
	}, uuid.New(), "127.0.0.1")

	require.Error(t, err)
	var ve *ValidationError
	assert.True(t, errors.As(err, &ve))
	assert.Contains(t, err.Error(), "role")
}

func TestUserService_UpdateUser_ValidationError_NoFields(t *testing.T) {
	svc := NewUserService(nil, nil, nil, nil)

	_, err := svc.UpdateUser(context.Background(), uuid.New(), uuid.New(), model.UpdateUserRequest{}, uuid.New(), "127.0.0.1")

	require.Error(t, err)
	var ve *ValidationError
	assert.True(t, errors.As(err, &ve))
}

func TestUserService_UpdateUser_ValidationError_InvalidRole(t *testing.T) {
	svc := NewUserService(nil, nil, nil, nil)
	badRole := "superadmin"

	_, err := svc.UpdateUser(context.Background(), uuid.New(), uuid.New(), model.UpdateUserRequest{
		Role: &badRole,
	}, uuid.New(), "127.0.0.1")

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
