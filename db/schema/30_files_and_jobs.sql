-- Table for book_files (physical files belonging to a book)
CREATE TABLE IF NOT EXISTS book_files (
    id TEXT PRIMARY KEY,
    book_id TEXT NOT NULL,
    path TEXT NOT NULL, -- Relative or absolute path
    format TEXT NOT NULL, -- e.g. epub, pdf
    size_bytes INTEGER NOT NULL,
    mod_time DATETIME NOT NULL,
    hash TEXT, -- SHA-256 for deduplication
    state TEXT DEFAULT 'referenced', -- referenced or managed
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(path),
    FOREIGN KEY (book_id) REFERENCES books(id) ON DELETE CASCADE
);

-- Table for background jobs
CREATE TABLE IF NOT EXISTS jobs (
    id TEXT PRIMARY KEY,
    type TEXT NOT NULL, -- e.g. scan, metadata
    status TEXT DEFAULT 'pending', -- pending, running, completed, failed
    progress INTEGER DEFAULT 0,
    total INTEGER DEFAULT 0,
    error_msg TEXT,
    payload_json TEXT, -- Additional context for the job
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_book_files_book ON book_files(book_id);
CREATE INDEX IF NOT EXISTS idx_book_files_hash ON book_files(hash);
CREATE INDEX IF NOT EXISTS idx_jobs_status ON jobs(status);
CREATE INDEX IF NOT EXISTS idx_jobs_type ON jobs(type);
