# Business Schema

A public, language-neutral description of the entities a business runs on:
what a customer is, which properties it has, the events it raises. Systems that
agree on these definitions can exchange and reconcile records without bespoke
mappings.

**Status: draft (0.x).** Definitions may change incompatibly until 1.0. From
1.0, a change within a major version only adds.

## Layout

| Path | Holds |
| --- | --- |
| `base/entity.json` | Identity properties shared by every entity |
| `entities/<key>.json` | Entity types: things with their own identity, such as `customer` |
| `types/<key>.json` | Value types: reusable shapes held inside entities, such as `address` |
| `examples/<key>.json` | An example record for each type, shown on its page |
| `cmd/site` | Builds the business-schema.org website from the definitions |
| `schema.go` | The definition format, as Go types |
| `validate.go`, `load.go` | Validation, and loading of the definitions in this release |

Each file holds one definition and is named after its `key`.

## Definition format

An entity has `properties`, `statuses`, `relations`, `actions` and `events`. A
value type has `properties` only. A property carries:

| Field | Meaning |
| --- | --- |
| `type` | `string`, `number`, `boolean`, `date`, `datetime`, `enum` or `value` |
| `value` | The value type's key, when `type` is `value` |
| `format` | Narrows a string: `email`, `phone`, `country`, `currency`, `language`, `url` |
| `list` | An array of values; with `many`, each labelled entry holds an array |
| `many` | Several values keyed by a label, such as `billing` and `shipping` addresses |
| `pii`, `secure` | Personal data; masked until revealed |
| `compare` | When two systems' values count as equal, e.g. `case-insensitive` |
| `matchKey` | Identifies the same record across systems |
| `sameAs` | The equivalent term in another vocabulary, such as schema.org |

## Entity identity

Every entity inherits `id` and `external_references` from `base/entity.json`.
Value types have no identity. The loader includes these shared properties in
entity definitions returned by Go and published by the site; entity files
contain only their own properties and cannot redefine the shared keys.

`id` is required and identifies the record within the system holding it.
External references link it to records in other systems:

```json
{
  "id": "customer_123",
  "external_references": {
    "stripe/customer": [
      {"routing_key": "brand_1", "id": "cust_32HDDsd"},
      {"routing_key": "brand_2", "id": "cust_87ABCxy"}
    ],
    "fortifi/customer": [
      {"id": "CST_456"}
    ]
  }
}
```

Keys use `vendor/type`; each key always holds a list, even for one reference.
A reference's optional `routing_key` identifies an account or brand within
that namespace. Omit it when routing is unambiguous. Reference identity is
`(vendor/type, routing_key, id)`, with an omitted routing key representing the
default route. IDs and routing keys are opaque, case-sensitive strings.
External references are optional; omit the property when none are known.

## Using it from Go

```go
catalogue, err := schema.Load()
customer, _ := catalogue.Entity("customer")
```

`Load` returns the definitions released with the module version you depend on,
so an upgrade is a deliberate change to your `go.mod`.

## Building the site

```shell
go run ./cmd/site -out _site
```

This writes a landing page, a page and a JSON document for each type, and
`catalogue.json` for the whole release. Pushing to `master` builds and deploys
it through GitHub Pages.

## Checking a change

```shell
go test ./...
```

The tests load every definition and reject unknown fields, undefined
references and malformed versions. They also build the site, and reject an
example that uses a property its type does not define.
