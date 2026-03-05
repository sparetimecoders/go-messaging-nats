# Publishers

Publishers send messages to NATS subjects. JetStream is used for event and custom streams; Core NATS is used for request-reply.

## Creating a Publisher

```go
pub := nats.NewPublisher()
```

## Wiring to Streams

### Event Stream Publisher

Publishes to the `events` JetStream stream:

```go
conn.Start(ctx,
    nats.EventStreamPublisher(pub),
)

pub.Publish(ctx, "Order.Created", OrderCreated{OrderID: "abc-123"})
// Publishes to subject: events.Order.Created
```

### Custom Stream Publisher

Publishes to a named JetStream stream:

```go
conn.Start(ctx,
    nats.StreamPublisher("audit", pub),
)

pub.Publish(ctx, "User.Login", UserLogin{UserID: "u-42"})
// Publishes to subject: audit.User.Login
```

### Service Request Publisher

Publishes to a service's request subject using Core NATS request-reply:

```go
conn.Start(ctx,
    nats.ServicePublisher("billing-service", pub),
)

pub.Publish(ctx, "Invoice.Create", InvoiceRequest{OrderID: "abc-123"})
// Publishes to subject: billing-service.request.Invoice.Create
```

## Publishing Messages

```go
err := pub.Publish(ctx, "Order.Created", OrderCreated{
    OrderID: "abc-123",
    Amount:  42,
})
```

### What Publish Does

1. Serializes `msg` to JSON
2. Generates a UUID message ID
3. Sets CloudEvents headers (`ce_specversion`, `ce_type`, `ce_source`, `ce_id`, `ce_time`, `ce_datacontenttype`)
4. Injects trace context into NATS headers
5. For JetStream: publishes and waits for stream acknowledgment
6. For Core NATS: sends request and waits for reply (with configurable timeout)

### Custom Headers

```go
pub.Publish(ctx, "Order.Created", payload,
    nats.Header{Key: "priority", Value: "high"},
    nats.Header{Key: "region", Value: "eu-west-1"},
)
```

The reserved header key `"service"` is used internally — do not use it.

## Wire Format

Every published message has these NATS headers:

| Header | Value |
|--------|-------|
| `ce_specversion` | `1.0` |
| `ce_type` | routing key |
| `ce_source` | service name |
| `ce_id` | UUID v4 |
| `ce_time` | RFC 3339 UTC |
| `ce_datacontenttype` | `application/json` |
| `service` | service name |

NATS uses underscore-separated CE header names (not `ce-` or `cloudEvents:`).

## Subject Naming

| Pattern | Subject format | Example |
|---------|---------------|---------|
| Event stream | `events.{routingKey}` | `events.Order.Created` |
| Custom stream | `{stream}.{routingKey}` | `audit.User.Login` |
| Service request | `{service}.request.{routingKey}` | `billing.request.Invoice.Create` |
