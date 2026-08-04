-- Job listing sorts created_at DESC with LIMIT; the `('' = ? OR col = ?)` filter guards
-- stop the planner using a status index, so this lets it stop at LIMIT (53ms -> 0.001ms
-- at 500k rows). ponytail: no (status, created_at) composite, measured unused.
CREATE INDEX IF NOT EXISTS idx_jobs_created ON jobs(created_at DESC);
