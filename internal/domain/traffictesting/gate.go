package traffictesting

import "time"

type Gate struct {
	ID         GateID
	Name       GateName
	LiveURL    GateURL
	ShadowURL  GateURL
	DiffConfig DiffConfig
	CreatedAt  time.Time
}

type gateOption func(*Gate)

func WithGateID(id GateID) gateOption {
	return func(g *Gate) {
		g.ID = id
	}
}

func WithGateCreatedAt(t time.Time) gateOption {
	return func(g *Gate) {
		g.CreatedAt = t
	}
}

func WithGateDiffConfig(dc DiffConfig) gateOption {
	return func(g *Gate) {
		g.DiffConfig = dc
	}
}

func NewGate(name GateName, live, shadow GateURL, opts ...gateOption) (*Gate, error) {
	gate := &Gate{
		Name:       name,
		LiveURL:    live,
		ShadowURL:  shadow,
		DiffConfig: DefaultDiffConfig(),
	}

	for _, o := range opts {
		o(gate)
	}

	if gate.ID.IsZero() {
		gate.ID = NewGateID()
	}

	if gate.CreatedAt.IsZero() {
		gate.CreatedAt = time.Now()
	}

	return gate, nil
}
