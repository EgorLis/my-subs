package domain

import (
	"context"
	"errors"
	"time"
)

var ErrNotFound = errors.New("subscription not found")

type SubscriptionRepository interface {
	Ping(ctx context.Context) error
	Close()
	AddSub(ctx context.Context, sub Subscription) (Subscription, error)
	UpdateSub(ctx context.Context, sub Subscription) error
	DeleteSub(ctx context.Context, id string) error
	GetSub(ctx context.Context, id string) (Subscription, error)
	ListSubs(ctx context.Context) ([]Subscription, error)
	TotalCost(ctx context.Context, serviceName, userID string, start, end time.Time) (int, error)
}
