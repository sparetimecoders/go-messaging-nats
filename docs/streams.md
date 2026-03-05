# Streams & Retention

Event and custom stream patterns use **JetStream** for durable message storage. Streams must have retention limits configured to prevent unbounded growth.

## Stream Creation

Streams are created automatically during `Start()` when publishers or consumers are registered. If a stream already exists, its configuration is updated to match.

All streams use **FileStorage** for durability.

## Retention Configuration

### Default Retention

Apply default limits to all streams:

```go
nats.WithStreamDefaults(nats.StreamConfig{
    MaxAge:   7 * 24 * time.Hour,  // 7 days
    MaxBytes: 1 << 30,              // 1 GiB
    MaxMsgs:  1_000_000,            // 1 million messages
})
```

### Per-Stream Overrides

Override defaults for specific streams:

```go
nats.WithStreamConfig("audit", nats.StreamConfig{
    MaxAge:   90 * 24 * time.Hour,  // 90 days (compliance)
    MaxBytes: 10 << 30,             // 10 GiB
    MaxMsgs:  0,                    // unlimited
})
```

Per-stream config replaces the defaults entirely for that stream.

### Warning on Missing Limits

If a stream is created without any retention limits (no MaxAge, MaxBytes, or MaxMsgs), a warning is logged:

```
WARN stream created without retention limits stream=events
```

Always set at least one limit to prevent unbounded disk usage.

## StreamConfig Fields

| Field | Type | Zero value | Description |
|-------|------|------------|-------------|
| `MaxAge` | `time.Duration` | unlimited | Delete messages older than this |
| `MaxBytes` | `int64` | unlimited | Delete oldest messages when total size exceeds this |
| `MaxMsgs` | `int64` | unlimited | Delete oldest messages when count exceeds this |

When multiple limits are set, whichever is hit first triggers cleanup.

## Subject Mapping

Each stream accepts messages on subjects matching `{stream}.>`:

| Stream | Subjects | Example |
|--------|----------|---------|
| `events` | `events.>` | `events.Order.Created`, `events.Payment.Completed` |
| `audit` | `audit.>` | `audit.User.Login`, `audit.Config.Changed` |

## Consumer Configuration

JetStream consumers are created with explicit acknowledgment. Consumers are durable by default (survive restarts) unless registered as transient.

### Redelivery

Configure how failed messages are retried:

```go
// Connection-level defaults
nats.WithConsumerDefaults(nats.ConsumerDefaults{
    MaxDeliver: 5,
    BackOff:    []time.Duration{1*time.Second, 5*time.Second, 30*time.Second},
})

// Per-consumer override
nats.EventStreamConsumer("Order.Created", handler,
    nats.WithMaxDeliver(10),
    nats.WithBackOff(500*time.Millisecond, 2*time.Second, 10*time.Second),
)
```

| Setting | Default | Description |
|---------|---------|-------------|
| `MaxDeliver` | unlimited | Max delivery attempts before dropping message |
| `BackOff` | immediate | Delay between redelivery attempts |

The backoff schedule repeats the last value for attempts beyond the schedule length.

## Comparison with AMQP

| | AMQP | NATS JetStream |
|---|------|----------------|
| Storage | Quorum queues | JetStream streams (file) |
| Retention | TTL per queue (5 days default) | MaxAge, MaxBytes, MaxMsgs per stream |
| Redelivery | Immediate requeue on NACK | Configurable backoff schedule |
| Max attempts | Unlimited (use dead letter exchange) | MaxDeliver setting |
| Consumer isolation | One queue per service | One durable consumer per service |
| Multiple routing keys | Multiple queue bindings | Multiple filter subjects on one consumer |
