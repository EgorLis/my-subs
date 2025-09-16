package app

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/EgorLis/my-subs/internal/config"
	"github.com/EgorLis/my-subs/internal/domain"
	"github.com/EgorLis/my-subs/internal/infra/database/mock"
	"github.com/EgorLis/my-subs/internal/transport/web"
)

type App struct {
	config *config.Config
	db     domain.SubscriptionRepository
	server *web.Server
}

func Build() (*App, error) {
	cfg, err := config.LoadFromEnv()
	if err != nil {
		return nil, fmt.Errorf("failed load config: %w", err)
	}

	log.Printf("app: configuration %s-------------------", cfg)

	return &App{
		config: cfg,
	}, nil
}

func BuildMock() (*App, error) {
	cfg, err := config.LoadFromEnv()
	if err != nil {
		return nil, fmt.Errorf("failed load config: %w", err)
	}

	log.Printf("app: configuration %s-------------------", cfg)

	mockDB := mock.NewMockRepo()

	server := web.New(cfg, mockDB)

	return &App{
		config: cfg,
		server: server,
		db:     mockDB,
	}, nil
}

func (a *App) Run(ctx context.Context) error {
	log.Println("app: start application...")

	go a.server.Run()

	<-ctx.Done()
	log.Println("app: stop application...")

	// graceful stop

	stopCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	a.server.Close(stopCtx)

	return nil
}
