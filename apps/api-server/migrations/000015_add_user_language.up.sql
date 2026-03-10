-- Add language preference column to users table (NULL = auto-detect from browser)
ALTER TABLE public.users ADD COLUMN language VARCHAR(5);

-- Recreate find_user_for_auth to include the new language column.
-- Must DROP + SET ROLE + CREATE to preserve openoms_auth ownership.
DROP FUNCTION IF EXISTS public.find_user_for_auth(text, uuid);

SET ROLE openoms_auth;

CREATE FUNCTION public.find_user_for_auth(p_email text, p_tenant_id uuid)
 RETURNS TABLE(id uuid, tenant_id uuid, email text, name text, password_hash text, role text, role_id uuid, created_at timestamp with time zone, updated_at timestamp with time zone, totp_secret text, totp_enabled boolean, language character varying)
 LANGUAGE sql SECURITY DEFINER
 SET search_path TO 'public'
AS $$
    SELECT u.id, u.tenant_id, u.email, u.name, u.password_hash,
           u.role, u.role_id, u.created_at, u.updated_at,
           u.totp_secret, u.totp_enabled, u.language
    FROM users u
    WHERE u.email = p_email AND u.tenant_id = p_tenant_id;
$$;

RESET ROLE;

-- Restore execution permissions
REVOKE EXECUTE ON FUNCTION public.find_user_for_auth(text, uuid) FROM PUBLIC;

DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'openoms_app') THEN
    GRANT EXECUTE ON FUNCTION public.find_user_for_auth(text, uuid) TO openoms_app;
  END IF;
END;
$$;
