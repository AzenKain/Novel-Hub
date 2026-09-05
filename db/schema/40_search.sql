CREATE VIRTUAL TABLE IF NOT EXISTS fts_chapters USING fts5(
    book_id UNINDEXED,
    chapter_id UNINDEXED,
    title,
    content
);
