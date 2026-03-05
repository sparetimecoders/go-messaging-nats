# go-messaging-nats

<p align="center">
  <strong>NATS/JetStream transport for the gomessaging framework.</strong>
</p>

<p align="center">
  <a href="https://github.com/sparetimecoders/go-messaging-nats/actions"><img alt="CI" src="https://github.com/sparetimecoders/go-messaging-nats/actions/workflows/ci.yml/badge.svg"></a>
  <a href="https://pkg.go.dev/github.com/sparetimecoders/go-messaging-nats"><img alt="Go Reference" src="https://pkg.go.dev/badge/github.com/sparetimecoders/go-messaging-nats.svg"></a>
  <a href="LICENSE"><img alt="License: MIT" src="https://img.shields.io/badge/license-MIT-blue.svg"></a>
</p>

---

This package implements the [gomessaging specification](https://github.com/sparetimecoders/messaging) for NATS. It uses **JetStream** for durable event streams and custom streams, and **Core NATS** for request-reply patterns. The API mirrors the AMQP transport -- swap `amqp` for `nats` and it works.

## Installation

```bash
go get github.com/sparetimecoders/go-messaging-nats
```

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "log"

    nats "github.com/sparetimecoders/go-messaging-nats"
    "github.com/sparetimecoders/gomessaging/spec"
)

type OrderCreated struct {
    OrderID string `json:"order_id"`
    Amount  int    `json:"amount"`
}

func main() {
    ctx := context.Background()

    conn, err := nats.NewConnection("order-service", "nats://localhost:4222")
    if err != nil {
        log.Fatal(err)
    }
    defer conn.Close()

    pub := nats.NewPublisher()

    err = conn.Start(ctx,
        nats.EventStreamPublisher(pub),
        nats.EventStreamConsumer("Order.Created", func(ctx context.Context, e spec.ConsumableEvent[OrderCreated]) error {
            fmt.Printf("Order %s, amount %d\n", e.Payload.OrderID, e.Payload.Amount)
            return nil
        }),
    )
    if err != nil {
        log.Fatal(err)
    }

    err = pub.Publish(ctx, "Order.Created", OrderCreated{OrderID: "abc-123", Amount: 42})
    if err != nil {
        log.Fatal(err)
    }
}
```

## Messaging Patterns

### Event Stream

Publish domain events to the default `events` JetStream stream. Any number of services subscribe by routing key with durable or ephemeral consumers.

```go
pub := nats.NewPublisher()

conn.Start(ctx,
    // Publisher
    nats.EventStreamPublisher(pub),

    // Durable consumer -- survives restarts
    nats.EventStreamConsumer("Order.Created", func(ctx context.Context, e spec.ConsumableEvent[OrderCreated]) error {
        return processOrder(e.Payload)
    }),

    // Ephemeral consumer -- auto-deleted on disconnect
    nats.TransientEventStreamConsumer("Order.*", func(ctx context.Context, e spec.ConsumableEvent[json.RawMessage]) error {
        return logEvent(e)
    }),
)

pub.Publish(ctx, "Order.Created", OrderCreated{OrderID: "abc-123", Amount: 42})
```

### Custom Stream

Same as event stream but on a named JetStream stream instead of the default `events` stream. Use when events belong to a separate domain.

```go
pub := nats.NewPublisher()

conn.Start(ctx,
    nats.StreamPublisher("audit", pub),
    nats.StreamConsumer("audit", "User.Login", func(ctx context.Context, e spec.ConsumableEvent[UserLogin]) error {
        return recordLogin(e.Payload)
    }),
    nats.TransientStreamConsumer("audit", "User.*", func(ctx context.Context, e spec.ConsumableEvent[json.RawMessage]) error {
        return logEvent(e)
    }),
)
```

### Service Request-Response

Synchronous request-reply between services using Core NATS. The handler receives the request, returns a response, and the reply is sent automatically.

```go
// Server side: handle incoming requests
conn.Start(ctx,
    nats.RequestResponseHandler[BillingRequest, BillingResponse]("Billing.Charge",
        func(ctx context.Context, e spec.ConsumableEvent[BillingRequest]) (BillingResponse, error) {
            return BillingResponse{TransactionID: "txn-456"}, nil
        },
    ),
)

