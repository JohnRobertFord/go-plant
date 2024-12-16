package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"

	"github.com/JohnRobertFord/go-plant/internal/server"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func MetricRouter() chi.Router {
	m := server.NewMemStorage()

	r := chi.NewRouter()
	r.Use(server.Logging, server.Middleware)
	r.Use(middleware.SetHeader("Content-Type", "text/plain"))
	r.Get("/", m.GetAll)
	r.Route("/update/", func(r chi.Router) {
		r.Post("/", m.WriteJSONMetric)
		r.Post("/{MT}/{M}/{V}", m.WriteMetric)
	})
	r.Route("/value/", func(r chi.Router) {
		r.Post("/", m.GetJSONMetric)
		r.Get("/{MT}/{M}", m.GetMetric)
	})

	return r
}

func main() {

	bind := flag.String("a", ":8080", "adderss and port to run server, or use env ADDRESS")
	envAddr := os.Getenv("ADDRESS")
	flag.Parse()

	if envAddr == "" {
		envAddr = *bind
	}

	httpServer := &http.Server{
		Addr:    envAddr,
		Handler: MetricRouter(),
	}

	log.Fatal(httpServer.ListenAndServe())
	httpServer.Shutdown(context.Background())

}
