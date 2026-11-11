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
| `many` | Several values keyed by a label, such as `billing` and `shipping` addresses |
| `pii`, `secure` | Personal data; masked until revealed |
| `compare` | When two systems' values count as equal, e.g. `case-insensitive` |
| `matchKey` | Identifies the same record across systems |
| `sameAs` | The equivalent term in another vocabulary, such as schema.org |

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
