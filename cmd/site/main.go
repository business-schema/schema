// Command site builds the business-schema.org website from the definitions:
// a landing page, a page and a JSON document for each type, and one catalogue
// document for the whole release.
package main

import (
	"bytes"
	"embed"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/business-schema/schema"
)

//go:embed assets
var assets embed.FS

func main() {
	out := flag.String("out", "_site", "directory to write the site into")
	examples := flag.String("examples", "examples", "directory of example records, one <key>.json per type")
	flag.Parse()
	catalogue, err := schema.Load()
	if err != nil {
		log.Fatal(err)
	}
	if err := build(catalogue, os.DirFS(*examples), *out); err != nil {
		log.Fatal(err)
	}
}

// typePage is one entity or value type as the site presents it.
type typePage struct {
	Key, Name, Description, Version, SameAs string
	Kind                                    string // Entity, Base entity or Value type
	Base                                    string
	Properties                              []propertyRow
	Statuses                                []schema.Status
	Actions                                 []schema.Action
	Events                                  []schema.Event
	Example                                 string
	Nav                                     []navGroup
}

// navGroup is one heading in the site navigation and the types under it.
type navGroup struct {
	Title string
	Items []navItem
}

type navItem struct {
	Key, Name string
	Current   bool
}

// navigation lists every type under its kind, marking the page being shown.
func navigation(pages []typePage, current string) []navGroup {
	groups := []navGroup{{Title: "Entities"}, {Title: "Value types"}, {Title: "Base types"}}
	for _, page := range pages {
		group := &groups[1]
		if page.Kind == "Entity" {
			group = &groups[0]
		} else if page.Kind == "Base entity" {
			group = &groups[2]
		}
		group.Items = append(group.Items, navItem{Key: page.Key, Name: page.Name, Current: page.Key == current})
	}
	return groups
}

type propertyRow struct {
	schema.Property
	TypeLabel string
	TypeLink  string // Key of the value type this property holds
	Notes     []string
}

func build(catalogue schema.Catalogue, examples fs.FS, out string) error {
	pages, err := typePages(catalogue, examples)
	if err != nil {
		return err
	}
	templates, err := template.ParseFS(assets, "assets/*.html")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(out, 0o755); err != nil {
		return err
	}
	if err := render(templates, "index.html", filepath.Join(out, "index.html"), map[string]any{"Types": pages, "Nav": navigation(pages, "")}); err != nil {
		return err
	}
	for _, page := range pages {
		if err := render(templates, "type.html", filepath.Join(out, page.Key+".html"), page); err != nil {
			return err
		}
	}
	if err := writeJSON(filepath.Join(out, "entity.json"), catalogue.BaseEntity); err != nil {
		return err
	}
	for _, e := range catalogue.Entities {
		if err := writeJSON(filepath.Join(out, e.Key+".json"), e); err != nil {
			return err
		}
	}
	for _, v := range catalogue.Values {
		if err := writeJSON(filepath.Join(out, v.Key+".json"), v); err != nil {
			return err
		}
	}
	if err := writeJSON(filepath.Join(out, "catalogue.json"), catalogue); err != nil {
		return err
	}
	style, err := assets.ReadFile("assets/style.css")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(out, "style.css"), style, 0o644)
}

func typePages(catalogue schema.Catalogue, examples fs.FS) ([]typePage, error) {
	values := catalogue.ValueSet()
	base := catalogue.BaseEntity
	pages := []typePage{{Key: base.Key, Name: base.Name, Description: base.Description, Version: base.Version, Kind: "Base entity", Properties: propertyRows(base.Properties, values)}}
	for _, definition := range catalogue.Entities {
		e, _ := catalogue.Entity(definition.Key)
		pages = append(pages, typePage{
			Key: e.Key, Name: e.Name, Description: e.Description, Version: e.Version, SameAs: e.SameAs, Kind: "Entity", Base: e.Base,
			Properties: propertyRows(e.Properties, values), Statuses: e.Statuses, Actions: e.Actions, Events: e.Events,
		})
	}
	for _, v := range catalogue.Values {
		pages = append(pages, typePage{
			Key: v.Key, Name: v.Name, Description: v.Description, Version: v.Version, SameAs: v.SameAs, Kind: "Value type",
			Properties: propertyRows(v.Properties, values),
		})
	}
	sort.SliceStable(pages, func(i, j int) bool { return pages[i].Kind < pages[j].Kind })
	for i := range pages {
		example, err := loadExample(examples, pages[i].Key, pages[i].Properties, values)
		if err != nil {
			return nil, err
		}
		pages[i].Example = example
	}
	for i := range pages {
		pages[i].Nav = navigation(pages, pages[i].Key)
	}
	return pages, nil
}

