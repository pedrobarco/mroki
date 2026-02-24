-- name: SaveGate :exec
INSERT INTO gates (id, live_url, shadow_url)
VALUES ($1, $2, $3);

-- name: GetGateByID :one
SELECT id, live_url, shadow_url
FROM gates
WHERE id = $1;

-- name: GetAllGates :many
SELECT id, live_url, shadow_url
FROM gates
ORDER BY id
LIMIT sqlc.arg('limit')
OFFSET sqlc.arg('offset');

-- name: CountGates :one
SELECT COUNT(*) FROM gates;

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

  diff.from_response_id AS diff_from_response_id,
  diff.to_response_id AS diff_to_response_id,
  diff.content AS diff_content

FROM requests req
LEFT JOIN responses resp ON resp.request_id = req.id
LEFT JOIN diffs diff ON diff.request_id = req.id
WHERE req.gate_id = $1 AND req.id = $2;

-- name: GetFilteredRequests :many
SELECT id, gate_id, agent_id, method, path, headers, body, created_at
FROM requests
WHERE
  gate_id = sqlc.arg('gate_id')
  -- Optional method filter: empty array = no filter
  AND (
    cardinality(sqlc.narg('methods')::text[]) = 0
    OR method = ANY(sqlc.narg('methods')::text[])
  )
  -- Optional path pattern filter: NULL = no filter
  AND (
    sqlc.narg('path_pattern')::text IS NULL
    OR sqlc.narg('path_pattern')::text = ''
    OR path LIKE sqlc.narg('path_pattern')::text
  )
  -- Optional date range filters
  AND (sqlc.narg('from_date')::timestamptz IS NULL OR created_at >= sqlc.narg('from_date')::timestamptz)
  AND (sqlc.narg('to_date')::timestamptz IS NULL OR created_at <= sqlc.narg('to_date')::timestamptz)
  -- Optional agent ID filter: NULL or empty = no filter
  AND (
    sqlc.narg('agent_id')::text IS NULL
    OR sqlc.narg('agent_id')::text = ''
    OR agent_id = sqlc.narg('agent_id')::text
  )
  -- Optional diff existence filter: NULL = no filter
  AND (
    sqlc.narg('has_diff')::boolean IS NULL
    OR (
      sqlc.narg('has_diff')::boolean = true
      AND EXISTS (SELECT 1 FROM diffs d WHERE d.request_id = requests.id)
    )
    OR (
      sqlc.narg('has_diff')::boolean = false
      AND NOT EXISTS (SELECT 1 FROM diffs d WHERE d.request_id = requests.id)
    )
  )
ORDER BY 
  -- Dynamic sorting with CASE statements
  CASE 
    WHEN sqlc.arg('sort_order') = 'asc' THEN
      CASE sqlc.arg('sort_field')
        WHEN 'method' THEN method
        WHEN 'path' THEN path
        WHEN 'created_at' THEN created_at::text
        ELSE created_at::text
      END
  END ASC,
  CASE 
    WHEN sqlc.arg('sort_order') = 'desc' THEN
      CASE sqlc.arg('sort_field')
        WHEN 'method' THEN method
        WHEN 'path' THEN path
        WHEN 'created_at' THEN created_at::text
        ELSE created_at::text
      END
  END DESC
LIMIT sqlc.arg('limit')
OFFSET sqlc.arg('offset');

-- name: CountFilteredRequests :one
SELECT COUNT(*)
FROM requests
WHERE
  gate_id = sqlc.arg('gate_id')
  -- Same filters as GetFilteredRequests for consistency
  AND (
    cardinality(sqlc.narg('methods')::text[]) = 0
    OR method = ANY(sqlc.narg('methods')::text[])
  )
  AND (
    sqlc.narg('path_pattern')::text IS NULL
    OR sqlc.narg('path_pattern')::text = ''
    OR path LIKE sqlc.narg('path_pattern')::text
  )
  AND (sqlc.narg('from_date')::timestamptz IS NULL OR created_at >= sqlc.narg('from_date')::timestamptz)
  AND (sqlc.narg('to_date')::timestamptz IS NULL OR created_at <= sqlc.narg('to_date')::timestamptz)
  AND (
    sqlc.narg('agent_id')::text IS NULL
    OR sqlc.narg('agent_id')::text = ''
    OR agent_id = sqlc.narg('agent_id')::text
  )
  AND (
    sqlc.narg('has_diff')::boolean IS NULL
    OR (
      sqlc.narg('has_diff')::boolean = true
      AND EXISTS (SELECT 1 FROM diffs d WHERE d.request_id = requests.id)
    )
    OR (
      sqlc.narg('has_diff')::boolean = false
      AND NOT EXISTS (SELECT 1 FROM diffs d WHERE d.request_id = requests.id)
    )
  );

-- name: SaveResponse :exec
INSERT INTO responses (id, request_id, type, status_code, headers, body, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7);

-- name: SaveDiff :exec
INSERT INTO diffs (request_id, from_response_id, to_response_id, content)
VALUES ($1, $2, $3, $4);
