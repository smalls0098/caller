package http

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

func WithServerHost(host string) Option {
	return func(s *Server) { s.host = host }
}
func WithServerPort(port int) Option {
	return func(s *Server) { s.port = port }
}
func WithServerTimeout(timeout time.Duration) Option {
	return func(s *Server) { s.timeout = timeout }
}

type Option func(s *Server)

type Server struct {
	*gin.Engine
	httpSrv *http.Server
	host    string
	port    int
	timeout time.Duration
}

func NewServer(engine *gin.Engine, opts ...Option) *Server {
	s := &Server{
		Engine: engine,
		host:   "0.0.0.0",
		port:   9999,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

func (s *Server) Start(ctx context.Context) error {
	s.httpSrv = &http.Server{
		Addr:    fmt.Sprintf("%s:%d", s.host, s.port),
		Handler: s,
	}
	if err := s.httpSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Printf("listen: %s", err)
		return err
	}
	return nil
}

func (s *Server) Stop(ctx context.Context) error {
	log.Print("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), s.timeout)
	defer cancel()
	if err := s.httpSrv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown: ", err)
		return err
	}
	log.Print("Server exiting")
	return nil
}
