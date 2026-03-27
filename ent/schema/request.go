package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// Request holds the schema definition for the Request entity.
type Request struct {
	ent.Schema
}

// Fields of the Request.
func (Request) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(uuid.New).
			Immutable(),
		field.UUID("gate_id", uuid.UUID{}),
		field.String("agent_id").
			Optional().
			Nillable(),
		field.String("method").
			NotEmpty(),
		field.String("path").
			NotEmpty(),
		field.JSON("headers", map[string][]string{}).
			Optional(),
		field.Bytes("body").
			Optional(),
		field.Time("created_at"),
	}
}

// Edges of the Request.
func (Request) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("gate", Gate.Type).
			Ref("requests").
			Field("gate_id").
			Unique().
			Required(),
		edge.To("responses", Response.Type).
			Annotations(entsql.OnDelete(entsql.Cascade)),
		edge.To("diff", Diff.Type).
			Unique().
			Annotations(entsql.OnDelete(entsql.Cascade)),
	}
}

// Indexes of the Request.
func (Request) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("gate_id"),
		index.Fields("gate_id", "created_at"),
		index.Fields("gate_id", "method"),
		index.Fields("gate_id", "path"),
		index.Fields("agent_id"),
	}
}
