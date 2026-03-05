# Request-Response

The request-response pattern uses **Core NATS** request-reply (not JetStream). The caller publishes a request and waits for a response on an auto-generated reply subject.

## Architecture

```
order-service ──request──> billing.request.Invoice.Create ──> billing-service
                                                                    │
order-service <──response── (NATS reply subject) <──────────────────┘
```

Unlike AMQP, NATS has built-in request-reply support. No explicit response exchange or queue is needed — NATS handles response routing via reply subjects automatically.

## Handler Side

```go
billingConn.Start(ctx,
    nats.RequestResponseHandler("Invoice.Create",
        func(ctx context.Context, e spec.ConsumableEvent[InvoiceRequest]) (InvoiceResponse, error) {
            invoice, err := createInvoice(e.Payload.OrderID)
            if err != nil {
                return InvoiceResponse{}, err
            }
            return InvoiceResponse{InvoiceID: invoice.ID}, nil
        }),
)
```

`RequestResponseHandler` subscribes to `billing-service.request.Invoice.Create` and automatically sends the return value as a reply.

### Error Handling

| Handler returns | Behavior |
|----------------|----------|
| `(response, nil)` | Response JSON sent to caller's reply subject |
| `(_, error)` | Error JSON sent to caller's reply subject |

Unlike the AMQP transport, errors are sent back as a JSON error object rather than silently NACKing. The caller always gets a response (success or error).

## Caller Side

```go
pub := nats.NewPublisher()

orderConn.Start(ctx,
    nats.ServicePublisher("billing-service", pub),
    nats.ServiceResponseConsumer("billing-service", "Invoice.Create",
        func(ctx context.Context, e spec.ConsumableEvent[InvoiceResponse]) error {
            fmt.Printf("invoice: %s\n", e.Payload.InvoiceID)
            return nil
        }),
)

pub.Publish(ctx, "Invoice.Create", InvoiceRequest{OrderID: "abc-123"})
```

### Request Timeout

```go
nats.WithRequestTimeout(10 * time.Second)
```

Default: 30 seconds. If no response arrives within the timeout, `Publish()` returns an error.

## Comparison with AMQP Request-Response

| | AMQP | NATS |
|---|------|------|
| Request routing | Direct exchange | Core NATS subject |
| Response routing | Headers exchange with `service` header match | Built-in reply subject |
| Response queue | Explicit per-caller queue | Automatic (NATS inbox) |
| Timeout | Application-level | Built into Core NATS request |
| Error response | No response sent, request NACKed | Error JSON sent to reply subject |
| Persistence | Quorum queues (durable) | Not persisted (Core NATS) |

The NATS approach is simpler — no exchange or queue declaration needed for request-reply.
