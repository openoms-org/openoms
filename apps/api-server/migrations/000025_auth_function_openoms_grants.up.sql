-- Repair self-hosted app-role EXECUTE grants after auth helper functions were
-- recreated by earlier migrations.
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'openoms') THEN
    GRANT EXECUTE ON FUNCTION public.find_tenant_by_slug(text) TO openoms;
    GRANT EXECUTE ON FUNCTION public.find_user_for_auth(text, uuid) TO openoms;
    GRANT EXECUTE ON FUNCTION public.find_invitation_by_token(text) TO openoms;
    GRANT EXECUTE ON FUNCTION public.find_return_by_token(text) TO openoms;
    GRANT EXECUTE ON FUNCTION public.find_order_tenant_id(uuid) TO openoms;
    GRANT EXECUTE ON FUNCTION public.use_invitation(text) TO openoms;
  END IF;
END;
$$;
