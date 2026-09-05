package domain

import "context"

// Repository and EventPublisher are domain-facing ports. Concrete MySQL/Kafka
// adapters will implement them later without leaking infrastructure inward.
type Repository interface {
	Create(ctx context.Context, order Order) error
	Get(ctx context.Context, orderID string) (Order, error)
}

type EventPublisher interface {
	Publish(ctx context.Context, event Event) error
}

type Order struct {
	ID     string
	UserID string
	Status string
}

type Event struct {
	ID        string
	Type      string
	Aggregate string
}
