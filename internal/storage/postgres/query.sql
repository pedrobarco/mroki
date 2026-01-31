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
INSERT INTO requests (id, gate_id, agent_id, method, path, headers, body, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8);

-- name: GetRequestByID :many
SELECT
  req.id as request_id,
  req.gate_id as request_gate_id,
  req.agent_id as request_agent_id,
  req.method as request_method,
  req.path as request_path,
  req.headers as request_headers,
  req.body as request_body,
  req.created_at as request_created_at,

  resp.id AS response_id,
  resp.type AS response_type,
  resp.status_code AS response_status_code,
  resp.headers AS response_headers,
  resp.body AS response_body,
  resp.created_at AS response_created_at,

  diff.id AS diff_id,
  diff.from_response_id AS diff_from_response_id,
  diff.to_response_id AS diff_to_response_id,
  diff.content AS diff_content

FROM requests req
LEFT JOIN responses resp ON resp.request_id = req.id
LEFT JOIN diffs diff ON diff.request_id = req.id
WHERE req.gate_id = $1 AND req.id = $2;

-- name: GetAllRequestsByGateID :many
SELECT id, gate_id, agent_id, method, path, headers, body, created_at
FROM requests
WHERE gate_id = $1
ORDER BY created_at DESC;

-- name: SaveResponse :exec
INSERT INTO responses (id, request_id, type, status_code, headers, body, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7);

-- name: SaveDiff :exec
INSERT INTO diffs (id, request_id, from_response_id, to_response_id, content)
VALUES ($1, $2, $3, $4, $5);
