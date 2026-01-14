package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

// Tunnel holds the schema definition for the Tunnel entity.
type Tunnel struct {
	ent.Schema
}

// Fields of the Tunnel.
func (Tunnel) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).Default(uuid.New).StorageKey("id"),
		field.String("name"),
		field.Enum("type").Values("cloudflare", "ngrok"),
		field.String("target"),
		field.Bool("enabled").Default(true),
		field.Time("created_at").Default(time.Now).Immutable(),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
		field.String("ngrok_authtoken").Optional().Nillable(),
		field.String("ngrok_domain").Optional().Nillable(),
	}
}

// Edges of the Tunnel.
func (Tunnel) Edges() []ent.Edge {
	return nil
}
