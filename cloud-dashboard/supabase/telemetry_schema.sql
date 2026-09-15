-- Supabase Schema for AI Tutor Minimal App Telemetry (Tier 1 Heartbeat)
-- Tracks lightweight anonymous application start pings to understand active usage, OS, and version distribution.

CREATE TABLE IF NOT EXISTS public.app_telemetry (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    installation_id UUID NOT NULL,
    event TEXT NOT NULL,         -- 'app_started', 'app_updated', 'app_crashed'
    app_version TEXT NOT NULL,   -- e.g. '1.4.1'
    platform TEXT NOT NULL,      -- 'windows', 'darwin', 'linux'
    architecture TEXT NOT NULL,  -- 'amd64', 'arm64'
    created_at TIMESTAMP WITH TIME ZONE DEFAULT timezone('utc'::text, now()) NOT NULL
);

-- Enable RLS on telemetry
ALTER TABLE public.app_telemetry ENABLE ROW LEVEL SECURITY;

-- 1. Allow anonymous clients to insert launch pings
CREATE POLICY "Allow anonymous inserts app_telemetry" 
ON public.app_telemetry 
FOR INSERT 
TO anon 
WITH CHECK (true);

-- 2. Deny direct client reads/updates/deletes for public anon
CREATE POLICY "Deny anonymous reads app_telemetry" 
ON public.app_telemetry 
FOR SELECT 
TO anon 
USING (false);

CREATE POLICY "Deny anonymous updates app_telemetry" 
ON public.app_telemetry 
FOR UPDATE 
TO anon 
USING (false)
WITH CHECK (false);

CREATE POLICY "Deny anonymous deletes app_telemetry" 
ON public.app_telemetry 
FOR DELETE 
TO anon 
USING (false);

-- Indexes for lightning fast dashboard aggregation
CREATE INDEX IF NOT EXISTS idx_app_telemetry_created_at ON public.app_telemetry(created_at);
CREATE INDEX IF NOT EXISTS idx_app_telemetry_installation_id ON public.app_telemetry(installation_id);
CREATE INDEX IF NOT EXISTS idx_app_telemetry_platform ON public.app_telemetry(platform);
CREATE INDEX IF NOT EXISTS idx_app_telemetry_version ON public.app_telemetry(app_version);

-- 3. Automated Purge (90-day retention) via pg_cron
DO $outer$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_extension WHERE extname = 'pg_cron') THEN
    PERFORM cron.schedule(
      'purge-old-app-telemetry',
      '0 1 * * *', -- Daily at 1:00 AM UTC
      $job$
      DELETE FROM public.app_telemetry
      WHERE created_at < NOW() - INTERVAL '90 days';
      $job$
    );
  END IF;
END;
$outer$;
