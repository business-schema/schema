package schema

import (
	"strings"
	"testing"
	"testing/fstest"
)

func TestReleasedDefinitionsLoad(t *testing.T) {
	c, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	customer, ok := c.Entity("customer")
	if !ok {
		t.Fatal("customer is not defined")
	}
	id, ok := customer.Property("id")
	if !ok || !id.Required || id.Type != PropertyTypeString {
		t.Fatalf("customer is missing inherited identity: %+v", id)
	}
	refs, ok := customer.Property("external_references")
	if !ok || !refs.Many || !refs.List || refs.Value != "external_reference" {
		t.Fatalf("customer external references: %+v", refs)
	}
	if _, ok := c.ValueSet()["person"].Property("id"); ok {
		t.Fatal("value types must not inherit entity identity")
	}
	email, _ := customer.Property("email")
	if !email.MatchKey || email.Compare != CompareCaseInsensitive || email.SameAs != "https://schema.org/email" {
		t.Errorf("customer email: %+v", email)
	}
	for _, p := range customer.Properties {
		if p.Type == PropertyTypeValue {
			if _, ok := c.ValueSet()[p.Value]; !ok {
				t.Errorf("customer.%s uses undefined value %q", p.Key, p.Value)
			}
		}
	}
	for _, key := range []string{"address", "organization", "person"} {
		if _, ok := c.ValueSet()[key]; !ok {
			t.Errorf("value %q is not defined", key)
		}
	}
}

const minimalEntity = `{"key":"customer","name":"Customer","pluralName":"Customers","version":"0.1.0",
	"properties":[{"key":"email","name":"Email","type":"string","matchKey":true},{"key":"home","name":"Home","type":"value","value":"address"}],
	"statuses":[{"key":"active","name":"Active"}],
	"actions":[{"key":"create","name":"Create","emits":["created"]}],
	"events":[{"key":"created","name":"Created"}]}`

const minimalValue = `{"key":"address","name":"Address","version":"0.1.0","properties":[{"key":"city","name":"City","type":"string"}]}`

func TestLoadRejectsBrokenDefinitions(t *testing.T) {
	load := func(entity, value string) error {
		_, err := loadFS(fstest.MapFS{
			"base/entity.json":       {Data: []byte(`{"key":"entity","name":"Entity","version":"0.1.0"}`)},
			"entities/customer.json": {Data: []byte(entity)},
			"types/address.json":     {Data: []byte(value)},
		})
		return err
	}
	if err := load(minimalEntity, minimalValue); err != nil {
		t.Fatalf("minimal definitions should load: %v", err)
	}
	for name, tc := range map[string]struct{ entity, value, want string }{
		"unknown field":        {strings.Replace(minimalEntity, `"version"`, `"colour":"red","version"`, 1), minimalValue, "unknown field"},
		"wrong file name":      {strings.Replace(minimalEntity, `"key":"customer"`, `"key":"client"`, 1), minimalValue, `expected "customer"`},
		"bad version":          {strings.Replace(minimalEntity, `0.1.0`, `v1`, 1), minimalValue, "semantic version"},
		"undefined value type": {strings.Replace(minimalEntity, `"value":"address"`, `"value":"location"`, 1), minimalValue, "undefined value type"},
		"undefined event":      {strings.Replace(minimalEntity, `"emits":["created"]`, `"emits":["archived"]`, 1), minimalValue, "undefined event"},
		"no statuses":          {strings.Replace(minimalEntity, `{"key":"active","name":"Active"}`, ``, 1), minimalValue, "status is required"},
		"composite match key":  {strings.Replace(minimalEntity, `"type":"value"`, `"type":"value","matchKey":true`, 1), minimalValue, "single scalar"},
		"duplicate property":   {strings.Replace(minimalEntity, `"key":"home"`, `"key":"email"`, 1), minimalValue, "defined twice"},
		"insecure sameAs":      {minimalEntity, strings.Replace(minimalValue, `"version"`, `"sameAs":"http://schema.org/PostalAddress","version"`, 1), "https URL"},
		"match key in a value": {minimalEntity, strings.Replace(minimalValue, `"type":"string"`, `"type":"string","matchKey":true`, 1), "belongs on an entity"},
		"undefined relation":   {strings.Replace(minimalEntity, `"statuses"`, `"relations":[{"key":"orders","name":"has orders","entity":"order"}],"statuses"`, 1), minimalValue, `undefined entity "order"`},
	} {
		t.Run(name, func(t *testing.T) {
			if err := load(tc.entity, tc.value); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("got %v, want an error containing %q", err, tc.want)
			}
		})
	}
}

func TestEntityCannotRedefineBaseProperties(t *testing.T) {
	_, err := loadFS(fstest.MapFS{
		"base/entity.json":       {Data: []byte(`{"key":"entity","name":"Entity","version":"0.1.0","properties":[{"key":"email","name":"Shared email","type":"string"}]}`)},
		"entities/customer.json": {Data: []byte(minimalEntity)},
		"types/address.json":     {Data: []byte(minimalValue)},
	})
	if err == nil || !strings.Contains(err.Error(), `property "email": defined twice`) {
		t.Fatalf("got %v, want duplicate inherited property error", err)
	}
}
