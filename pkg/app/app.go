package app

import (
	"context"
	"github.com/smalls0098/caller/pkg/app/server"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func WithServer(servers ...server.Server) Option {
	return func(a *App) { a.servers = servers }
}

func WithName(name string) Option {
	return func(a *App) { a.name = name }
}

type Option func(a *App)

type App struct {
	name    string
	servers []server.Server
}

func New(opts ...Option) *App {
	a := &App{
		name:    "test",
		servers: make([]server.Server, 1),
	}
	for _, opt := range opts {
		opt(a)
	}
	return a
}

func (a *App) Run(ctx context.Context) error {
	var cancel context.CancelFunc
	ctx, cancel = context.WithCancel(ctx)
	defer cancel()

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)

	for _, srv := range a.servers {
		go func(srv server.Server) {
			err := srv.Start(ctx)
			if err != nil {
				log.Printf("Server start err: %v", err)
			}
		}(srv)
	}

	select {
	case <-signals:
		// 终止信号
		log.Println("Received termination signal")
	case <-ctx.Done():
		//取消
		log.Println("Context canceled")
	}

	// 优雅停止
	for _, srv := range a.servers {
		err := srv.Stop(ctx)
		if err != nil {
			log.Printf("Server stop err: %v", err)
		}
	}

	return nil
}
