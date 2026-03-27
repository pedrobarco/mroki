package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// Response holds the schema definition for the Response entity.
type Response struct {
	ent.Schema
}

// Fields of the Response.
func (Response) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(uuid.New).
			Immutable(),
		field.UUID("request_id", uuid.UUID{}),
		field.String("type").
			NotEmpty(),
		field.Int32("status_code"),
		field.JSON("headers", map[string][]string{}).
			Optional(),
		field.Bytes("body").
			Optional(),
		field.Time("created_at"),
	}
}

// Edges of the Response.
func (Response) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("request", Request.Type).
			Ref("responses").
			Field("request_id").
			Unique().
			Required(),
		edge.To("diffs_from", Diff.Type),
		edge.To("diffs_to", Diff.Type),
	}
}

// Indexes of the Response.
func (Response) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("request_id"),
	}
}
