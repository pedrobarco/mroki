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

-- name: SaveRequest :exec
INSERT INTO requests (id, gate_id, method, path, headers, body, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7);

-- name: GetRequestByID :one
SELECT id, gate_id, method, path, headers, body, created_at
FROM requests
WHERE id = $1 AND gate_id = $2;

-- name: GetAllRequestsByGateID :many
SELECT id, gate_id, method, path, headers, body, created_at
FROM requests
WHERE gate_id = $1
ORDER BY created_at DESC;

