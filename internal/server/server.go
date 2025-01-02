package server

import (
	"context"
	"log"
	"net/http"

	"github.com/JohnRobertFord/go-plant/internal/compress"
	"github.com/JohnRobertFord/go-plant/internal/config"
	"github.com/JohnRobertFord/go-plant/internal/handler"
	"github.com/JohnRobertFord/go-plant/internal/logger"
	"github.com/JohnRobertFord/go-plant/internal/storage/metrics"
	"github.com/JohnRobertFord/go-plant/internal/storage/metrics/cache"
	"github.com/go-chi/chi"
)

type server struct {
	Server  *http.Server
	storage metrics.Storage
}

func (s server) RunServer() {
	log.Fatal(s.Server.ListenAndServe())
	s.Server.Shutdown(context.Background())
}

func NewMetricServer(cfg *config.Config, ms metrics.Storage) *server {
	r := chi.NewRouter()
	r.Use(logger.Logging, compress.GzipMiddleware, cache.Middleware)

	// r.MethodNotAllowed()

	r.Get("/", handler.GetAll(ms))
	r.Get("/ping", handler.Ping(ms))
	r.Route("/update/", func(r chi.Router) {
		r.Post("/", handler.WriteJSONMetric(ms))
		r.Post("/{MT}/{M}/{V}", handler.WriteMetric(ms))
	})
	r.Route("/value/", func(r chi.Router) {
		r.Post("/", handler.GetJSONMetric(ms))
		r.Get("/{MT}/{M}", handler.GetMetric(ms))
	})

	return &server{
		Server: &http.Server{
			Addr:    cfg.Bind,
			Handler: r,
		},
		storage: ms,
	}
}
