-- name: ListBooksTitleAuthor :many
SELECT b.id, b.title, COALESCE(a.name, '') AS author_name
FROM books b
LEFT JOIN authors a ON a.id = b.author_id
ORDER BY b.title;

-- Plain re-attribute: no UNIQUE constraint on book_id.
-- name: MergeChapters :exec
UPDATE chapters SET book_id = sqlc.arg('target_id') WHERE book_id = sqlc.arg('source_id');

-- name: MergeHighlights :exec
UPDATE highlights SET book_id = sqlc.arg('target_id') WHERE book_id = sqlc.arg('source_id');

-- Composite-PK link tables: INSERT OR IGNORE fires the FTS triggers on
-- 45_metadata_fts.sql (tags/series/publishers/languages) so the target's
-- aggregated fts_metadata row is recomputed; a bare UPDATE would leave it stale.
-- name: MergeBookTags :exec
INSERT OR IGNORE INTO book_tags (book_id, tag_id)
SELECT sqlc.arg('target_id'), bt.tag_id FROM book_tags bt WHERE bt.book_id = sqlc.arg('source_id');

-- name: DeleteBookTags :exec
DELETE FROM book_tags WHERE book_id = sqlc.arg('source_id');

-- name: MergeBookSeries :exec
INSERT OR IGNORE INTO book_series (book_id, series_id, series_index)
SELECT sqlc.arg('target_id'), bs.series_id, bs.series_index FROM book_series bs WHERE bs.book_id = sqlc.arg('source_id');

-- name: DeleteBookSeries :exec
DELETE FROM book_series WHERE book_id = sqlc.arg('source_id');

-- name: MergeBookPublishers :exec
INSERT OR IGNORE INTO book_publishers (book_id, publisher_id)
SELECT sqlc.arg('target_id'), bp.publisher_id FROM book_publishers bp WHERE bp.book_id = sqlc.arg('source_id');

-- name: DeleteBookPublishers :exec
DELETE FROM book_publishers WHERE book_id = sqlc.arg('source_id');

-- name: MergeBookLanguages :exec
INSERT OR IGNORE INTO book_languages (book_id, language_id)
SELECT sqlc.arg('target_id'), bl.language_id FROM book_languages bl WHERE bl.book_id = sqlc.arg('source_id');

-- name: DeleteBookLanguages :exec
DELETE FROM book_languages WHERE book_id = sqlc.arg('source_id');

-- name: MergeBookFilesRest :exec
-- UPDATE in place keeps the file id, so audiobook_chapters/reading_progress
-- file_id references stay valid. Paths are <baseDir>/<bookID>/<filename>; the
-- bookID segment is rewritten. A row whose rewritten path already exists on the
-- target (hash-identical collision) is left behind and dropped by DeleteBookFiles.
UPDATE book_files SET
  book_id = sqlc.arg('target_id'),
  path = substr(book_files.path, 1, instr(book_files.path, sqlc.arg('source_id')) - 1) || sqlc.arg('target_id') || substr(book_files.path, instr(book_files.path, sqlc.arg('source_id')) + length(sqlc.arg('source_id')))
WHERE book_files.book_id = sqlc.arg('source_id')
  AND NOT EXISTS (
    SELECT 1 FROM book_files t
    WHERE t.path = substr(book_files.path, 1, instr(book_files.path, sqlc.arg('source_id')) - 1) || sqlc.arg('target_id') || substr(book_files.path, instr(book_files.path, sqlc.arg('source_id')) + length(sqlc.arg('source_id')))
  );

-- name: MergeFTSChapters :exec
UPDATE fts_chapters SET book_id = sqlc.arg('target_id') WHERE book_id = sqlc.arg('source_id');

-- name: DeleteBookFiles :exec
DELETE FROM book_files WHERE book_id = sqlc.arg('source_id');

