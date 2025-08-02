-- name: SaveGate :exec
INSERT INTO gates (id, live_url, shadow_url)
VALUES ($1, $2, $3);

-- name: GetGateByID :one
SELECT id, live_url, shadow_url
FROM gates
WHERE id = $1;

-- name: GetAllGates :many
SELECT id, live_url, shadow_url
FROM gates;
