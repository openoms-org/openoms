-- SECURITY DEFINER function to look up invitations by token.
-- Needed because registration happens before tenant context is set (no RLS session var).
CREATE OR REPLACE FUNCTION find_invitation_by_token(p_token TEXT)
RETURNS TABLE (
    id          UUID,
    tenant_id   UUID,
    email       TEXT,
    role        TEXT,
    expires_at  TIMESTAMPTZ,
    used_at     TIMESTAMPTZ
) LANGUAGE sql SECURITY DEFINER SET search_path = public AS $$
    SELECT i.id, i.tenant_id, i.email, i.role, i.expires_at, i.used_at
    FROM invitations i
    WHERE i.token = p_token
    LIMIT 1;
$$;

-- SECURITY DEFINER function to mark invitation as used.
CREATE OR REPLACE FUNCTION use_invitation(p_token TEXT)
RETURNS VOID LANGUAGE sql SECURITY DEFINER SET search_path = public AS $$
    UPDATE invitations SET used_at = now() WHERE token = p_token AND used_at IS NULL;
$$;