-- name: MergeCollectionBooks :exec
INSERT OR IGNORE INTO collection_books (collection_id, book_id, added_at)
SELECT cb.collection_id, sqlc.arg('target_id'), cb.added_at FROM collection_books cb WHERE cb.book_id = sqlc.arg('source_id');

-- name: DeleteCollectionBooks :exec
DELETE FROM collection_books WHERE book_id = sqlc.arg('source_id');

-- name: MergeReadListBooks :exec
INSERT OR IGNORE INTO read_list_books (read_list_id, book_id, position, added_at)
SELECT rlb.read_list_id, sqlc.arg('target_id'), rlb.position, rlb.added_at FROM read_list_books rlb WHERE rlb.book_id = sqlc.arg('source_id');

-- name: DeleteReadListBooks :exec
DELETE FROM read_list_books WHERE book_id = sqlc.arg('source_id');

-- name: MergeBookmarks :exec
INSERT OR IGNORE INTO bookmarks (user_id, book_id, created_at)
SELECT bm.user_id, sqlc.arg('target_id'), bm.created_at FROM bookmarks bm WHERE bm.book_id = sqlc.arg('source_id');

-- name: DeleteBookmarks :exec
DELETE FROM bookmarks WHERE book_id = sqlc.arg('source_id');

-- name: MergeBookReviews :exec
INSERT OR IGNORE INTO book_reviews (user_id, book_id, rating, review, created_at, updated_at)
SELECT br.user_id, sqlc.arg('target_id'), br.rating, br.review, br.created_at, br.updated_at FROM book_reviews br WHERE br.book_id = sqlc.arg('source_id');

-- name: DeleteBookReviews :exec
DELETE FROM book_reviews WHERE book_id = sqlc.arg('source_id');

-- name: MergeBookContentWarnings :exec
INSERT OR IGNORE INTO book_content_warnings (book_id, warning_id)
SELECT sqlc.arg('target_id'), bcw.warning_id FROM book_content_warnings bcw WHERE bcw.book_id = sqlc.arg('source_id');

-- name: DeleteBookContentWarnings :exec
DELETE FROM book_content_warnings WHERE book_id = sqlc.arg('source_id');

-- name: MergeReadingProgress :exec
INSERT OR IGNORE INTO reading_progress (user_id, book_id, file_id, chapter_ref, chapter_title, chapter_index, progress_percent, location_cfi, location_type, opened_count, qualified_read_count, last_opened_at, last_counted_at, updated_at)
SELECT rp.user_id, sqlc.arg('target_id'), rp.file_id, rp.chapter_ref, rp.chapter_title, rp.chapter_index, rp.progress_percent, rp.location_cfi, rp.location_type, rp.opened_count, rp.qualified_read_count, rp.last_opened_at, rp.last_counted_at, rp.updated_at
FROM reading_progress rp WHERE rp.book_id = sqlc.arg('source_id');

-- name: DeleteReadingProgress :exec
DELETE FROM reading_progress WHERE book_id = sqlc.arg('source_id');

-- name: MergeBookTrackerMappings :exec
INSERT OR IGNORE INTO book_tracker_mappings (id, user_id, book_id, provider, external_series_id, created_at)
SELECT btm.id, btm.user_id, sqlc.arg('target_id'), btm.provider, btm.external_series_id, btm.created_at FROM book_tracker_mappings btm WHERE btm.book_id = sqlc.arg('source_id');

-- name: DeleteBookTrackerMappings :exec
DELETE FROM book_tracker_mappings WHERE book_id = sqlc.arg('source_id');

-- name: MergeKoboSyncedBooks :exec
INSERT OR IGNORE INTO kobo_synced_books (user_id, book_id, synced_at)
SELECT ksb.user_id, sqlc.arg('target_id'), ksb.synced_at FROM kobo_synced_books ksb WHERE ksb.book_id = sqlc.arg('source_id');

-- name: DeleteKoboSyncedBooksByBook :exec
DELETE FROM kobo_synced_books WHERE book_id = sqlc.arg('source_id');

