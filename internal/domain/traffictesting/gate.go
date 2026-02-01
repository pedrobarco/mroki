package traffictesting

type Gate struct {
	ID        GateID
	LiveURL   GateURL
	ShadowURL GateURL

	Requests []Request
}

type gateOption func(*Gate)

func WithGateID(id GateID) gateOption {
	return func(g *Gate) {
		g.ID = id
	}
}

func NewGate(live, shadow GateURL, opts ...gateOption) (*Gate, error) {
	gate := &Gate{
		LiveURL:   live,
		ShadowURL: shadow,
	}

	for _, o := range opts {
		o(gate)
	}

	if gate.ID.IsZero() {
		gate.ID = NewGateID()
	}

	return gate, nil
}
