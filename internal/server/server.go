package server

import (
	"context"
	"log"
	"net/http"
	"strings"

	"github.com/JohnRobertFord/go-plant/internal/compress"
	"github.com/JohnRobertFord/go-plant/internal/config"
	"github.com/JohnRobertFord/go-plant/internal/handler"
	"github.com/JohnRobertFord/go-plant/internal/logger"
	"github.com/JohnRobertFord/go-plant/internal/storage/metrics"
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
	r.Use(logger.Logging, compress.GzipMiddleware, Middleware)

	r.Get("/", handler.GetAll(ms))
	r.Get("/ping", handler.Ping(ms))
	r.Post("/updates/", handler.WriteJSONMetric(ms))
	r.Route("/update/", func(r chi.Router) {
		r.Post("/", handler.WriteJSONMetric(ms))
		r.Post("/{MetricType}/{MetricID}/{MetricValue}", handler.WriteMetric(ms))
	})
	r.Route("/value/", func(r chi.Router) {
		r.Post("/", handler.GetJSONMetric(ms))
		r.Get("/{MetricType}/{MetricID}", handler.GetMetric(ms))
	})

	return &server{
		Server: &http.Server{
			Addr:    cfg.Bind,
			Handler: r,
		},
		storage: ms,
	}
}
func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {

		path := strings.Split(req.URL.Path, "/")
		if req.Method == http.MethodPost && strings.Contains(path[1], "update") && len(path) == 3 {
			// check valid REQUEST
		} else if req.Method == http.MethodPost && path[1] == "value" && len(path) == 3 {
			// check valid REQUEST
		} else if req.Method == http.MethodPost {
			req.Header.Set("Accept", "*/*")
			if len(path) != 5 {
				http.Error(w, "Not Found", http.StatusNotFound)
				return
			}

			val := path[4]
			if (strings.Compare(path[2], "counter") != 0 || !metrics.IsCounter(val)) &&
				(strings.Compare(path[2], "gauge") != 0 || !metrics.IsGauge(val)) {
				http.Error(w, "Bad Request!", http.StatusBadRequest)
				return
			}
		} else if req.Method == http.MethodGet {
			if len(path) != 4 && len(path) != 2 {
				http.Error(w, "Not Found", http.StatusNotFound)
				return
			}
			if (len(path) > 2) &&
				(strings.Compare(path[2], "counter") != 0) &&
				(strings.Compare(path[2], "gauge") != 0) {
				http.Error(w, "Bad Request!", http.StatusBadRequest)
				return
			}
		} else {
			http.Error(w, "Only POST or GET requests are allowed!", http.StatusMethodNotAllowed)
			return
		}
		next.ServeHTTP(w, req)
	})
}