-- name: MergeBookShareEvents :exec
INSERT OR IGNORE INTO book_share_events (book_id, actor_key, window_bucket, created_at)
SELECT sqlc.arg('target_id'), bse.actor_key, bse.window_bucket, bse.created_at FROM book_share_events bse WHERE bse.book_id = sqlc.arg('source_id');

-- name: DeleteBookShareEvents :exec
DELETE FROM book_share_events WHERE book_id = sqlc.arg('source_id');

-- name: MergeAudiobookChapters :exec
INSERT OR IGNORE INTO audiobook_chapters (id, book_id, file_id, chapter_index, title, start_sec, end_sec, created_at, updated_at)
SELECT ac.id, sqlc.arg('target_id'), ac.file_id, ac.chapter_index, ac.title, ac.start_sec, ac.end_sec, ac.created_at, ac.updated_at
FROM audiobook_chapters ac WHERE ac.book_id = sqlc.arg('source_id');

-- name: DeleteAudiobookChapters :exec
DELETE FROM audiobook_chapters WHERE book_id = sqlc.arg('source_id');

-- Reading sessions: rows without a same-date target row move over, the rest fold
-- their counters into the target's same-date row, then all source rows are dropped.
-- name: MergeReadingSessionsRest :exec
UPDATE reading_sessions SET book_id = sqlc.arg('target_id')
WHERE reading_sessions.book_id = sqlc.arg('source_id')
  AND NOT EXISTS (
    SELECT 1 FROM reading_sessions t
    WHERE t.user_id = reading_sessions.user_id
      AND t.book_id = sqlc.arg('target_id')
      AND t.session_date = reading_sessions.session_date
  );

-- name: FoldReadingSessions :exec
UPDATE reading_sessions SET
  duration_seconds = duration_seconds + (
    SELECT COALESCE(SUM(s.duration_seconds), 0) FROM reading_sessions s
    WHERE s.book_id = sqlc.arg('source_id')
      AND s.user_id = reading_sessions.user_id
      AND s.session_date = reading_sessions.session_date
  ),
  words_read = words_read + (
    SELECT COALESCE(SUM(s.words_read), 0) FROM reading_sessions s
    WHERE s.book_id = sqlc.arg('source_id')
      AND s.user_id = reading_sessions.user_id
      AND s.session_date = reading_sessions.session_date
  ),
  updated_at = CURRENT_TIMESTAMP
WHERE reading_sessions.book_id = sqlc.arg('target_id')
  AND EXISTS (
    SELECT 1 FROM reading_sessions s
    WHERE s.book_id = sqlc.arg('source_id')
      AND s.user_id = reading_sessions.user_id
      AND s.session_date = reading_sessions.session_date
  );

-- name: DeleteReadingSessions :exec
DELETE FROM reading_sessions WHERE book_id = sqlc.arg('source_id');

-- Per-book stats: ensure the target row exists, fold source counters in, drop source.
-- name: EnsureBookReadStats :exec
INSERT INTO book_read_stats (book_id, total_open_count, qualified_read_count)
VALUES (sqlc.arg('target_id'), 0, 0)
ON CONFLICT (book_id) DO NOTHING;

-- name: MergeBookReadStats :exec
UPDATE book_read_stats SET
  total_open_count = total_open_count + COALESCE((SELECT src.total_open_count FROM book_read_stats src WHERE src.book_id = sqlc.arg('source_id')), 0),
  qualified_read_count = qualified_read_count + COALESCE((SELECT src.qualified_read_count FROM book_read_stats src WHERE src.book_id = sqlc.arg('source_id')), 0),
  last_opened_at = CASE
    WHEN (SELECT src.last_opened_at FROM book_read_stats src WHERE src.book_id = sqlc.arg('source_id')) IS NOT NULL
     AND (SELECT src.last_opened_at FROM book_read_stats src WHERE src.book_id = sqlc.arg('source_id')) > COALESCE(last_opened_at, '1970-01-01')
    THEN (SELECT src.last_opened_at FROM book_read_stats src WHERE src.book_id = sqlc.arg('source_id'))
    ELSE last_opened_at END,
  last_counted_at = CASE
    WHEN (SELECT src.last_counted_at FROM book_read_stats src WHERE src.book_id = sqlc.arg('source_id')) IS NOT NULL
     AND (SELECT src.last_counted_at FROM book_read_stats src WHERE src.book_id = sqlc.arg('source_id')) > COALESCE(last_counted_at, '1970-01-01')
    THEN (SELECT src.last_counted_at FROM book_read_stats src WHERE src.book_id = sqlc.arg('source_id'))
    ELSE last_counted_at END,
  updated_at = CURRENT_TIMESTAMP
