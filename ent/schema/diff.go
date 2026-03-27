package schema

import (
	"time"

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
		field.Time("created_at").
			Default(time.Now).
			Immutable(),
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
		edge.From("from_response", Response.Type).
			Ref("diffs_from").
			Field("from_response_id").
			Unique().
			Required(),
		edge.From("to_response", Response.Type).
			Ref("diffs_to").
			Field("to_response_id").
			Unique().
			Required(),
	}
}
