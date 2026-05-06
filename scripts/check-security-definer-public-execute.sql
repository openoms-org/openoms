\set ON_ERROR_STOP on

DO $$
DECLARE
  exposed_functions text;
BEGIN
  SELECT string_agg(
           format('%I.%I(%s)', n.nspname, p.proname, pg_get_function_identity_arguments(p.oid)),
           E'\n'
           ORDER BY n.nspname, p.proname, pg_get_function_identity_arguments(p.oid)
         )
    INTO exposed_functions
  FROM pg_proc p
  JOIN pg_namespace n ON n.oid = p.pronamespace
  JOIN LATERAL aclexplode(COALESCE(p.proacl, acldefault('f', p.proowner))) acl ON true
    WHERE n.nspname = 'public'
      AND p.prosecdef
      AND acl.grantee = 0
      AND acl.privilege_type = 'EXECUTE';

  IF exposed_functions IS NOT NULL THEN
    RAISE EXCEPTION 'SECURITY DEFINER functions executable by PUBLIC:%', E'\n' || exposed_functions;
  END IF;
END;
$$;