// Client side: send request to the billing service
pub := nats.NewPublisher()
clientConn.Start(ctx,
    nats.ServicePublisher("billing", pub),
)
pub.Publish(ctx, "Billing.Charge", BillingRequest{Amount: 100})
```

`ServicePublisher` uses Core NATS request-reply with a configurable timeout (default 30s, see `WithRequestTimeout`).

### Service Response Consumer

Register a consumer for topology tracking of service responses. In NATS, the actual response delivery is handled by the Core NATS reply mechanism, so this setup exists for topology registration and validation.

```go
conn.Start(ctx,
    nats.ServiceResponseConsumer[BillingResponse]("billing", "Billing.Charge",
        func(ctx context.Context, e spec.ConsumableEvent[BillingResponse]) error {
            return handleResponse(e.Payload)
        },
    ),
)
```

## Configuration

### Connection Options

Setup functions configure the connection before or during `Start`. Pass them as arguments to `conn.Start(ctx, ...)`.

| Function | Description |
|----------|-------------|
| `WithLogger(logger)` | Set a custom `*slog.Logger` for structured logging |
| `WithTracing(tp)` | Set an OpenTelemetry `TracerProvider` |
| `WithPropagator(p)` | Set an OpenTelemetry `TextMapPropagator` for context propagation |
| `WithSpanNameFn(fn)` | Custom function for consumer span names; receives `spec.DeliveryInfo` |
| `WithPublishSpanNameFn(fn)` | Custom function for publish span names; receives `(stream, routingKey)` |
| `WithRequestTimeout(d)` | Timeout for Core NATS request-reply (default 30s) |
| `WithStreamDefaults(cfg)` | Default `StreamConfig` applied to all streams |
| `WithStreamConfig(stream, cfg)` | Override `StreamConfig` for a specific stream |
| `WithConsumerDefaults(cfg)` | Default `ConsumerDefaults` for all JetStream consumers |
| `WithNotificationChannel(ch)` | Channel to receive `spec.Notification` on handler success |
| `WithErrorChannel(ch)` | Channel to receive `spec.ErrorNotification` on handler failure |

### Stream Configuration

Configure retention limits for JetStream streams via `StreamConfig`:

```go
type StreamConfig struct {
    MaxAge   time.Duration // Maximum age of messages (0 = unlimited)
    MaxBytes int64         // Maximum total bytes (0 = unlimited)
    MaxMsgs  int64         // Maximum number of messages (0 = unlimited)
}
```

Apply defaults to all streams, then override per stream:

```go
conn.Start(ctx,
    nats.WithStreamDefaults(nats.StreamConfig{
        MaxAge:   24 * time.Hour,
        MaxBytes: 1 << 30, // 1 GiB
    }),
    nats.WithStreamConfig("audit", nats.StreamConfig{
        MaxAge: 90 * 24 * time.Hour, // 90 days for audit
    }),
    nats.EventStreamPublisher(pub),
)
```

If no retention limits are set for a stream, a warning is logged at startup.

### Consumer Options

Per-consumer options are passed as trailing arguments to consumer setup functions:

| Option | Description |
|--------|-------------|
| `AddConsumerNameSuffix(suffix)` | Append a suffix to the durable consumer name |
| `WithMaxDeliver(n)` | Maximum delivery attempts before the message is terminated |
| `WithBackOff(durations...)` | Redelivery backoff schedule between attempts |

Connection-level defaults apply to all JetStream consumers unless overridden:

```go
conn.Start(ctx,
    nats.WithConsumerDefaults(nats.ConsumerDefaults{
        MaxDeliver: 5,
        BackOff:    []time.Duration{1 * time.Second, 5 * time.Second, 30 * time.Second},
    }),
    nats.EventStreamConsumer("Order.Created", handler),
    nats.EventStreamConsumer("Order.Shipped", handler,
        nats.WithMaxDeliver(10), // override for this consumer
    ),
)
```

## Observability

### Tracing

OpenTelemetry spans are created for every publish and consume operation. Configure a `TracerProvider` and `TextMapPropagator` to enable distributed tracing:

```go
import (
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/propagation"
    sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

tp := sdktrace.NewTracerProvider(
    sdktrace.WithBatcher(exporter),
)
defer tp.Shutdown(ctx)

conn.Start(ctx,
    nats.WithTracing(tp),
    nats.WithPropagator(propagation.TraceContext{}),
    nats.EventStreamPublisher(pub),
    nats.EventStreamConsumer("Order.Created", handler),
)
```

Span attributes follow OpenTelemetry semantic conventions: `messaging.system`, `messaging.operation`, `messaging.destination.name`, `messaging.nats.subject`, `messaging.message.id`, `messaging.message.body.size`.

### Metrics

Register Prometheus metrics once at startup with `InitMetrics`:

```go
import "github.com/prometheus/client_golang/prometheus"

err := nats.InitMetrics(prometheus.DefaultRegisterer)
```

Use `WithRoutingKeyMapper` to normalize dynamic routing key segments and prevent unbounded label cardinality:

```go
nats.InitMetrics(prometheus.DefaultRegisterer, nats.WithRoutingKeyMapper(func(key string) string {
    // Replace UUIDs in routing keys with a placeholder
    return uuidRegex.ReplaceAllString(key, "{id}")
}))
```

All metrics use the `nats_` prefix:

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `nats_events_received` | counter | consumer, routing_key | Events received |
| `nats_events_ack` | counter | consumer, routing_key | Events acknowledged |
| `nats_events_nak` | counter | consumer, routing_key | Events negatively acknowledged (redelivery) |
| `nats_events_without_handler` | counter | consumer, routing_key | Events with no matching handler |
| `nats_events_not_parsable` | counter | consumer, routing_key | Events that failed JSON parsing |
| `nats_events_processed_duration` | histogram | consumer, routing_key, result | Processing time in milliseconds |
| `nats_events_publish_succeed` | counter | stream, routing_key | Successful publishes |
| `nats_events_publish_failed` | counter | stream, routing_key | Failed publishes |
| `nats_events_publish_duration` | histogram | stream, routing_key, result | Publish time in milliseconds |

### Notifications

Monitor handler outcomes programmatically via channels:

```go
notifications := make(chan spec.Notification, 100)
errors := make(chan spec.ErrorNotification, 100)

conn.Start(ctx,
    nats.WithNotificationChannel(notifications),
    nats.WithErrorChannel(errors),
    nats.EventStreamConsumer("Order.Created", handler),
)

go func() {
    for n := range notifications {
        log.Printf("success: %s %s (%dms)", n.DeliveryInfo.Key, n.DeliveryInfo.Destination, n.Duration)
    }
}()

go func() {
    for e := range errors {
        log.Printf("failure: %s %s: %v", e.DeliveryInfo.Key, e.DeliveryInfo.Destination, e.Error)
    }
}()
```

## NATS-Specific Features

### JetStream vs Core NATS

The transport automatically selects the protocol based on the messaging pattern:

| Pattern | Protocol | Why |
|---------|----------|-----|
| Event stream | JetStream | Durable streams with at-least-once delivery and replay |
| Custom stream | JetStream | Same as event stream on a named stream |
| Service request | Core NATS | Built-in request-reply with automatic response routing |
| Service response | Core NATS | Reply subjects handled by the NATS client |

### Stream Retention Policies

JetStream streams are created with `FileStorage` and configurable retention limits (`MaxAge`, `MaxBytes`, `MaxMsgs`). Streams without any retention limits produce a warning at startup. Per-stream overrides replace connection-level defaults entirely (no field-level merging).

### Consumer Delivery Limits and Backoff

JetStream consumers support `MaxDeliver` to cap redelivery attempts and `BackOff` to schedule delays between attempts. When a handler returns an error:

- `spec.ErrParseJSON` or `ErrNoMessageTypeForRouteKey` -- message is **terminated** (no redelivery)
- Other errors -- message is **nacked** for redelivery according to the backoff schedule

### Consumer Grouping

When multiple routing keys are registered on the same stream for the same service, they are grouped into a single NATS JetStream consumer with multiple filter subjects. This matches the AMQP pattern of one queue with multiple routing key bindings.

```go
// These two consumers share a single NATS durable consumer with two filter subjects
conn.Start(ctx,
    nats.EventStreamConsumer("Order.Created", handler1),
    nats.EventStreamConsumer("Order.Shipped", handler2),
)
```

## Topology Export

Export the declared messaging topology for validation or visualization without connecting to a broker:

```go
// From a live connection
topology := conn.Topology()

// Without connecting to NATS
topology, err := nats.CollectTopology("order-service",
    nats.EventStreamPublisher(nats.NewPublisher()),
    nats.EventStreamConsumer("Order.Created", handler),
)
```

The returned `spec.Topology` can be passed to `spec.Validate`, `spec.ValidateTopologies`, or `spec.Mermaid` from the [specification module](https://github.com/sparetimecoders/messaging).

## Development

```bash
# Start NATS with JetStream
docker compose up -d

# Run tests
go test ./...

# Vet
go vet ./...
```

## TCK Adapter

The `cmd/tck-adapter/` directory contains the reference adapter for the gomessaging [Technology Compatibility Kit](https://github.com/sparetimecoders/messaging#technology-compatibility-kit-tck). It implements the JSON-RPC subprocess protocol and exercises all supported messaging patterns against a real NATS broker.

## License

MIT
