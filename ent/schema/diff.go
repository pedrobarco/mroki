package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
	"github.com/pedrobarco/mroki/pkg/diff"
)

// DiffConfigSnapshot is the storage representation of a gate's diff
// configuration at the time a diff was computed. Stored as a JSON column
// so the frontend can interpret patch indices without consulting the
// (possibly changed) current gate config.
type DiffConfigSnapshot struct {
	SortArrays     bool     `json:"sort_arrays"`
	IgnoredFields  []string `json:"ignored_fields,omitempty"`
	IncludedFields []string `json:"included_fields,omitempty"`
	FloatTolerance float64  `json:"float_tolerance,omitempty"`
}

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
		field.JSON("content", []diff.PatchOp{}),
		field.JSON("config", DiffConfigSnapshot{}).
			Optional(),
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
