CREATE VIRTUAL TABLE IF NOT EXISTS fts_metadata USING fts5(
    book_id UNINDEXED,
    title,
    author,
    description,
    tags,
    series,
    publishers,
    languages,
    tokenize="unicode61 remove_diacritics 1"
);

-- When a book is created, insert it into FTS.
CREATE TRIGGER IF NOT EXISTS t_books_ai AFTER INSERT ON books BEGIN
  INSERT INTO fts_metadata(rowid, book_id, title, author, description, tags, series, publishers, languages)
  VALUES (
    new.rowid,
    new.id,
    new.title,
    (SELECT name FROM authors WHERE id = new.author_id),
    new.description,
    (SELECT group_concat(t.name, ' ') FROM book_tags bt JOIN tags t ON bt.tag_id = t.id WHERE bt.book_id = new.id),
    (SELECT group_concat(s.name, ' ') FROM book_series bs JOIN series s ON bs.series_id = s.id WHERE bs.book_id = new.id),
    (SELECT group_concat(p.name, ' ') FROM book_publishers bp JOIN publishers p ON bp.publisher_id = p.id WHERE bp.book_id = new.id),
    (SELECT group_concat(l.name, ' ') FROM book_languages bl JOIN languages l ON bl.language_id = l.id WHERE bl.book_id = new.id)
  );
END;

-- When a book is updated, update its core fields in FTS.
CREATE TRIGGER IF NOT EXISTS t_books_au AFTER UPDATE ON books BEGIN
  UPDATE fts_metadata SET
    title = new.title,
    author = (SELECT name FROM authors WHERE id = new.author_id),
    description = new.description
  WHERE rowid = old.rowid;
END;

-- When a book is deleted, remove it from FTS.
CREATE TRIGGER IF NOT EXISTS t_books_ad AFTER DELETE ON books BEGIN
  DELETE FROM fts_metadata WHERE rowid = old.rowid;
END;

-- TRIGGERS for TAGS
CREATE TRIGGER IF NOT EXISTS t_book_tags_ai AFTER INSERT ON book_tags BEGIN
  UPDATE fts_metadata SET tags = (SELECT group_concat(t.name, ' ') FROM book_tags bt JOIN tags t ON bt.tag_id = t.id WHERE bt.book_id = new.book_id) WHERE rowid = (SELECT rowid FROM books WHERE id = new.book_id);
END;
CREATE TRIGGER IF NOT EXISTS t_book_tags_ad AFTER DELETE ON book_tags BEGIN
  UPDATE fts_metadata SET tags = (SELECT group_concat(t.name, ' ') FROM book_tags bt JOIN tags t ON bt.tag_id = t.id WHERE bt.book_id = old.book_id) WHERE rowid = (SELECT rowid FROM books WHERE id = old.book_id);
END;

-- TRIGGERS for SERIES
CREATE TRIGGER IF NOT EXISTS t_book_series_ai AFTER INSERT ON book_series BEGIN
  UPDATE fts_metadata SET series = (SELECT group_concat(s.name, ' ') FROM book_series bs JOIN series s ON bs.series_id = s.id WHERE bs.book_id = new.book_id) WHERE rowid = (SELECT rowid FROM books WHERE id = new.book_id);
END;
CREATE TRIGGER IF NOT EXISTS t_book_series_ad AFTER DELETE ON book_series BEGIN
  UPDATE fts_metadata SET series = (SELECT group_concat(s.name, ' ') FROM book_series bs JOIN series s ON bs.series_id = s.id WHERE bs.book_id = old.book_id) WHERE rowid = (SELECT rowid FROM books WHERE id = old.book_id);
END;

-- TRIGGERS for PUBLISHERS
CREATE TRIGGER IF NOT EXISTS t_book_publishers_ai AFTER INSERT ON book_publishers BEGIN
  UPDATE fts_metadata SET publishers = (SELECT group_concat(p.name, ' ') FROM book_publishers bp JOIN publishers p ON bp.publisher_id = p.id WHERE bp.book_id = new.book_id) WHERE rowid = (SELECT rowid FROM books WHERE id = new.book_id);
END;
CREATE TRIGGER IF NOT EXISTS t_book_publishers_ad AFTER DELETE ON book_publishers BEGIN
  UPDATE fts_metadata SET publishers = (SELECT group_concat(p.name, ' ') FROM book_publishers bp JOIN publishers p ON bp.publisher_id = p.id WHERE bp.book_id = old.book_id) WHERE rowid = (SELECT rowid FROM books WHERE id = old.book_id);
END;

-- TRIGGERS for LANGUAGES
CREATE TRIGGER IF NOT EXISTS t_book_languages_ai AFTER INSERT ON book_languages BEGIN
  UPDATE fts_metadata SET languages = (SELECT group_concat(l.name, ' ') FROM book_languages bl JOIN languages l ON bl.language_id = l.id WHERE bl.book_id = new.book_id) WHERE rowid = (SELECT rowid FROM books WHERE id = new.book_id);
END;
CREATE TRIGGER IF NOT EXISTS t_book_languages_ad AFTER DELETE ON book_languages BEGIN
  UPDATE fts_metadata SET languages = (SELECT group_concat(l.name, ' ') FROM book_languages bl JOIN languages l ON bl.language_id = l.id WHERE bl.book_id = old.book_id) WHERE rowid = (SELECT rowid FROM books WHERE id = old.book_id);
END;

-- TRIGGER for AUTHOR Name updates
CREATE TRIGGER IF NOT EXISTS t_authors_au AFTER UPDATE OF name ON authors BEGIN
  UPDATE fts_metadata SET author = new.name WHERE rowid IN (SELECT rowid FROM books WHERE author_id = new.id);
END;

-- Backfill existing data
INSERT INTO fts_metadata(rowid, book_id, title, author, description, tags, series, publishers, languages)
SELECT 
  b.rowid,
  b.id,
  b.title,
  (SELECT name FROM authors WHERE id = b.author_id),
  b.description,
  (SELECT group_concat(t.name, ' ') FROM book_tags bt JOIN tags t ON bt.tag_id = t.id WHERE bt.book_id = b.id),
  (SELECT group_concat(s.name, ' ') FROM book_series bs JOIN series s ON bs.series_id = s.id WHERE bs.book_id = b.id),
  (SELECT group_concat(p.name, ' ') FROM book_publishers bp JOIN publishers p ON bp.publisher_id = p.id WHERE bp.book_id = b.id),
  (SELECT group_concat(l.name, ' ') FROM book_languages bl JOIN languages l ON bl.language_id = l.id WHERE bl.book_id = b.id)
FROM books b
WHERE NOT EXISTS (SELECT 1 FROM fts_metadata WHERE rowid = b.rowid);
