REVOKE EXECUTE ON FUNCTION public.mark_license_token_used(uuid, uuid, text, text) FROM openoms_auth;
REVOKE EXECUTE ON FUNCTION public.check_license_token_used(uuid) FROM openoms_auth;
DROP FUNCTION IF EXISTS public.mark_license_token_used(uuid, uuid, text, text);
DROP FUNCTION IF EXISTS public.check_license_token_used(uuid);
DROP TABLE IF EXISTS public.used_license_tokens;
