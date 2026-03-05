# Consumers

All consumers are registered as `Setup` functions passed to `conn.Start()`. JetStream consumers are used for event and custom streams; Core NATS subscriptions are used for request-reply.

## Consumer Types

### Durable Event Stream Consumer

Subscribes to the `events` JetStream stream. The consumer survives restarts.

```go
nats.EventStreamConsumer("Order.Created",
    func(ctx context.Context, e spec.ConsumableEvent[OrderCreated]) error {
        return processOrder(e.Payload)
    })
```

### Transient Event Stream Consumer

Ephemeral consumer that is removed when the service disconnects:

```go
nats.TransientEventStreamConsumer("Order.*",
    func(ctx context.Context, e spec.ConsumableEvent[any]) error {
        log.Println("event:", e.DeliveryInfo.Key)
        return nil
    })
```

### Custom Stream Consumer

Subscribe to a named stream:

```go
nats.StreamConsumer("audit", "User.Login", handler)
nats.TransientStreamConsumer("audit", "User.*", handler)
```

### Service Request/Response Consumers

See [Request-Response](request-response.md).

## Handler Contract

```go
type EventHandler[T any] func(ctx context.Context, event spec.ConsumableEvent[T]) error
```

### Acknowledgment

| Handler returns | NATS action |
|----------------|-------------|
| `nil` | Message **ACK** — removed from stream |
| `spec.ErrParseJSON` | Message **terminated** — no redelivery |
| `ErrNoMessageTypeForRouteKey` | Message **terminated** — no redelivery |
| Other error | Message **NAK** — redelivered per backoff schedule |

Unlike AMQP where rejected messages are requeued immediately, NATS supports configurable backoff for redelivery.

## Consumer Options

```go
nats.EventStreamConsumer("Order.Created", handler,
    nats.WithMaxDeliver(5),
    nats.WithBackOff(1*time.Second, 5*time.Second, 30*time.Second),
)
```

### Consumer Name Suffix

```go
nats.AddConsumerNameSuffix("priority")
```

Appends a suffix to the durable consumer name. Use when the same service needs multiple consumers for the same routing key.

### Max Delivery Attempts

```go
nats.WithMaxDeliver(5)
```

After 5 delivery attempts, the message is dropped. Without this, messages are redelivered indefinitely.

### Backoff Schedule

```go
nats.WithBackOff(1*time.Second, 5*time.Second, 30*time.Second)
```

Controls the delay between redelivery attempts. The schedule repeats after the last value. Combined with `MaxDeliver` for bounded retry.

### Connection-Level Defaults

Apply default delivery behavior to all consumers:

```go
nats.WithConsumerDefaults(nats.ConsumerDefaults{
    MaxDeliver: 5,
    BackOff:    []time.Duration{1*time.Second, 5*time.Second, 30*time.Second},
})
```

Per-consumer options override these defaults.

## Consumer Grouping

When multiple routing keys are registered on the same stream for the same service, they are automatically grouped into a **single NATS JetStream consumer** with multiple filter subjects:

```go
// Creates ONE consumer with TWO filter subjects
conn.Start(ctx,
    nats.EventStreamConsumer("Order.Created", handler1),
    nats.EventStreamConsumer("Order.Shipped", handler2),
)
```

This matches the AMQP model of one queue with multiple bindings. Messages are dispatched to the correct handler based on subject matching.

## Wildcard Routing

NATS wildcards are used for subject filtering:

| Pattern | Matches |
|---------|---------|
| `Order.Created` | Exactly `Order.Created` |
| `Order.*` | `Order.Created`, `Order.Updated` (one segment) |
| `Order.>` | `Order.Created`, `Order.Item.Added` (any depth) |

The spec translates AMQP `#` to NATS `>` automatically, so you can use either form in your code.

## Dynamic Type Mapping

Handle multiple message types on the same routing pattern:

```go
nats.TypeMappingHandler(genericHandler, func(routingKey string) (reflect.Type, bool) {
    switch routingKey {
    case "Order.Created":
        return reflect.TypeOf(OrderCreated{}), true
    case "Order.Updated":
        return reflect.TypeOf(OrderUpdated{}), true
    default:
        return nil, false
    }
})
```

If the mapper returns `false`, the message is terminated (no redelivery).
