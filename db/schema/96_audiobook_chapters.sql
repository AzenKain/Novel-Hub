-- Audiobook chapter markers, one row per chapter of an audiobook. Chapters
-- are either hand-edited, imported from Audnexus (by ASIN), or auto-written
-- by the merge_audio job (one chapter per merged source track).
CREATE TABLE IF NOT EXISTS audiobook_chapters (
    id TEXT PRIMARY KEY,
    book_id TEXT NOT NULL,
    file_id TEXT, -- optional: which book_file the chapter belongs to (pre-merge)
    chapter_index INTEGER NOT NULL,
    title TEXT NOT NULL,
    start_sec REAL NOT NULL DEFAULT 0,
    end_sec REAL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(book_id, chapter_index),
    FOREIGN KEY (book_id) REFERENCES books(id) ON DELETE CASCADE,
    FOREIGN KEY (file_id) REFERENCES book_files(id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_audiobook_chapters_book ON audiobook_chapters(book_id, chapter_index);