package app

import (
	"context"
	"fmt"
	"log"

	"github.com/EgorLis/my-subs/internal/config"
)

type App struct {
	config *config.Config
	//postgres
	//server
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

func (a *App) Run(ctx context.Context) error {
	log.Println("app: start application...")

	<-ctx.Done()
	log.Println("app: stop application...")

	// graceful stop

	//stopCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	//defer cancel()

	return nil
}
