DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'openoms_auth') THEN
    EXECUTE 'REVOKE EXECUTE ON FUNCTION public.update_license_token_tenant(uuid, uuid) FROM openoms_auth';
    EXECUTE 'REVOKE EXECUTE ON FUNCTION public.mark_license_token_used(uuid, uuid, text, text) FROM openoms_auth';
    EXECUTE 'REVOKE EXECUTE ON FUNCTION public.check_license_token_used(uuid) FROM openoms_auth';
  END IF;
END;
$$;

DROP FUNCTION IF EXISTS public.update_license_token_tenant(uuid, uuid);
DROP FUNCTION IF EXISTS public.mark_license_token_used(uuid, uuid, text, text);
DROP FUNCTION IF EXISTS public.check_license_token_used(uuid);
DROP TABLE IF EXISTS public.used_license_tokens;
