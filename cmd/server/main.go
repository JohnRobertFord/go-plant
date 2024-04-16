package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func init() {
	log.Println("Initialization...")
}

type gauge float64
type counter int64
type MemStorage struct {
	mapa map[string]any
}

func NewMemStorage() *MemStorage {
	return &MemStorage{
		mapa: make(map[string]any),
	}
}

func (m *MemStorage) WriteMetric(w http.ResponseWriter, req *http.Request) {

	input := chi.URLParam(req, "V")
	metric := chi.URLParam(req, "M")
	metrictype := chi.URLParam(req, "MT")

	switch metrictype {
	case "gauge":
		if f64, err := strconv.ParseFloat(input, 64); err == nil {
			m.mapa[metric] = gauge(f64)
		}
	case "counter":
		if i64, err := strconv.ParseInt(input, 10, 64); err == nil {
			if c, ok := m.mapa[metric].(counter); ok {
				c += counter(i64)
				m.mapa[metric] = c
			} else {
				m.mapa[metric] = counter(i64)
			}
		}
	}
	// log.Println(time.Now().Second())
	w.WriteHeader(http.StatusOK)
}

func (m *MemStorage) GetMetric(w http.ResponseWriter, req *http.Request) {

	metric := chi.URLParam(req, "M")
	res, ok := m.mapa[metric]
	if !ok {
		http.Error(w, "Metric Not Found", http.StatusNotFound)
		return
	}

	io.WriteString(w, fmt.Sprintf("%v", res))
}

func (m *MemStorage) GetAll(w http.ResponseWriter, req *http.Request) {
	var list []string
	for k := range m.mapa {
		list = append(list, k)
	}

	io.WriteString(w, strings.Join(list, ", "))
}

func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {

		path := strings.Split(req.URL.Path, "/")

		if req.Method == http.MethodPost {

			req.Header.Set("Accept", "*/*")
			if len(path) != 5 {
				http.Error(w, "Not Found", http.StatusNotFound)
				return
			}
			val := path[4]

			if (strings.Compare(path[2], "counter") != 0 || !IsCounter(val)) &&
				(strings.Compare(path[2], "gauge") != 0 || !IsGauge(val)) {
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

func IsCounter(input string) bool {
	if _, err := strconv.ParseInt(input, 10, 64); err == nil {
		return true
	}
	return false
}

func IsGauge(input string) bool {
	if _, err := strconv.ParseFloat(input, 64); err == nil {
		return true
	}
	return false
}

func MetricRouter() chi.Router {
	m := NewMemStorage()

	r := chi.NewRouter()
	r.Use(Middleware)
	r.Use(middleware.SetHeader("Content-Type", "text/plain"))
	r.Get("/", m.GetAll)
	r.Route("/update", func(r chi.Router) {
		r.Post("/{MT}/{M}/{V}", m.WriteMetric)
	})
	r.Route("/value", func(r chi.Router) {
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

	log.Fatal(http.ListenAndServe(envAddr, MetricRouter()))

}
