-- Gates table
CREATE TABLE gates (
    id UUID PRIMARY KEY,
    live_url TEXT,
    shadow_url TEXT
);

-- Requests table
CREATE TABLE requests (
    id UUID PRIMARY KEY,
    gate_id UUID REFERENCES gates(id) ON DELETE CASCADE,
    method TEXT,
    path TEXT,
    headers JSONB,
    body BYTEA,
    created_at TIMESTAMPTZ
);

-- Responses table
CREATE TABLE responses (
    id UUID PRIMARY KEY,
    request_id UUID REFERENCES requests(id) ON DELETE CASCADE,
    type TEXT,
    status_code INTEGER,
    headers JSONB,
    body BYTEA,
    created_at TIMESTAMPTZ
);

-- Diffs table
CREATE TABLE diffs (
    id UUID PRIMARY KEY,
    request_id UUID REFERENCES requests(id) ON DELETE CASCADE,
    from_response_id UUID REFERENCES responses(id) ON DELETE CASCADE,
    to_response_id UUID REFERENCES responses(id) ON DELETE CASCADE,
    content BYTEA
);

-- -- Index for efficient lookup
-- CREATE INDEX idx_responses_request_id ON responses(request_id);