WHERE book_read_stats.book_id = sqlc.arg('target_id');

-- name: DeleteBookReadStats :exec
DELETE FROM book_read_stats WHERE book_id = sqlc.arg('source_id');

-- name: EnsureBookDownloadStats :exec
INSERT INTO book_download_stats (book_id, total_download_count)
VALUES (sqlc.arg('target_id'), 0)
ON CONFLICT (book_id) DO NOTHING;

-- name: MergeBookDownloadStats :exec
UPDATE book_download_stats SET
  total_download_count = total_download_count + COALESCE((SELECT src.total_download_count FROM book_download_stats src WHERE src.book_id = sqlc.arg('source_id')), 0),
  last_downloaded_at = CASE
    WHEN (SELECT src.last_downloaded_at FROM book_download_stats src WHERE src.book_id = sqlc.arg('source_id')) IS NOT NULL
     AND (SELECT src.last_downloaded_at FROM book_download_stats src WHERE src.book_id = sqlc.arg('source_id')) > COALESCE(last_downloaded_at, '1970-01-01')
    THEN (SELECT src.last_downloaded_at FROM book_download_stats src WHERE src.book_id = sqlc.arg('source_id'))
    ELSE last_downloaded_at END,
  updated_at = CURRENT_TIMESTAMP
WHERE book_download_stats.book_id = sqlc.arg('target_id');

-- name: DeleteBookDownloadStats :exec
DELETE FROM book_download_stats WHERE book_id = sqlc.arg('source_id');

-- name: EnsureBookSocialStats :exec
INSERT INTO book_social_stats (book_id, bookmark_count, rating_count, average_rating, share_count)
VALUES (sqlc.arg('target_id'), 0, 0, 0, 0)
ON CONFLICT (book_id) DO NOTHING;

-- name: MergeBookSocialStats :exec
UPDATE book_social_stats SET
  bookmark_count = bookmark_count + COALESCE((SELECT src.bookmark_count FROM book_social_stats src WHERE src.book_id = sqlc.arg('source_id')), 0),
  share_count = share_count + COALESCE((SELECT src.share_count FROM book_social_stats src WHERE src.book_id = sqlc.arg('source_id')), 0),
  rating_count = rating_count + COALESCE((SELECT src.rating_count FROM book_social_stats src WHERE src.book_id = sqlc.arg('source_id')), 0),
  average_rating = CASE
    WHEN rating_count + COALESCE((SELECT src.rating_count FROM book_social_stats src WHERE src.book_id = sqlc.arg('source_id')), 0) = 0 THEN 0
    ELSE (average_rating * rating_count + COALESCE((SELECT src.average_rating FROM book_social_stats src WHERE src.book_id = sqlc.arg('source_id')), 0) * COALESCE((SELECT src.rating_count FROM book_social_stats src WHERE src.book_id = sqlc.arg('source_id')), 0))
         / (rating_count + COALESCE((SELECT src.rating_count FROM book_social_stats src WHERE src.book_id = sqlc.arg('source_id')), 0))
  END,
  updated_at = CURRENT_TIMESTAMP
WHERE book_social_stats.book_id = sqlc.arg('target_id');

-- name: DeleteBookSocialStats :exec
DELETE FROM book_social_stats WHERE book_id = sqlc.arg('source_id');