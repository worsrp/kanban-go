-- name: CreateBoard :one
INSERT INTO boards (title)
VALUES ($1)
RETURNING *;

-- name: GetBoard :one
SELECT * FROM boards
WHERE id = $1 LIMIT 1;

-- name: ListColumnsByBoard :many
SELECT * FROM columns
WHERE board_id = $1
ORDER BY position ASC;

-- name: CreateColumn :one
INSERT INTO columns (board_id, title, position)
VALUES ($1, $2, $3)
RETURNING *;

-- name: ListCardsByBoard :many
SELECT c.* FROM cards c
JOIN columns col ON c.column_id = col.id
WHERE col.board_id = $1
ORDER BY c.position ASC;

-- name: CreateCard :one
INSERT INTO cards (column_id, title, description, position)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: MoveCard :one
UPDATE cards
SET column_id = $2, position = $3, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: UpdateCard :one
UPDATE cards
SET title = $2, description = $3, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteCard :exec
DELETE FROM cards
WHERE id = $1;