package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSystemRoleAdminPermissionsExcludeUsersManage(t *testing.T) {
	assert.NotContains(t, SystemRoleAdminPermissions, PermUsersManage)
	assert.Contains(t, SystemRoleOwnerPermissions, PermUsersManage)
}

func TestSystemPermissionsForRoleReturnsCopy(t *testing.T) {
	perms := SystemPermissionsForRole("member")
	assert.Contains(t, perms, PermOrdersView)

	perms[0] = "mutated"
	assert.NotEqual(t, "mutated", SystemRoleMemberPermissions[0])
}
