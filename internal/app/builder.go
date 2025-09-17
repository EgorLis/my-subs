package app

import (
	"context"
	"fmt"
	"log"
	"os"
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
	log    *log.Logger
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
	base := log.New(os.Stdout, "[app] ", log.LstdFlags)

	serverLog := log.New(base.Writer(), base.Prefix()+"[server] ", base.Flags())

	cfg, err := config.LoadFromEnv()
	if err != nil {
		return nil, fmt.Errorf("failed load config: %w", err)
	}

	base.Printf("\n  configuration: %s-------------------", cfg)

	mockDB := mock.NewMockRepo()

	server := web.New(serverLog, cfg, mockDB)

	return &App{
		config: cfg,
		server: server,
		db:     mockDB,
		log:    base,
	}, nil
}

func (a *App) Run(ctx context.Context) error {
	a.log.Println("start application...")

	go a.server.Run()

	<-ctx.Done()
	a.log.Println("stop application...")

	// graceful stop

	stopCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	a.server.Close(stopCtx)

	return nil
}
