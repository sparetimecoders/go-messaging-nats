# Observability

go-messaging-nats provides three observability layers: distributed tracing (OpenTelemetry), metrics (Prometheus), and event notifications.

## Tracing

### Setup

```go
import (
    "go.opentelemetry.io/otel/propagation"
    sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

tp := sdktrace.NewTracerProvider(sdktrace.WithBatcher(exporter))

conn.Start(ctx,
    nats.WithTracing(tp),
    nats.WithPropagator(propagation.TraceContext{}),
    // ...
)
```

### Span Attributes

| Attribute | Value |
|-----------|-------|
| `messaging.system` | `nats` |
| `messaging.operation` | `publish` or `receive` |
| `messaging.destination.name` | Stream or subject name |
| `messaging.nats.subject` | Full NATS subject |
| `messaging.destination.subscription` | Consumer name (consume only) |
| `messaging.message.id` | UUID |
| `messaging.message.body.size` | Payload size in bytes |
| `messaging.client_id` | Service name |

Trace context propagates through NATS message headers automatically.

## Metrics

### Setup

```go
err := nats.InitMetrics(prometheus.DefaultRegisterer)
```

With routing key normalization:

```go
nats.InitMetrics(prometheus.DefaultRegisterer,
    nats.WithRoutingKeyMapper(func(key string) string {
        return uuidPattern.ReplaceAllString(key, "<id>")
    }),
)
```

### Available Metrics

**Counters:**

| Metric | Labels | Description |
|--------|--------|-------------|
| `nats_events_received` | `consumer`, `routing_key` | Messages received |
| `nats_events_ack` | `consumer`, `routing_key` | Messages acknowledged |
| `nats_events_nak` | `consumer`, `routing_key` | Messages rejected |
| `nats_events_without_handler` | `consumer`, `routing_key` | No matching handler |
| `nats_events_not_parsable` | `consumer`, `routing_key` | JSON parse failures |
| `nats_events_publish_succeed` | `stream`, `routing_key` | Successful publishes |
| `nats_events_publish_failed` | `stream`, `routing_key` | Failed publishes |

**Histograms (milliseconds):**

| Metric | Labels | Description |
|--------|--------|-------------|
| `nats_events_processed_duration` | `consumer`, `routing_key`, `result` | Processing time |
| `nats_events_publish_duration` | `stream`, `routing_key`, `result` | Publish time |

Note: NATS metrics use `consumer` and `stream` labels (vs AMQP's `queue` and `exchange`).

## Notifications

```go
notifyCh := make(chan spec.Notification, 100)
errorCh := make(chan spec.ErrorNotification, 100)

conn.Start(ctx,
    nats.WithNotificationChannel(notifyCh),
    nats.WithErrorChannel(errorCh),
    // ...
)

go func() {
    for n := range notifyCh {
        log.Printf("processed %s in %dms", n.DeliveryInfo.Key, n.Duration)
    }
}()

go func() {
    for e := range errorCh {
        log.Printf("failed %s: %v", e.DeliveryInfo.Key, e.Error)
    }
}()
```

## Comparison with AMQP Observability

The metrics and tracing interfaces are identical across transports, with only the prefix and label names differing:

| | AMQP | NATS |
|---|------|------|
| Metric prefix | `amqp_` | `nats_` |
| Consumer label | `queue` | `consumer` |
| Publisher label | `exchange` | `stream` |
| Tracing system | `rabbitmq` | `nats` |

This means dashboards and alerts can use the same structure for both transports.
