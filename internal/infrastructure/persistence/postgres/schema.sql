-- Gates table
CREATE TABLE gates (
    id UUID PRIMARY KEY,
    live_url TEXT NOT NULL,
    shadow_url TEXT NOT NULL
);

-- Requests table
CREATE TABLE requests (
    id UUID PRIMARY KEY,
    gate_id UUID REFERENCES gates(id) ON DELETE CASCADE,
    agent_id TEXT,
    method TEXT NOT NULL,
    path TEXT NOT NULL,
    headers JSONB,
    body BYTEA,
    created_at TIMESTAMPTZ NOT NULL
);

-- Responses table
CREATE TABLE responses (
    id UUID PRIMARY KEY,
    request_id UUID REFERENCES requests(id) ON DELETE CASCADE,
    type TEXT NOT NULL,
    status_code INTEGER NOT NULL,
    headers JSONB,
    body BYTEA,
    created_at TIMESTAMPTZ NOT NULL
);

-- Diffs table
CREATE TABLE diffs (
    request_id UUID PRIMARY KEY REFERENCES requests(id) ON DELETE CASCADE,
    from_response_id UUID NOT NULL REFERENCES responses(id) ON DELETE CASCADE,
    to_response_id UUID NOT NULL REFERENCES responses(id) ON DELETE CASCADE,
    content TEXT NOT NULL
);

-- PERFORMANCE INDICES
-- High-priority indices for query optimization
CREATE INDEX idx_requests_gate_id ON requests(gate_id);
CREATE INDEX idx_requests_gate_id_created_at ON requests(gate_id, created_at DESC); -- Composite for list queries
CREATE INDEX idx_responses_request_id ON responses(request_id);
CREATE INDEX idx_requests_agent_id ON requests(agent_id);
