# go-messaging-nats Documentation

Detailed guides for the NATS/JetStream transport. For a quick overview, see the [main README](../README.md).

| Guide | Description |
|-------|-------------|
| [Connection Lifecycle](connection.md) | Creating, starting, monitoring, and closing connections |
| [Consumers](consumers.md) | Durable and transient consumers, error handling, redelivery |
| [Publishers](publishers.md) | Publishing messages, custom headers |
| [Request-Response](request-response.md) | Synchronous RPC using Core NATS |
| [Streams & Retention](streams.md) | JetStream stream configuration, retention policies |
| [Observability](observability.md) | OpenTelemetry tracing, Prometheus metrics, notifications |

## Related

- [gomessaging specification](https://github.com/sparetimecoders/messaging) — shared spec, TCK, validation, visualization
- [gomessaging/amqp](https://github.com/sparetimecoders/go-messaging-amqp) — AMQP/RabbitMQ transport
