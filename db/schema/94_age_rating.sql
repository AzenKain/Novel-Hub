CREATE TABLE IF NOT EXISTS content_warnings (
    id TEXT PRIMARY KEY,
    name TEXT UNIQUE NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS book_content_warnings (
    book_id TEXT NOT NULL,
    warning_id TEXT NOT NULL,
    PRIMARY KEY (book_id, warning_id),
    FOREIGN KEY (book_id) REFERENCES books(id) ON DELETE CASCADE,
    FOREIGN KEY (warning_id) REFERENCES content_warnings(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_book_content_warnings_book ON book_content_warnings(book_id);
CREATE INDEX IF NOT EXISTS idx_book_content_warnings_warning ON book_content_warnings(warning_id);

INSERT OR IGNORE INTO content_warnings (id, name, description) VALUES
('cw-violence', 'Violence', 'Depictions of physical violence or combat'),
('cw-nudity', 'Nudity', 'Depictions of partial or full nudity'),
('cw-gore', 'Gore', 'Graphic depictions of blood or severe injuries'),
('cw-language', 'Explicit Language', 'Strong or vulgar language'),
('cw-substance', 'Substance Use', 'Depictions of drug or alcohol use'),
('cw-sexual', 'Sexual Content', 'Explicit or implicit sexual themes');
