-- Restore the previous Administrator system role default.
UPDATE roles
SET permissions = array_append(permissions, 'users.manage'),
    updated_at = NOW()
WHERE is_system = TRUE
  AND name = 'Administrator'
  AND NOT ('users.manage' = ANY(permissions));
