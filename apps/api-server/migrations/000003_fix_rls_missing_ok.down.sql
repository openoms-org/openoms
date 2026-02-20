-- Down migration: revert missing_ok=true to original (not recommended)
-- This is a no-op because the new policies are strictly safer.
-- Reverting would break transaction mode pooler compatibility.
SELECT 1;
