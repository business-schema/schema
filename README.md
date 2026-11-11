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

An entity has `properties`, `relations`, `actions` and `events`. A
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

Every entity declares `"base": "entity"` and inherits `id` and
`external_references` from `base/entity.json`. Value types have no identity.
The base is required, and entities cannot redefine its properties.

Source files, published JSON definitions and `Catalogue.Entities` contain
only each entity's own properties and its base reference. The site publishes
the shared definition at `entity.json` and shows inherited properties on each
entity's page. In Go, `Catalogue.Entity(key)` returns the entity with its
inherited properties resolved.

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

## Customer relationship and account state

Customer covers potential, current and former customers. `party_type`
distinguishes an `individual` from an `organization`. `primary_contact`
holds the individual themselves or the main representative of an organization,
using the reusable `person` value type.

These independent enums describe the relationship and account:

| Property | Values | Meaning |
| --- | --- | --- |
| `lifecycle_stage` | `lead`, `prospect`, `customer`, `former_customer`, `disqualified` | Overall relationship; a prospect is a qualified lead |
| `lead_temperature` | `cold`, `warm`, `hot` | Current interest or engagement |
| `qualification_stage` | `unqualified`, `marketing_qualified`, `sales_qualified` | Qualification reached |
| `status` | `pending`, `active`, `inactive`, `suspended`, `closed` | Operational account state |
| `commercial_model` | `none`, `free`, `freemium`, `paid` | How the account is offered |
| `billing_stage` | `none`, `trial`, `paying`, `cancelled` | Billing relationship |
| `payment_status` | `none`, `current`, `past_due`, `delinquent` | Whether payment obligations are met |

Set lifecycle and status explicitly. Enum options on the `status` property
are the single definition of account statuses; there is no separate statuses
list or implicit default. Omit optional fields when unknown or inapplicable;
`none` explicitly means no offering, billing relationship or payment obligation.

Temperature and qualification are independent: a sales-qualified prospect
can go cold. A former customer can retain an active account. Free means no
payment is expected, while freemium describes an offering with a free tier
and paid upgrades. A paid offering can be in trial with no payment obligation.
Cancellation does not itself end access or close an account. Past due means
a payment is overdue; delinquent means the account meets the business's
escalation policy. Billing and payment fields are account summaries when
multiple subscriptions exist; each subscription retains its own state.

Use `change_lifecycle` and `change_status` to change those fields; they emit
`lifecycle_changed` and `status_changed`, respectively. `is_test_account`
marks accounts used for testing. `timezone` holds an IANA timezone name such
as `Europe/London`.

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
