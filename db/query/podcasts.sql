-- name: CreatePodcast :one
INSERT INTO podcasts (id, library_id, feed_url, title, description, cover_url, author, auto_download, last_checked_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING id, library_id, feed_url, title, description, cover_url, author, auto_download, last_checked_at, created_at, updated_at;

-- name: GetPodcast :one
SELECT id, library_id, feed_url, title, description, cover_url, author, auto_download, last_checked_at, created_at, updated_at
FROM podcasts
WHERE id = ?;

-- name: GetPodcastByFeedURL :one
SELECT id, library_id, feed_url, title, description, cover_url, author, auto_download, last_checked_at, created_at, updated_at
FROM podcasts
WHERE feed_url = ?;

-- name: ListPodcastIDs :many
SELECT id FROM podcasts ORDER BY title COLLATE NOCASE ASC;

-- name: ListPodcastsWithCounts :many
SELECT p.id, p.library_id, p.feed_url, p.title, p.description, p.cover_url, p.author,
       p.auto_download, p.last_checked_at, p.created_at, p.updated_at,
       COUNT(e.id) AS episode_count
FROM podcasts p
LEFT JOIN podcast_episodes e ON e.podcast_id = p.id
GROUP BY p.id
ORDER BY p.title COLLATE NOCASE ASC;

-- name: UpdatePodcast :one
UPDATE podcasts SET
    title = ?, description = ?, cover_url = ?, author = ?,
    auto_download = ?, last_checked_at = ?, updated_at = CURRENT_TIMESTAMP
WHERE id = ?
RETURNING id, library_id, feed_url, title, description, cover_url, author, auto_download, last_checked_at, created_at, updated_at;

-- name: DeletePodcast :exec
DELETE FROM podcasts WHERE id = ?;

-- name: UpsertPodcastEpisode :one
INSERT INTO podcast_episodes (id, podcast_id, guid, title, description, audio_url, duration_sec, published_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(podcast_id, guid) DO UPDATE SET
    title = excluded.title,
    description = excluded.description,
    audio_url = excluded.audio_url,
    duration_sec = excluded.duration_sec,
    published_at = excluded.published_at,
    updated_at = CURRENT_TIMESTAMP
RETURNING id, podcast_id, guid, title, description, audio_url, duration_sec, published_at, downloaded, book_id, created_at, updated_at;

-- name: ListEpisodeGuidsByPodcast :many
SELECT guid FROM podcast_episodes WHERE podcast_id = ?;

-- name: ListEpisodesByPodcast :many
SELECT id, podcast_id, guid, title, description, audio_url, duration_sec, published_at, downloaded, book_id, created_at, updated_at
FROM podcast_episodes
WHERE podcast_id = ?
ORDER BY published_at DESC, created_at DESC;

-- name: GetPodcastEpisode :one
SELECT id, podcast_id, guid, title, description, audio_url, duration_sec, published_at, downloaded, book_id, created_at, updated_at
FROM podcast_episodes
WHERE id = ?;

-- name: UpdateEpisodeDownloaded :exec
UPDATE podcast_episodes SET downloaded = 1, book_id = ?, updated_at = CURRENT_TIMESTAMP
WHERE id = ?;