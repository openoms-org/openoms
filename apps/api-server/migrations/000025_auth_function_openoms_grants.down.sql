-- Roll back the explicit self-hosted app-role grants added in migration 25.
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'openoms') THEN
    REVOKE EXECUTE ON FUNCTION public.find_tenant_by_slug(text) FROM openoms;
    REVOKE EXECUTE ON FUNCTION public.find_user_for_auth(text, uuid) FROM openoms;
    REVOKE EXECUTE ON FUNCTION public.find_invitation_by_token(text) FROM openoms;
    REVOKE EXECUTE ON FUNCTION public.find_return_by_token(text) FROM openoms;
    REVOKE EXECUTE ON FUNCTION public.find_order_tenant_id(uuid) FROM openoms;
    REVOKE EXECUTE ON FUNCTION public.use_invitation(text) FROM openoms;
  END IF;
END;
$$;
