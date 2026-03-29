package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
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
		field.String("name").
			NotEmpty().
			Unique(),
		field.String("live_url").
			NotEmpty().
			Immutable(),
		field.String("shadow_url").
			NotEmpty().
			Immutable(),
		field.Time("created_at").
			Default(time.Now).
			Immutable(),
	}
}

// Edges of the Gate.
func (Gate) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("requests", Request.Type).
			Annotations(entsql.OnDelete(entsql.Cascade)),
	}
}

// Indexes of the Gate.
func (Gate) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("live_url", "shadow_url").
			Unique(),
	}
}
