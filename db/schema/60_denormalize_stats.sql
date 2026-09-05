CREATE INDEX IF NOT EXISTS idx_books_popular ON books(download_count DESC, average_rating DESC, read_count DESC, created_at DESC);

CREATE TRIGGER IF NOT EXISTS t_bds_ai AFTER INSERT ON book_download_stats BEGIN
  UPDATE books SET download_count = new.total_download_count WHERE id = new.book_id;
END;
CREATE TRIGGER IF NOT EXISTS t_bds_au AFTER UPDATE OF total_download_count ON book_download_stats BEGIN
  UPDATE books SET download_count = new.total_download_count WHERE id = new.book_id;
END;

CREATE TRIGGER IF NOT EXISTS t_bss_ai AFTER INSERT ON book_social_stats BEGIN
  UPDATE books SET average_rating = new.average_rating, rating_count = new.rating_count WHERE id = new.book_id;
END;
CREATE TRIGGER IF NOT EXISTS t_bss_au AFTER UPDATE ON book_social_stats BEGIN
  UPDATE books SET average_rating = new.average_rating, rating_count = new.rating_count WHERE id = new.book_id;
END;

CREATE TRIGGER IF NOT EXISTS t_brs_ai AFTER INSERT ON book_read_stats BEGIN
  UPDATE books SET read_count = new.qualified_read_count, open_count = new.total_open_count WHERE id = new.book_id;
END;
CREATE TRIGGER IF NOT EXISTS t_brs_au AFTER UPDATE ON book_read_stats BEGIN
  UPDATE books SET read_count = new.qualified_read_count, open_count = new.total_open_count WHERE id = new.book_id;
END;

UPDATE books SET download_count = COALESCE((SELECT total_download_count FROM book_download_stats WHERE book_id = books.id), 0);
UPDATE books SET average_rating = COALESCE((SELECT average_rating FROM book_social_stats WHERE book_id = books.id), 0.0);
UPDATE books SET rating_count = COALESCE((SELECT rating_count FROM book_social_stats WHERE book_id = books.id), 0);
UPDATE books SET read_count = COALESCE((SELECT qualified_read_count FROM book_read_stats WHERE book_id = books.id), 0);
UPDATE books SET open_count = COALESCE((SELECT total_open_count FROM book_read_stats WHERE book_id = books.id), 0);

