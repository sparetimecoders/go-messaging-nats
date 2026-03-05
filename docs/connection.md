# Connection Lifecycle

## Creating a Connection

A connection requires a **service name** and a **NATS URL**. Optional `nats.Option` values configure the underlying NATS client.

```go
conn, err := nats.NewConnection("order-service", "nats://localhost:4222")
if err != nil {
    log.Fatal(err)
}
```

With NATS client options:

```go
conn, err := nats.NewConnection("order-service", "nats://localhost:4222",
    natsgo.UserInfo("user", "pass"),
    natsgo.MaxReconnects(10),
)
```

The connection is not yet established — that happens in `Start()`.

## Starting

`Start()` connects to NATS, creates JetStream streams, and starts consumers:

```go
err := conn.Start(ctx,
    nats.WithLogger(logger),
    nats.WithStreamDefaults(nats.StreamConfig{MaxAge: 7 * 24 * time.Hour}),
    nats.EventStreamPublisher(pub),
    nats.EventStreamConsumer("Order.Created", handler),
)
```

`Start()` can only be called once. Calling it again returns `ErrAlreadyStarted`.

### What Start Does

1. Connects to the NATS server
2. Creates a JetStream context
3. Runs each `Setup` function — these create streams, consumers, and register handlers
4. Starts consuming from all registered JetStream consumers and Core NATS subscriptions

## Configuration

All configuration is passed as `Setup` functions to `Start()`.

### Logger

```go
nats.WithLogger(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
```

Default: `slog.Default()`. Enriched with `"service"` attribute.

### Request Timeout

```go
nats.WithRequestTimeout(10 * time.Second)
```

Default: 30 seconds. Timeout for Core NATS request-reply operations.

### Tracing

```go
nats.WithTracing(tp)
nats.WithPropagator(propagation.TraceContext{})
```

Defaults: `otel.GetTracerProvider()` and `otel.GetTextMapPropagator()`.

### Custom Span Names

```go
nats.WithSpanNameFn(func(d spec.DeliveryInfo) string {
    return fmt.Sprintf("consume %s", d.Key)
})

nats.WithPublishSpanNameFn(func(stream, routingKey string) string {
    return fmt.Sprintf("publish %s to %s", routingKey, stream)
})
```

## Graceful Shutdown

```go
err := conn.Close()
```

`Close()` drains the NATS connection — it waits for in-flight messages to be processed, then disconnects. This is safer than an abrupt close.

## Topology Export

After `Start()`, inspect the declared topology:

```go
topology := conn.Topology()
errors := spec.Validate(topology)
diagram := spec.Mermaid([]spec.Topology{topology})
```

Without connecting (dry run):

```go
topology, err := nats.CollectTopology("order-service",
    nats.EventStreamPublisher(pub),
    nats.EventStreamConsumer("Order.Created", handler),
)
```
