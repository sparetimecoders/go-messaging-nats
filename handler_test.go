package nats

import (
	"context"
	"encoding/json"
	"testing"

	spec "github.com/sparetimecoders/messaging/specification/spec"
	"github.com/stretchr/testify/assert"
)

func TestNewWrappedHandler_InvalidJSON(t *testing.T) {
	handler := newWrappedHandler(func(ctx context.Context, event spec.ConsumableEvent[testMessage]) error {
		return nil
	})

	evt := unmarshalEvent{
		Metadata:     spec.Metadata{},
		DeliveryInfo: spec.DeliveryInfo{Key: "test.key"},
		Payload:      json.RawMessage([]byte("not valid json")),
	}
	err := handler(context.Background(), evt)
	assert.ErrorIs(t, err, spec.ErrParseJSON)
}

func TestNewWrappedHandler_ValidJSON(t *testing.T) {
	var received testMessage
	handler := newWrappedHandler(func(ctx context.Context, event spec.ConsumableEvent[testMessage]) error {
		received = event.Payload
		return nil
	})

	payload, err := json.Marshal(testMessage{Name: "hello", Value: 42})
	assert.NoError(t, err)

	evt := unmarshalEvent{
		Payload: json.RawMessage(payload),
	}
	err = handler(context.Background(), evt)
	assert.NoError(t, err)
	assert.Equal(t, "hello", received.Name)
	assert.Equal(t, 42, received.Value)
}
