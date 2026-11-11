// Package schema is the Business Schema: a public, language-neutral
// description of the entities a business runs on. The JSON files in this
// repository are the definitions; this package describes their format,
// validates them and loads the set released with this version of the module.
package schema

// BaseEntity defines the properties inherited by every entity type.
// It has no lifecycle, actions or events of its own.
type BaseEntity struct {
	Key         string     `json:"key"`
	Name        string     `json:"name"`
	Description string     `json:"description,omitempty"`
	Version     string     `json:"version"`
	Properties  []Property `json:"properties"`
}

// Entity describes a business entity type: a thing with its own identity that
// systems read, change through actions and hear about through events.
type Entity struct {
	Base        string `json:"base"` // Key of the shared base entity definition.
	Key         string `json:"key"`  // e.g. customer
	Name        string `json:"name"`
	PluralName  string `json:"pluralName"`
	Description string `json:"description,omitempty"` // Plain-language description
	Version     string `json:"version"`               // Semantic version; changes within a major version only add
	SameAs      string `json:"sameAs,omitempty"`      // URL of the equivalent term in another vocabulary

	Properties []Property `json:"properties,omitempty"`
	Statuses   []Status   `json:"statuses,omitempty"` // The first status is the default for a new record
	Relations  []Relation `json:"relations,omitempty"`
	Actions    []Action   `json:"actions,omitempty"`
	Events     []Event    `json:"events,omitempty"`
}

// Value describes a reusable shape with no identity of its own, such as an
// address. It is held inside an entity and changes as part of that entity.
type Value struct {
	Key         string `json:"key"` // e.g. address
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Version     string `json:"version"`
	SameAs      string `json:"sameAs,omitempty"`

	Properties []Property `json:"properties,omitempty"`
}

type PropertyType string

const (
	PropertyTypeString   PropertyType = "string"
	PropertyTypeNumber   PropertyType = "number"
	PropertyTypeBoolean  PropertyType = "boolean"
	PropertyTypeDate     PropertyType = "date"     // YYYY-MM-DD
	PropertyTypeDateTime PropertyType = "datetime" // RFC 3339
	PropertyTypeEnum     PropertyType = "enum"     // One of Options
	PropertyTypeValue    PropertyType = "value"    // A Value, named by Property.Value
)

// Property formats narrow a string property.
const (
	FormatEmail    = "email"
	FormatPhone    = "phone"    // E.164
	FormatCountry  = "country"  // ISO 3166-1 alpha-2
	FormatCurrency = "currency" // ISO 4217
	FormatLanguage = "language" // BCP 47
	FormatURL      = "url"      // Absolute http or https URL
)

// Comparison decides when two systems' values for a property count as equal.
type Comparison string

const (
	CompareExact           Comparison = ""
	CompareCaseInsensitive Comparison = "case-insensitive"
)

type Property struct {
	Key         string `json:"key"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`

	Type     PropertyType      `json:"type"`
	Value    string            `json:"value,omitempty"` // Value key, when Type is PropertyTypeValue
	Required bool              `json:"required,omitempty"`
	Format   string            `json:"format,omitempty"`
	PII      bool              `json:"pii,omitempty"`     // Personal data, redactable
	Secure   bool              `json:"secure,omitempty"`  // Masked until revealed
	Options  map[string]string `json:"options,omitempty"` // Enum values and their names

	// Many holds several values keyed by a caller-chosen label, such as the
	// billing and shipping addresses of a customer.
	Many bool `json:"many,omitempty"`

	// List holds an array of values. With Many, each labelled entry is an array.
	List bool `json:"list,omitempty"`

	Compare  Comparison `json:"compare,omitempty"`
	MatchKey bool       `json:"matchKey,omitempty"` // Identifies the same record across systems
	SameAs   string     `json:"sameAs,omitempty"`   // URL of the equivalent term in another vocabulary
}

type Tone string

const (
	ToneNeutral Tone = "neutral"
	ToneInfo    Tone = "info"
	ToneSuccess Tone = "success"
	ToneWarning Tone = "warning"
	ToneDanger  Tone = "danger"
)

type Status struct {
	Key  string `json:"key"`
	Name string `json:"name"`
	Tone Tone   `json:"tone,omitempty"`
}

// Relation links this entity to another entity type.
type Relation struct {
	Key         string `json:"key"`                   // e.g. subscriptions
	Name        string `json:"name"`                  // Read from this entity, e.g. "has subscriptions"
	InverseName string `json:"inverseName,omitempty"` // Read from the other entity, e.g. "belongs to customer"
	Entity      string `json:"entity"`                // Key of the related entity type
	Many        bool   `json:"many,omitempty"`
	Owned       bool   `json:"owned,omitempty"` // The related record has exactly one owner of this type
}

type Action struct {
	Key         string   `json:"key"`
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Emits       []string `json:"emits,omitempty"` // Event keys
}

type Event struct {
	Key         string `json:"key"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

func (e Entity) Property(key string) (Property, bool) { return findProperty(e.Properties, key) }

func (v Value) Property(key string) (Property, bool) { return findProperty(v.Properties, key) }

func (e Entity) Status(key string) (Status, bool) {
	for _, s := range e.Statuses {
		if s.Key == key {
			return s, true
		}
	}
	return Status{}, false
}

func (e Entity) Event(key string) (Event, bool) {
	for _, ev := range e.Events {
		if ev.Key == key {
			return ev, true
		}
	}
	return Event{}, false
}

func findProperty(properties []Property, key string) (Property, bool) {
	for _, p := range properties {
		if p.Key == key {
			return p, true
		}
	}
	return Property{}, false
}
