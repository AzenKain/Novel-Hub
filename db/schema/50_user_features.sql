-- Table for user collections
CREATE TABLE IF NOT EXISTS collections (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    name TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- Table for linking books to collections
CREATE TABLE IF NOT EXISTS collection_books (
    collection_id TEXT NOT NULL,
    book_id TEXT NOT NULL,
    added_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (collection_id, book_id),
    FOREIGN KEY (collection_id) REFERENCES collections(id) ON DELETE CASCADE,
    FOREIGN KEY (book_id) REFERENCES books(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_collections_user_id ON collections(user_id);

CREATE TABLE IF NOT EXISTS read_lists (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    name TEXT NOT NULL,
    description TEXT,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- position is deliberately not UNIQUE: swapping two adjacent entries has to pass through a state
-- where both hold the same number, and a UNIQUE constraint would abort mid-transaction.
CREATE TABLE IF NOT EXISTS read_list_books (
    read_list_id TEXT NOT NULL,
    book_id TEXT NOT NULL,
    position INTEGER NOT NULL,
    added_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (read_list_id, book_id),
    FOREIGN KEY (read_list_id) REFERENCES read_lists(id) ON DELETE CASCADE,
    FOREIGN KEY (book_id) REFERENCES books(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_read_lists_user_created ON read_lists(user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_read_list_books_position ON read_list_books(read_list_id, position);
CREATE INDEX IF NOT EXISTS idx_read_list_books_book ON read_list_books(book_id);

CREATE TABLE IF NOT EXISTS bookmarks (
    user_id TEXT NOT NULL,
    book_id TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (user_id, book_id),
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (book_id) REFERENCES books(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS book_reviews (
    user_id TEXT NOT NULL,
    book_id TEXT NOT NULL,
    rating INTEGER NOT NULL CHECK (rating BETWEEN 1 AND 5),
    review TEXT,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (user_id, book_id),
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (book_id) REFERENCES books(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_bookmarks_user_time ON bookmarks(user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_book_reviews_book ON book_reviews(book_id, updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_book_reviews_rating ON book_reviews(book_id, rating);

CREATE TABLE IF NOT EXISTS book_social_stats (
    book_id TEXT PRIMARY KEY,
    bookmark_count INTEGER NOT NULL DEFAULT 0,
    rating_count INTEGER NOT NULL DEFAULT 0,
    average_rating REAL NOT NULL DEFAULT 0,
    share_count INTEGER NOT NULL DEFAULT 0,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (book_id) REFERENCES books(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_book_social_stats_rating ON book_social_stats(average_rating DESC, rating_count DESC, updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_book_social_stats_bookmarks ON book_social_stats(bookmark_count DESC, updated_at DESC);

CREATE TABLE IF NOT EXISTS book_share_events (
    book_id TEXT NOT NULL,
    actor_key TEXT NOT NULL,
    window_bucket INTEGER NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (book_id, actor_key, window_bucket),
    FOREIGN KEY (book_id) REFERENCES books(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_book_share_events_book_time ON book_share_events(book_id, created_at DESC);

INSERT INTO book_social_stats (
    book_id,
    bookmark_count,
    rating_count,
    average_rating,
    share_count,
    updated_at
)
SELECT
    b.id,
    COALESCE(bm.bookmark_count, 0),
    COALESCE(rv.rating_count, 0),
    COALESCE(rv.average_rating, 0),
    COALESCE(existing.share_count, 0),
    CURRENT_TIMESTAMP
FROM books b
LEFT JOIN (
    SELECT book_id, COUNT(*) AS bookmark_count
    FROM bookmarks
    GROUP BY book_id
) bm ON bm.book_id = b.id
LEFT JOIN (
    SELECT book_id, COUNT(*) AS rating_count, AVG(rating) AS average_rating
    FROM book_reviews
    GROUP BY book_id
) rv ON rv.book_id = b.id
LEFT JOIN book_social_stats existing ON existing.book_id = b.id
WHERE bm.bookmark_count IS NOT NULL OR rv.rating_count IS NOT NULL OR existing.book_id IS NOT NULL
ON CONFLICT(book_id) DO UPDATE SET
    bookmark_count = excluded.bookmark_count,
    rating_count = excluded.rating_count,
    average_rating = excluded.average_rating,
    share_count = excluded.share_count,
    updated_at = CURRENT_TIMESTAMP;
