package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/EgorLis/my-subs/internal/app"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(),
		os.Interrupt, syscall.SIGTERM)
	defer stop()

	a, err := app.Build(ctx)
	if err != nil {
		log.Println("app build error:")
		panic(err)
	}

	if err := a.Run(ctx); err != nil {
		log.Println("app error:", err)
	}
}
