package schema

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strings"
)

//go:embed base/*.json entities/*.json types/*.json
var definitions embed.FS

// Catalogue is every definition in one release of the schema.
type Catalogue struct {
	BaseEntity Value    `json:"baseEntity"` // Shared properties, included in every loaded entity.
	Entities   []Entity `json:"entities"`
	Values     []Value  `json:"values"`
}

// ValueSet returns the catalogue's value types by key.
func (c Catalogue) ValueSet() Values { return NewValues(c.Values...) }

func (c Catalogue) Entity(key string) (Entity, bool) {
	for _, e := range c.Entities {
		if e.Key == key {
			return e, true
		}
	}
	return Entity{}, false
}

// Validate checks every definition in the catalogue against the others.
func (c Catalogue) Validate() error {
	values := c.ValueSet()
	if len(values) != len(c.Values) {
		return fmt.Errorf("a value type is defined twice")
	}
	if err := values.Validate(); err != nil {
		return err
	}
	if err := validateHeader(c.BaseEntity.Name, c.BaseEntity.Version, c.BaseEntity.SameAs); err != nil {
		return fmt.Errorf("base entity: %w", err)
	}
	if err := validateProperties(c.BaseEntity.Properties, values); err != nil {
		return fmt.Errorf("base entity: %w", err)
	}
	entities := map[string]bool{}
	for _, e := range c.Entities {
		if entities[e.Key] {
			return fmt.Errorf("entity %q: defined twice", e.Key)
		}
		entities[e.Key] = true
		if err := e.Validate(values); err != nil {
			return err
		}
	}
	for _, e := range c.Entities {
		for _, r := range e.Relations {
			if !entities[r.Entity] {
				return fmt.Errorf("entity %q: relation %q: undefined entity %q", e.Key, r.Key, r.Entity)
			}
		}
	}
	return nil
}

// Load returns the definitions released with this version of the module, in
// key order.
func Load() (Catalogue, error) { return loadFS(definitions) }

func loadFS(files fs.FS) (Catalogue, error) {
	var c Catalogue
	var bases []Value
	if err := readDir(files, "base", func(v Value) string { return v.Key }, &bases); err != nil {
		return Catalogue{}, err
	}
	if len(bases) != 1 || bases[0].Key != "entity" {
		return Catalogue{}, fmt.Errorf("base must contain only the entity definition")
	}
	c.BaseEntity = bases[0]
	if err := readDir(files, "types", func(v Value) string { return v.Key }, &c.Values); err != nil {
		return Catalogue{}, err
	}
	if err := readDir(files, "entities", func(e Entity) string { return e.Key }, &c.Entities); err != nil {
		return Catalogue{}, err
	}
	for i := range c.Entities {
		c.Entities[i].Properties = append(append([]Property(nil), c.BaseEntity.Properties...), c.Entities[i].Properties...)
	}
	if err := c.Validate(); err != nil {
		return Catalogue{}, err
	}
	return c, nil
}

// readDir decodes every JSON file in dir. A file is named after the key of the
// definition it holds, and holds nothing the format does not define.
func readDir[T any](files fs.FS, dir string, key func(T) string, out *[]T) error {
	entries, err := fs.ReadDir(files, dir)
	if err != nil {
		return err
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
	for _, entry := range entries {
		name := path.Join(dir, entry.Name())
		raw, err := fs.ReadFile(files, name)
		if err != nil {
			return err
		}
		var item T
		decoder := json.NewDecoder(bytes.NewReader(raw))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&item); err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
		if want := strings.TrimSuffix(entry.Name(), ".json"); key(item) != want {
			return fmt.Errorf("%s: holds %q, expected %q", name, key(item), want)
		}
		*out = append(*out, item)
	}
	return nil
}
