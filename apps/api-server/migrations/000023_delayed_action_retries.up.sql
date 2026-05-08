ALTER TABLE public.automation_delayed_actions
    ADD COLUMN attempt_count integer NOT NULL DEFAULT 0,
    ADD COLUMN last_attempt_at timestamp with time zone;

ALTER TABLE public.automation_delayed_actions
    ADD CONSTRAINT automation_delayed_actions_attempt_count_nonnegative
    CHECK (attempt_count >= 0);

COMMENT ON COLUMN public.automation_delayed_actions.attempt_count IS 'Number of execution attempts recorded by DelayedActionWorker.';
COMMENT ON COLUMN public.automation_delayed_actions.last_attempt_at IS 'Timestamp of the latest execution attempt, successful or failed.';
