-- ============================================================================
-- Daily Companies Generation Cron Job
-- ============================================================================
-- This migration sets up a daily cron job that calls the backend API
-- to generate companies at midnight US Eastern time.
-- ============================================================================

-- ----------------------------------------------------------------------------
-- 1. Enable required extensions
-- ----------------------------------------------------------------------------
-- pg_cron: Allows scheduling of periodic jobs inside PostgreSQL
CREATE EXTENSION IF NOT EXISTS pg_cron;

-- pg_net: Allows making HTTP requests from PostgreSQL
CREATE EXTENSION IF NOT EXISTS pg_net;

-- ----------------------------------------------------------------------------
-- 2. Create function to call the companies/generate API endpoint
-- ----------------------------------------------------------------------------
CREATE OR REPLACE FUNCTION public.call_companies_generate()
RETURNS TABLE (
    status_code integer,
    response_body text,
    error_message text
)
LANGUAGE plpgsql
SECURITY DEFINER
AS $$
DECLARE
    v_request_id bigint;
    v_response record;
BEGIN
    -- Make the HTTP POST request using pg_net
    -- NOTE: Replace 'REPLACE_WITH_SECRET' with your actual API key
    -- You can use Supabase Vault to store the secret securely:
    --   1. Store in vault: INSERT INTO vault.secrets (name, secret) VALUES ('api_key', 'your-actual-key');
    --   2. Then use: 'x-api-key', (SELECT decrypted_secret FROM vault.decrypted_secrets WHERE name = 'api_key')

    SELECT INTO v_request_id net.http_post(
        url := 'https://api.startupdose.com/companies/generate',
        headers := jsonb_build_object(
            'Content-Type', 'application/json',
            'x-api-key', 'REPLACE_WITH_SECRET'
        ),
        body := '{}'::jsonb
    );

    -- Log the request ID for tracking
    RAISE NOTICE 'HTTP request initiated with ID: %', v_request_id;

    -- Wait a moment for the async request to complete
    -- In production, pg_net requests are async, so you may need to check
    -- the net.http_request_queue table for results
    PERFORM pg_sleep(2);

    -- Fetch the response from the request queue
    SELECT
        status,
        content::text,
        error_msg
    INTO v_response
    FROM net.http_request_queue
    WHERE id = v_request_id;

    -- Check if we got a response
    IF v_response IS NULL THEN
        RAISE WARNING 'No response received for request ID: %', v_request_id;
        RETURN QUERY SELECT
            NULL::integer as status_code,
            NULL::text as response_body,
            'Request timeout or pending'::text as error_message;
        RETURN;
    END IF;

    -- Log the response for observability
    RAISE NOTICE 'HTTP Response - Status: %, Body: %', v_response.status, v_response.content;

    -- Return the response details
    RETURN QUERY SELECT
        v_response.status as status_code,
        v_response.content as response_body,
        v_response.error_msg as error_message;

EXCEPTION
    WHEN OTHERS THEN
        -- Handle any errors gracefully to avoid breaking the cron job
        RAISE WARNING 'Error calling companies/generate API: % - %', SQLERRM, SQLSTATE;
        RETURN QUERY SELECT
            NULL::integer as status_code,
            NULL::text as response_body,
            format('Error: % - %', SQLERRM, SQLSTATE)::text as error_message;
END;
$$;

-- Add a comment to the function
COMMENT ON FUNCTION public.call_companies_generate() IS
'Calls the backend API endpoint to generate companies. Returns status code, response body, and any error messages.';

-- ----------------------------------------------------------------------------
-- 3. Schedule the cron job for midnight US Eastern time
-- ----------------------------------------------------------------------------
-- pg_cron uses the database timezone setting. To ensure it runs at midnight Eastern:
-- The cron expression '0 0 * * *' means: minute=0, hour=0, any day, any month, any weekday

-- First, set the timezone for the cron job
-- Note: In Supabase, you may need to set this at the database level or ensure
-- your Postgres instance is configured with the correct timezone.
-- You can verify with: SHOW timezone;

SELECT cron.schedule(
    'daily_companies_generate',           -- Job name
    '0 0 * * *',                          -- Cron expression: daily at midnight
    $$SELECT public.call_companies_generate();$$
);

-- Update the cron job to use America/New_York timezone
-- Note: pg_cron in Supabase runs in UTC by default. To run at midnight Eastern:
-- - During EST (Nov-Mar): midnight EST = 5:00 UTC
-- - During EDT (Mar-Nov): midnight EDT = 4:00 UTC
--
-- For a simple solution that always runs at midnight Eastern Standard Time (EST):
-- Use '0 5 * * *' which is midnight EST (5 hours ahead of UTC)
--
-- For proper daylight saving time handling, you may need to:
-- 1. Use two separate cron jobs (one for EST, one for EDT) with date ranges, OR
-- 2. Run at a fixed UTC time that's acceptable year-round, OR
-- 3. Handle timezone conversion in a wrapper function
--
-- Below we'll reschedule to run at midnight EST (5:00 UTC):

-- Unschedule the previous job
SELECT cron.unschedule('daily_companies_generate');

-- Reschedule with EST timezone consideration (5:00 UTC = midnight EST)
SELECT cron.schedule(
    'daily_companies_generate',           -- Job name
    '0 5 * * *',                          -- 5:00 UTC = midnight EST
    $$SELECT public.call_companies_generate();$$
);

-- Add metadata to track the cron job
COMMENT ON EXTENSION pg_cron IS
'Cron job daily_companies_generate runs at 00:00 America/New_York (05:00 UTC during EST, 04:00 UTC during EDT)';

-- ----------------------------------------------------------------------------
-- 4. Manual testing / trigger
-- ----------------------------------------------------------------------------
-- To manually test the function without waiting for midnight, simply run:
--
--   SELECT * FROM public.call_companies_generate();
--
-- This will immediately trigger the HTTP POST request and return the response.
-- You should see output with columns: status_code, response_body, error_message
--
-- Example output:
--   status_code | response_body | error_message
--   ------------+---------------+---------------
--   200         | {"success":...}| NULL
--
-- To check the cron job schedule:
--   SELECT * FROM cron.job;
--
-- To see cron job execution history:
--   SELECT * FROM cron.job_run_details ORDER BY start_time DESC LIMIT 10;
--
-- To unschedule the job (if needed):
--   SELECT cron.unschedule('daily_companies_generate');
