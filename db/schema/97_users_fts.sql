-- Trigram index for admin user search: LIKE '%term%' cannot use a B-tree index.
-- Same matches as LIKE ("miya" still finds "Omiya"), 160ms -> 0.4ms on selective terms.
-- ponytail: one concatenated column, not three. Multi-column MATCH needs the
-- `<table> MATCH` form, which sqlc rejects. Costs a bigger index (96MB/200k users).
CREATE VIRTUAL TABLE IF NOT EXISTS fts_users USING fts5(
    user_id UNINDEXED,
    haystack,
    tokenize="trigram"
);

-- Keep haystack identical in all three triggers and the backfill below.
CREATE TRIGGER IF NOT EXISTS t_users_fts_ai AFTER INSERT ON users BEGIN
  INSERT INTO fts_users(user_id, haystack)
  VALUES (new.id, new.id || ' ' || new.email || ' ' || COALESCE(new.full_name, ''));
END;

CREATE TRIGGER IF NOT EXISTS t_users_fts_ad AFTER DELETE ON users BEGIN
  DELETE FROM fts_users WHERE user_id = old.id;
END;

CREATE TRIGGER IF NOT EXISTS t_users_fts_au AFTER UPDATE ON users BEGIN
  DELETE FROM fts_users WHERE user_id = old.id;
  INSERT INTO fts_users(user_id, haystack)
  VALUES (new.id, new.id || ' ' || new.email || ' ' || COALESCE(new.full_name, ''));
END;

INSERT INTO fts_users(user_id, haystack)
SELECT u.id, u.id || ' ' || u.email || ' ' || COALESCE(u.full_name, '')
FROM users u
WHERE NOT EXISTS (SELECT 1 FROM fts_users f WHERE f.user_id = u.id);
