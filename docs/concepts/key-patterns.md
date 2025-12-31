# Key patterns and taxonomy

Policy keys are part of the API contract. They should be stable, low-cardinality, and reviewed like surface area.

## Key shape

Use `namespace.name`, which `policy.ParseKey` splits into `Namespace` and `Name`.

- Namespace: stable boundary (service, client, subsystem).
- Name: operation inside that boundary.

Good:

- `"payments.Charge"`
- `"user-service.GetUser"`
- `"queue.orders.Publish"`

Avoid:

- `"GET /users/123"`
- `"user-service.GetUser?user_id=123"`

## Patterns to prefer

- Service or client methods: `"billing.ChargeCard"`, `"inventory.GetItem"`.
- Protocol specific operations: `"http.GetUser"`, `"grpc.UserService.GetUser"`.
- Storage tiers: `"db.Read"`, `"db.Write"`, `"db.Transaction"`.
- Queues or workflows: `"queue.orders.Publish"`, `"queue.orders.Consume"`.

## Keep cardinality bounded

Do not include high-cardinality values:

- IDs, UUIDs, emails, or raw tenant identifiers.
- Full URLs, query strings, or raw SQL.
- Hostnames, pod names, shard IDs, or per-instance labels.
- Per-request flags or trace IDs.

If you need a dimension, bucket it:

- Latency tier: `"tier.low"`, `"tier.high"`.
- Customer tier: `"plan.free"`, `"plan.paid"`.
- Region: `"region.us"`, `"region.eu"` (only if you run distinct policies per region).

## Key change discipline

- Treat new keys like API changes; coordinate with policy owners and dashboards.
- Avoid renames without migrating policies and telemetry.
- If you split a key, roll it out in stages and monitor both policies.

## Review checklist

- The key is stable across deploys.
- The number of possible keys is bounded.
- The key matches the policy and budget boundary.
- The key has an owner and is registered in policy sources.

See also [Policy keys](policy-keys.md) and [Gotchas and safety checklist](../gotchas.md).
