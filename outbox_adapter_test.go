// MIT License
//
// Copyright (c) 2026 sparetimecoders

package nats

import (
	"context"
	"errors"
	"testing"

	natsgo "github.com/nats-io/nats.go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOutboxRawPublisher_PublishRaw(t *testing.T) {
	t.Run("publishes message with correct subject, payload, and headers", func(t *testing.T) {
		var captured *natsgo.Msg
		var capturedSubject string

		pub := NewPublisher()
		pub.stream = "events"
		pub.publishFn = func(_ context.Context, subject string, msg *natsgo.Msg) error {
			capturedSubject = subject
			captured = msg
			return nil
		}

		adapter := NewOutboxRawPublisher(pub)
		err := adapter.PublishRaw(context.Background(), "order.created", []byte(`{"id":"123"}`), map[string]string{
			"ce-type":   "order.created",
			"ce-source": "order-service",
		})

		require.NoError(t, err)
		require.NotNil(t, captured)

		expectedSubject := subjectName("events", "order.created")
		assert.Equal(t, expectedSubject, capturedSubject)
		assert.Equal(t, expectedSubject, captured.Subject)
		assert.Equal(t, []byte(`{"id":"123"}`), captured.Data)
		assert.Equal(t, "order.created", captured.Header.Get("ce-type"))
		assert.Equal(t, "order-service", captured.Header.Get("ce-source"))
	})

	t.Run("uses custom subjectFn when set", func(t *testing.T) {
		var capturedSubject string

		pub := NewPublisher()
		pub.stream = "audit"
		pub.subjectFn = func(stream, routingKey string) string {
			return stream + ".custom." + routingKey
		}
		pub.publishFn = func(_ context.Context, subject string, msg *natsgo.Msg) error {
			capturedSubject = subject
			return nil
		}

		adapter := NewOutboxRawPublisher(pub)
		err := adapter.PublishRaw(context.Background(), "user.login", nil, nil)

		require.NoError(t, err)
		assert.Equal(t, "audit.custom.user.login", capturedSubject)
	})

	t.Run("returns ErrPublisherNotWired when publishFn is nil", func(t *testing.T) {
		pub := NewPublisher()

		adapter := NewOutboxRawPublisher(pub)
		err := adapter.PublishRaw(context.Background(), "order.created", nil, nil)

		assert.ErrorIs(t, err, ErrPublisherNotWired)
	})

	t.Run("propagates publish errors", func(t *testing.T) {
		pub := NewPublisher()
		pub.stream = "events"
		publishErr := errors.New("connection lost")
		pub.publishFn = func(_ context.Context, _ string, _ *natsgo.Msg) error {
			return publishErr
		}

		adapter := NewOutboxRawPublisher(pub)
		err := adapter.PublishRaw(context.Background(), "order.created", []byte(`{}`), nil)

		assert.ErrorIs(t, err, publishErr)
	})

	t.Run("handles empty headers", func(t *testing.T) {
		var captured *natsgo.Msg

		pub := NewPublisher()
		pub.stream = "events"
		pub.publishFn = func(_ context.Context, _ string, msg *natsgo.Msg) error {
			captured = msg
			return nil
		}

		adapter := NewOutboxRawPublisher(pub)
		err := adapter.PublishRaw(context.Background(), "test.event", []byte(`{}`), map[string]string{})

		require.NoError(t, err)
		assert.Empty(t, captured.Header)
	})
}
