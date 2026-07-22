-- Composite and single column performance indexes for external trackers
CREATE INDEX IF NOT EXISTS idx_user_trackers_user_provider ON user_trackers(user_id, provider);
CREATE INDEX IF NOT EXISTS idx_btm_book_provider ON book_tracker_mappings(book_id, provider);
CREATE INDEX IF NOT EXISTS idx_btm_external_id ON book_tracker_mappings(external_series_id, provider);
