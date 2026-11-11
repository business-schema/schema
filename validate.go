package schema

import (
	"fmt"
	"net/url"
	"regexp"
)

var (
	keyPattern     = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)
	versionPattern = regexp.MustCompile(`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$`)
)

// Values is the set of value types an entity's properties may use.
type Values map[string]Value

func NewValues(values ...Value) Values {
	set := make(Values, len(values))
	for _, v := range values {
		set[v.Key] = v
	}
	return set
}

// Validate checks every value type in the set against the set itself.
func (vs Values) Validate() error {
	for key, v := range vs {
		fail := func(format string, args ...any) error {
			return fmt.Errorf("value %q: %s", v.Key, fmt.Sprintf(format, args...))
		}
		if key != v.Key || !keyPattern.MatchString(v.Key) {
			return fail("invalid key")
		}
		if err := validateHeader(v.Name, v.Version, v.SameAs); err != nil {
			return fail("%s", err)
		}
		if err := validateProperties(v.Properties, vs); err != nil {
			return fail("%s", err)
		}
		for _, p := range v.Properties {
			if p.MatchKey {
				return fail("property %q: a match key belongs on an entity", p.Key)
			}
		}
	}
	return nil
}

// Validate checks the definition is complete and refers only to things it
// defines. values resolves the value types its properties use.
func (e Entity) Validate(values Values) error {
	if !keyPattern.MatchString(e.Key) {
		return fmt.Errorf("entity %q: invalid key", e.Key)
	}
	fail := func(format string, args ...any) error {
		return fmt.Errorf("entity %q: %s", e.Key, fmt.Sprintf(format, args...))
	}
	if err := validateHeader(e.Name, e.Version, e.SameAs); err != nil {
		return fail("%s", err)
	}
	if e.PluralName == "" {
		return fail("plural name is required")
	}
	if err := validateProperties(e.Properties, values); err != nil {
		return fail("%s", err)
	}
	if err := uniqueKeys("relation", len(e.Relations), func(i int) string { return e.Relations[i].Key }); err != nil {
		return fail("%s", err)
	}
	for _, r := range e.Relations {
		if !keyPattern.MatchString(r.Entity) {
			return fail("relation %q: invalid entity", r.Key)
		}
	}
	if err := uniqueKeys("event", len(e.Events), func(i int) string { return e.Events[i].Key }); err != nil {
		return fail("%s", err)
	}
	if err := uniqueKeys("action", len(e.Actions), func(i int) string { return e.Actions[i].Key }); err != nil {
		return fail("%s", err)
	}
	for _, a := range e.Actions {
		for _, emitted := range a.Emits {
			if _, ok := e.Event(emitted); !ok {
				return fail("action %q emits undefined event %q", a.Key, emitted)
			}
		}
	}
	return nil
}

func validateHeader(name, version, sameAs string) error {
	if name == "" {
		return fmt.Errorf("name is required")
	}
	if !versionPattern.MatchString(version) {
		return fmt.Errorf("version %q is not a semantic version", version)
	}
	return validateSameAs(sameAs)
}

func validateSameAs(sameAs string) error {
	if sameAs == "" {
		return nil
	}
	if u, err := url.Parse(sameAs); err != nil || u.Scheme != "https" || u.Host == "" {
		return fmt.Errorf("sameAs %q must be an https URL", sameAs)
	}
	return nil
}

func validateProperties(properties []Property, values Values) error {
	if err := uniqueKeys("property", len(properties), func(i int) string { return properties[i].Key }); err != nil {
		return err
	}
	for _, p := range properties {
		if p.Name == "" {
			return fmt.Errorf("property %q: name is required", p.Key)
		}
		switch p.Type {
		case PropertyTypeString, PropertyTypeNumber, PropertyTypeBoolean, PropertyTypeDate, PropertyTypeDateTime:
		case PropertyTypeEnum:
			if len(p.Options) == 0 {
				return fmt.Errorf("property %q: an enum needs options", p.Key)
			}
		case PropertyTypeValue:
			if _, ok := values[p.Value]; !ok {
				return fmt.Errorf("property %q: undefined value type %q", p.Key, p.Value)
			}
		default:
			return fmt.Errorf("property %q: unknown type %q", p.Key, p.Type)
		}
		if p.MatchKey && (p.Many || p.List || p.Type == PropertyTypeValue) {
			return fmt.Errorf("property %q: a match key must be a single scalar", p.Key)
		}
		if err := validateSameAs(p.SameAs); err != nil {
			return fmt.Errorf("property %q: %w", p.Key, err)
		}
	}
	return nil
}

func uniqueKeys(kind string, n int, key func(int) string) error {
	seen := map[string]bool{}
	for i := range n {
		k := key(i)
		if !keyPattern.MatchString(k) {
			return fmt.Errorf("%s %q: invalid key", kind, k)
		}
		if seen[k] {
			return fmt.Errorf("%s %q: defined twice", kind, k)
		}
		seen[k] = true
	}
	return nil
}
