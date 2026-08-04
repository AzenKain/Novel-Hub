-- Performance indexes for FKs, filtering and sorting
CREATE INDEX IF NOT EXISTS idx_books_library_id ON books(library_id);
CREATE INDEX IF NOT EXISTS idx_books_library_created ON books(library_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_books_status_created ON books(status, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_book_tags_tag ON book_tags(tag_id);
CREATE INDEX IF NOT EXISTS idx_book_series_series ON book_series(series_id);
CREATE INDEX IF NOT EXISTS idx_book_publishers_pub ON book_publishers(publisher_id);
CREATE INDEX IF NOT EXISTS idx_book_languages_lang ON book_languages(language_id);
CREATE INDEX IF NOT EXISTS idx_collection_books_book ON collection_books(book_id);
CREATE INDEX IF NOT EXISTS idx_collection_books_coll_added ON collection_books(collection_id, added_at DESC);

CREATE INDEX IF NOT EXISTS idx_reading_progress_book ON reading_progress(book_id);
CREATE INDEX IF NOT EXISTS idx_reading_progress_user_updated ON reading_progress(user_id, updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_bookmarks_user_created ON bookmarks(user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_bookmarks_book ON bookmarks(book_id);
CREATE INDEX IF NOT EXISTS idx_collections_user_created ON collections(user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_book_reviews_book_updated ON book_reviews(book_id, updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_book_reviews_user ON book_reviews(user_id);
CREATE INDEX IF NOT EXISTS idx_book_share_events_book ON book_share_events(book_id);
CREATE INDEX IF NOT EXISTS idx_highlights_user_book ON highlights(user_id, book_id);
CREATE INDEX IF NOT EXISTS idx_book_files_book ON book_files(book_id);
CREATE INDEX IF NOT EXISTS idx_chapters_book_index ON chapters(book_id, chapter_index ASC);
CREATE INDEX IF NOT EXISTS idx_role_permissions_role ON role_permissions(role_id);
CREATE INDEX IF NOT EXISTS idx_role_permissions_perm ON role_permissions(permission_key);
