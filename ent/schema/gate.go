package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

// Gate holds the schema definition for the Gate entity.
type Gate struct {
	ent.Schema
}

// Fields of the Gate.
func (Gate) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(uuid.New).
			Immutable(),
		field.String("live_url").
			NotEmpty(),
		field.String("shadow_url").
			NotEmpty(),
	}
}

// Edges of the Gate.
func (Gate) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("requests", Request.Type).
			Annotations(entsql.OnDelete(entsql.Cascade)),
	}
}
