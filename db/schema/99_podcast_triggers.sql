CREATE TRIGGER IF NOT EXISTS trg_books_deleted_update_podcast_episodes
AFTER DELETE ON books
FOR EACH ROW
BEGIN
    UPDATE podcast_episodes
    SET downloaded = 0, book_id = NULL
    WHERE book_id = OLD.id;
END;
