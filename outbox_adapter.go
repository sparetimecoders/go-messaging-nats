// MIT License
//
// Copyright (c) 2026 sparetimecoders
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package nats

import (
	"context"
	"errors"

	natsgo "github.com/nats-io/nats.go"
)

// ErrPublisherNotWired is returned when PublishRaw is called on a publisher
// that hasn't been wired to a connection yet.
var ErrPublisherNotWired = errors.New("publisher not wired — call a Setup function first")

// OutboxRawPublisher wraps a NATS Publisher to satisfy the outbox.RawPublisher
// interface. Headers are passed as ce-* prefixed keys (matching NATS CloudEvents
// binding convention) and forwarded as NATS message headers.
type OutboxRawPublisher struct {
	publisher *Publisher
}

// NewOutboxRawPublisher creates an adapter that wraps publisher for use with
// the outbox relay. The publisher must already be wired to a connection.
func NewOutboxRawPublisher(publisher *Publisher) *OutboxRawPublisher {
	return &OutboxRawPublisher{publisher: publisher}
}

// PublishRaw publishes a pre-serialized message to NATS with the given headers.
func (a *OutboxRawPublisher) PublishRaw(ctx context.Context, routingKey string, payload []byte, headers map[string]string) error {
	if a.publisher.publishFn == nil {
		return ErrPublisherNotWired
	}

	fn := subjectName
	if a.publisher.subjectFn != nil {
		fn = a.publisher.subjectFn
	}
	subject := fn(a.publisher.stream, routingKey)

	natsHdrs := natsgo.Header{}
	for k, v := range headers {
		natsHdrs.Set(k, v)
	}

	msg := &natsgo.Msg{
		Subject: subject,
		Data:    payload,
		Header:  natsHdrs,
	}

	return a.publisher.publishFn(ctx, subject, msg)
}
