package diffing

import (
	"net/url"

	"github.com/google/uuid"
)

type Gate struct {
	ID        uuid.UUID
	LiveURL   *url.URL
	ShadowURL *url.URL

	Requests []Request
}

type gateOption func(*Gate)

func WithGateID(id uuid.UUID) gateOption {
	return func(g *Gate) {
		g.ID = id
	}
}

func NewGate(live, shadow *url.URL, opts ...gateOption) (*Gate, error) {
	gate := &Gate{
		LiveURL:   live,
		ShadowURL: shadow,
	}

	for _, o := range opts {
		o(gate)
	}

	if gate.ID == uuid.Nil {
		gate.ID = uuid.New()
	}

	return gate, nil
}