func propertyRows(properties []schema.Property, values schema.Values) []propertyRow {
	rows := make([]propertyRow, 0, len(properties))
	for _, p := range properties {
		row := propertyRow{Property: p, TypeLabel: typeLabel(p, values)}
		if p.Type == schema.PropertyTypeValue {
			row.TypeLink = p.Value
		}
		if p.Required {
			row.Notes = append(row.Notes, "Required")
		}
		if p.Many {
			row.Notes = append(row.Notes, "Several, keyed by a label")
		}
		if p.List {
			row.Notes = append(row.Notes, "List of values")
		}
		if p.MatchKey {
			row.Notes = append(row.Notes, "Match key")
		}
		if p.Compare == schema.CompareCaseInsensitive {
			row.Notes = append(row.Notes, "Compared ignoring case")
		}
		if p.Secure {
			row.Notes = append(row.Notes, "Personal data, masked")
		} else if p.PII {
			row.Notes = append(row.Notes, "Personal data")
		}
		rows = append(rows, row)
	}
	return rows
}

var formatLabels = map[string]string{
	schema.FormatEmail:    "Email address",
	schema.FormatPhone:    "Phone number (E.164)",
	schema.FormatCountry:  "Country code (ISO 3166-1 alpha-2)",
	schema.FormatCurrency: "Currency code (ISO 4217)",
	schema.FormatLanguage: "Language tag (BCP 47)",
	schema.FormatURL:      "Web address",
}

func typeLabel(p schema.Property, values schema.Values) string {
	switch p.Type {
	case schema.PropertyTypeValue:
		return values[p.Value].Name
	case schema.PropertyTypeEnum:
		options := make([]string, 0, len(p.Options))
		for value := range p.Options {
			options = append(options, value)
		}
		sort.Strings(options)
		return "One of: " + strings.Join(options, ", ")
	case schema.PropertyTypeDate:
		return "Date (YYYY-MM-DD)"
	case schema.PropertyTypeDateTime:
		return "Date and time (RFC 3339)"
	case schema.PropertyTypeNumber:
		return "Number"
	case schema.PropertyTypeBoolean:
		return "True or false"
	}
	if label, ok := formatLabels[p.Format]; ok {
		return label
	}
	return "Text"
}

// loadExample returns the example record for a type, pretty-printed. An
// example may only use properties the type defines, so it cannot drift from
// the definition. A type with no example file has none.
func loadExample(examples fs.FS, key string, properties []propertyRow, values schema.Values) (string, error) {
	raw, err := fs.ReadFile(examples, key+".json")
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	var record map[string]any
	if err := json.Unmarshal(raw, &record); err != nil {
		return "", fmt.Errorf("example %s: %w", key, err)
	}
	defined := make([]schema.Property, len(properties))
	for i, row := range properties {
		defined[i] = row.Property
	}
	if err := checkExample(record, defined, values, key); err != nil {
		return "", err
	}
	// Indent the file as written, so properties keep the author's order.
	var pretty bytes.Buffer
	if err := json.Indent(&pretty, bytes.TrimSpace(raw), "", "  "); err != nil {
		return "", fmt.Errorf("example %s: %w", key, err)
	}
	return pretty.String(), nil
}

func checkExample(record map[string]any, properties []schema.Property, values schema.Values, path string) error {
	byKey := map[string]schema.Property{}
	for _, p := range properties {
		byKey[p.Key] = p
	}
	for key, value := range record {
		p, ok := byKey[key]
		if !ok {
			return fmt.Errorf("example %s: %q is not a defined property", path, key)
		}
		if p.Type != schema.PropertyTypeValue {
			continue
		}
		entries := map[string]any{"": value}
		if p.Many {
			var ok bool
			entries, ok = value.(map[string]any)
			if !ok {
				return fmt.Errorf("example %s.%s: must be an object", path, key)
			}
		}
		for label, entry := range entries {
			entryPath := strings.TrimSuffix(path+"."+key+"."+label, ".")
			items := []any{entry}
			if p.List {
				var ok bool
				items, ok = entry.([]any)
				if !ok {
					return fmt.Errorf("example %s: must be a list", entryPath)
				}
			}
			for _, item := range items {
				object, ok := item.(map[string]any)
				if !ok {
					return fmt.Errorf("example %s: must be an object", entryPath)
				}
				if err := checkExample(object, values[p.Value].Properties, values, entryPath); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func render(templates *template.Template, name, path string, data any) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	if err := templates.ExecuteTemplate(file, name, data); err != nil {
		file.Close()
		return err
	}
	return file.Close()
}

func writeJSON(path string, value any) error {
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(raw, '\n'), 0o644)
}
