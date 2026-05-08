-- Align existing system Administrator roles with the backend RBAC model:
-- administrators can manage settings/integrations/data, but only owners can manage users.
UPDATE roles
SET permissions = array_remove(permissions, 'users.manage'),
    updated_at = NOW()
WHERE is_system = TRUE
  AND name = 'Administrator'
  AND 'users.manage' = ANY(permissions);
