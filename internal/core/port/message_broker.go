package port

import "context"

// MessageBroker defines event publishing and subscription capabilities.
type MessageBroker interface {
	Publish(ctx context.Context, topic string, payload []byte) error
	Subscribe(ctx context.Context, topic string, handler func(ctx context.Context, payload []byte) error) error
}
