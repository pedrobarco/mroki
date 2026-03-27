package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

// Diff holds the schema definition for the Diff entity.
type Diff struct {
	ent.Schema
}

// Fields of the Diff.
func (Diff) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(uuid.New).
			Immutable(),
		field.UUID("request_id", uuid.UUID{}).
			Unique(),
		field.UUID("from_response_id", uuid.UUID{}),
		field.UUID("to_response_id", uuid.UUID{}),
		field.String("content"),
	}
}

// Edges of the Diff.
func (Diff) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("request", Request.Type).
			Ref("diff").
			Field("request_id").
			Unique().
			Required(),
	}
}
