-- Allow all Stripe subscription statuses in the check constraint.
-- Previously missing: incomplete, incomplete_expired, paused.
ALTER TABLE billing_subscriptions DROP CONSTRAINT IF EXISTS billing_subscriptions_status_check;
ALTER TABLE billing_subscriptions ADD CONSTRAINT billing_subscriptions_status_check
  CHECK (status IN ('incomplete', 'incomplete_expired', 'trialing', 'active', 'past_due', 'canceled', 'unpaid', 'paused'));
