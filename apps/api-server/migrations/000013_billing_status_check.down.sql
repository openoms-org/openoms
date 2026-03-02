ALTER TABLE billing_subscriptions DROP CONSTRAINT IF EXISTS billing_subscriptions_status_check;
ALTER TABLE billing_subscriptions ADD CONSTRAINT billing_subscriptions_status_check
  CHECK (status IN ('trialing', 'active', 'past_due', 'canceled', 'unpaid'));
