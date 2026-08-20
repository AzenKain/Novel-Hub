-- name: ListSoundscapeIDsAccessible :many
SELECT id FROM soundscapes
WHERE (is_system = 1 OR user_id = sqlc.arg('user_id'))
  AND (
      updated_at <= COALESCE(CAST(sqlc.narg('cursor_updated_at') AS TEXT), '9999-12-31 23:59:59')
      AND (sqlc.narg('cursor_updated_at') IS NULL OR updated_at < CAST(sqlc.narg('cursor_updated_at') AS TEXT) OR id < sqlc.narg('cursor_id'))
  )
ORDER BY updated_at DESC, id DESC
LIMIT sqlc.arg('limit');

-- name: ListSystemSoundscapeIDs :many
SELECT id FROM soundscapes
WHERE is_system = 1
  AND (
      updated_at <= COALESCE(CAST(sqlc.narg('cursor_updated_at') AS TEXT), '9999-12-31 23:59:59')
      AND (sqlc.narg('cursor_updated_at') IS NULL OR updated_at < CAST(sqlc.narg('cursor_updated_at') AS TEXT) OR id < sqlc.narg('cursor_id'))
  )
ORDER BY updated_at DESC, id DESC
LIMIT sqlc.arg('limit');

-- name: GetSoundscapeByID :one
SELECT id, user_id, name, category, file_path, icon, volume, is_system, created_at, updated_at
FROM soundscapes
WHERE id = ? LIMIT 1;

-- name: GetSoundscapesByIDs :many
SELECT id, user_id, name, category, file_path, icon, volume, is_system, created_at, updated_at
FROM soundscapes
WHERE id IN (sqlc.slice('ids'));

-- name: CreateSoundscape :one
INSERT INTO soundscapes (
    id, user_id, name, category, file_path, icon, volume, is_system, created_at, updated_at
) VALUES (
    ?, ?, ?, ?, ?, ?, ?, ?, datetime('now'), datetime('now')
)
RETURNING id, user_id, name, category, file_path, icon, volume, is_system, created_at, updated_at;

-- name: UpdateSoundscape :one
UPDATE soundscapes
SET name = ?, category = ?, icon = ?, volume = ?, updated_at = datetime('now')
WHERE id = ?
RETURNING id, user_id, name, category, file_path, icon, volume, is_system, created_at, updated_at;

-- name: DeleteSoundscape :exec
DELETE FROM soundscapes WHERE id = ?;


-- name: ListCustomFontIDsAccessible :many
SELECT id FROM custom_fonts
WHERE (is_system = 1 OR user_id = sqlc.arg('user_id'))
  AND (
      updated_at <= COALESCE(CAST(sqlc.narg('cursor_updated_at') AS TEXT), '9999-12-31 23:59:59')
      AND (sqlc.narg('cursor_updated_at') IS NULL OR updated_at < CAST(sqlc.narg('cursor_updated_at') AS TEXT) OR id < sqlc.narg('cursor_id'))
  )
ORDER BY updated_at DESC, id DESC
LIMIT sqlc.arg('limit');

-- name: ListSystemCustomFontIDs :many
SELECT id FROM custom_fonts
WHERE is_system = 1
  AND (
      updated_at <= COALESCE(CAST(sqlc.narg('cursor_updated_at') AS TEXT), '9999-12-31 23:59:59')
      AND (sqlc.narg('cursor_updated_at') IS NULL OR updated_at < CAST(sqlc.narg('cursor_updated_at') AS TEXT) OR id < sqlc.narg('cursor_id'))
  )
ORDER BY updated_at DESC, id DESC
LIMIT sqlc.arg('limit');

-- name: GetCustomFontByID :one
SELECT id, user_id, name, font_family, source_type, file_path, font_url, is_system, created_at, updated_at
FROM custom_fonts
WHERE id = ? LIMIT 1;

-- name: GetCustomFontsByIDs :many
SELECT id, user_id, name, font_family, source_type, file_path, font_url, is_system, created_at, updated_at
FROM custom_fonts
WHERE id IN (sqlc.slice('ids'));

-- name: CreateCustomFont :one
INSERT INTO custom_fonts (
    id, user_id, name, font_family, source_type, file_path, font_url, is_system, created_at, updated_at
) VALUES (
    ?, ?, ?, ?, ?, ?, ?, ?, datetime('now'), datetime('now')
)
RETURNING id, user_id, name, font_family, source_type, file_path, font_url, is_system, created_at, updated_at;

-- name: DeleteCustomFont :exec
DELETE FROM custom_fonts WHERE id = ?;


-- name: ListCustomThemeIDsAccessible :many
SELECT id FROM custom_themes
WHERE (is_system = 1 OR user_id = sqlc.arg('user_id'))
  AND (
      updated_at <= COALESCE(CAST(sqlc.narg('cursor_updated_at') AS TEXT), '9999-12-31 23:59:59')
      AND (sqlc.narg('cursor_updated_at') IS NULL OR updated_at < CAST(sqlc.narg('cursor_updated_at') AS TEXT) OR id < sqlc.narg('cursor_id'))
  )
ORDER BY updated_at DESC, id DESC
LIMIT sqlc.arg('limit');

-- name: ListSystemCustomThemeIDs :many
SELECT id FROM custom_themes
WHERE is_system = 1
  AND (
      updated_at <= COALESCE(CAST(sqlc.narg('cursor_updated_at') AS TEXT), '9999-12-31 23:59:59')
      AND (sqlc.narg('cursor_updated_at') IS NULL OR updated_at < CAST(sqlc.narg('cursor_updated_at') AS TEXT) OR id < sqlc.narg('cursor_id'))
  )
ORDER BY updated_at DESC, id DESC
LIMIT sqlc.arg('limit');

-- name: GetCustomThemeByID :one
SELECT id, user_id, name, bg_color, text_color, accent_color, custom_css, is_system, created_at, updated_at
FROM custom_themes
WHERE id = ? LIMIT 1;

-- name: GetCustomThemesByIDs :many
SELECT id, user_id, name, bg_color, text_color, accent_color, custom_css, is_system, created_at, updated_at
FROM custom_themes
WHERE id IN (sqlc.slice('ids'));

-- name: CreateCustomTheme :one
INSERT INTO custom_themes (
    id, user_id, name, bg_color, text_color, accent_color, custom_css, is_system, created_at, updated_at
) VALUES (
    ?, ?, ?, ?, ?, ?, ?, ?, datetime('now'), datetime('now')
)
RETURNING id, user_id, name, bg_color, text_color, accent_color, custom_css, is_system, created_at, updated_at;

-- name: UpdateCustomTheme :one
UPDATE custom_themes
SET name = ?, bg_color = ?, text_color = ?, accent_color = ?, custom_css = ?, updated_at = datetime('now')
WHERE id = ?
RETURNING id, user_id, name, bg_color, text_color, accent_color, custom_css, is_system, created_at, updated_at;

-- name: DeleteCustomTheme :exec
DELETE FROM custom_themes WHERE id = ?;
