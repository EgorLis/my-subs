package mock

import (
	"context"
	"sync"
	"time"

	"github.com/EgorLis/my-subs/internal/domain"
	"github.com/google/uuid"
)

type Repo struct {
	mu    sync.RWMutex
	items map[string]domain.Subscription
}

func NewMockRepo() *Repo {
	return &Repo{
		items: make(map[string]domain.Subscription),
	}
}

func (r *Repo) Ping(ctx context.Context) error {
	// для мока всегда "живой"
	return nil
}

func (r *Repo) AddSub(ctx context.Context, sub domain.Subscription) (domain.Subscription, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	sub.ID = uuid.NewString()
	if sub.StartDate.IsZero() {
		sub.StartDate = time.Now()
	}
	r.items[sub.ID] = sub
	return sub, nil
}

func (r *Repo) UpdateSub(ctx context.Context, sub domain.Subscription) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.items[sub.ID]; !ok {
		return domain.ErrNotFound
	}
	r.items[sub.ID] = sub
	return nil
}

func (r *Repo) DeleteSub(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.items[id]; !ok {
		return domain.ErrNotFound
	}
	delete(r.items, id)
	return nil
}

func (r *Repo) GetSub(ctx context.Context, id string) (domain.Subscription, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	sub, ok := r.items[id]
	if !ok {
		return domain.Subscription{}, domain.ErrNotFound
	}
	return sub, nil
}

func (r *Repo) ListSubs(ctx context.Context) ([]domain.Subscription, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	out := make([]domain.Subscription, 0, len(r.items))
	for _, v := range r.items {
		out = append(out, v)
	}
	return out, nil
}
