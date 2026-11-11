package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/business-schema/schema"
)

func TestBuildWritesAPageAndDocumentForEveryType(t *testing.T) {
	catalogue, err := schema.Load()
	if err != nil {
		t.Fatal(err)
	}
	out := t.TempDir()
	if err := build(catalogue, os.DirFS("../../examples"), out); err != nil {
		t.Fatal(err)
	}
	read := func(name string) string {
		raw, err := os.ReadFile(filepath.Join(out, name))
		if err != nil {
			t.Fatal(err)
		}
		return string(raw)
	}
	for _, name := range []string{"index.html", "style.css", "catalogue.json"} {
		if read(name) == "" {
			t.Errorf("%s is empty", name)
		}
	}
	for _, key := range []string{"entity", "external_reference", "customer", "person", "address"} {
		page := read(key + ".html")
		if !strings.Contains(page, "<h2>Properties</h2>") || !strings.Contains(page, "<h2>Example</h2>") || !strings.Contains(page, `href="`+key+`.json"`) {
			t.Errorf("%s page is missing a section", key)
		}
		if !strings.Contains(read(key+".json"), `"key": "`+key+`"`) {
			t.Errorf("%s.json does not hold the definition", key)
		}
		if !strings.Contains(read("index.html"), `href="`+key+`"`) {
			t.Errorf("index does not link to %s", key)
		}
	}
	// Links are relative, so the site also works under a path such as
	// business-schema.github.io/schema/.
	for _, page := range []string{"index.html", "customer.html"} {
		if strings.Contains(read(page), `href="/`) || strings.Contains(read(page), `src="/`) {
			t.Errorf("%s has a root-relative link", page)
		}
	}
	for _, want := range []string{`"key": "id"`, `"key": "external_references"`, `"list": true`} {
		if !strings.Contains(read("customer.json"), want) {
			t.Errorf("published customer definition is missing %s", want)
		}
	}
	customer := read("customer.html")
	for _, want := range []string{`<a href="address">Address</a>`, "Match key", "<h2>Statuses</h2>", "<h2>Events</h2>", "https://schema.org/email"} {
		if !strings.Contains(customer, want) {
			t.Errorf("customer page is missing %q", want)
		}
	}
}

func TestExamplesMayOnlyUseDefinedProperties(t *testing.T) {
	catalogue, err := schema.Load()
	if err != nil {
		t.Fatal(err)
	}
	for name, example := range map[string]string{
		"reference is not a list":    `{"external_references":{"stripe/customer":{"id":"cust_1"}}}`,
		"reference is not an object": `{"external_references":{"stripe/customer":["cust_1"]}}`,
		"unknown reference property": `{"external_references":{"stripe/customer":[{"vendor":"stripe","id":"cust_1"}]}}`,
		"unknown property":           `{"display_name":"Alex","nickname":"A"}`,
		"unknown nested property":    `{"person":{"nick":"H"}}`,
		"unknown in a labelled":      `{"addresses":{"billing":{"planet":"Earth"}}}`,
		"value given as text":        `{"person":"Alex Morgan"}`,
	} {
		t.Run(name, func(t *testing.T) {
			examples := fstest.MapFS{"customer.json": {Data: []byte(example)}}
			if err := build(catalogue, examples, t.TempDir()); err == nil || !strings.Contains(err.Error(), "example customer") {
				t.Fatalf("got %v, want an example error", err)
			}
		})
	}
}
