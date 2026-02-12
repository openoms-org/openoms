ALTER TABLE users ADD COLUMN totp_secret TEXT;
ALTER TABLE users ADD COLUMN totp_enabled BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE users ADD COLUMN totp_verified_at TIMESTAMPTZ;

-- Update find_user_for_auth to include TOTP fields (avoids RLS pool pollution issue)
DROP FUNCTION IF EXISTS find_user_for_auth(text, uuid);
CREATE FUNCTION find_user_for_auth(p_email TEXT, p_tenant_id UUID)
RETURNS TABLE(
    id UUID, tenant_id UUID, email TEXT, name TEXT, password_hash TEXT,
    role TEXT, role_id UUID, created_at TIMESTAMPTZ, updated_at TIMESTAMPTZ,
    totp_secret TEXT, totp_enabled BOOLEAN
)
LANGUAGE sql SECURITY DEFINER SET search_path = public
AS $$
    SELECT u.id, u.tenant_id, u.email, u.name, u.password_hash,
           u.role, u.role_id, u.created_at, u.updated_at,
           u.totp_secret, u.totp_enabled
    FROM users u
    WHERE u.email = p_email AND u.tenant_id = p_tenant_id;
$$;
