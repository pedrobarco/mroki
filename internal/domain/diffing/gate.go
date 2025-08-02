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

func WithID(id uuid.UUID) gateOption {
	return func(g *Gate) {
		g.ID = id
	}
}

func NewGate(live, shadow *url.URL) (*Gate, error) {
	return &Gate{
		ID:        uuid.New(),
		LiveURL:   live,
		ShadowURL: shadow,
	}, nil
}
