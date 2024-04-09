package main

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

func init() {
	println("Initialization...")
}

type gauge float64
type counter int64
type MemStorage struct {
	mapa map[string]any
}

func NewMemStorage() (h *MemStorage) {
	return &MemStorage{
		mapa: make(map[string]any),
	}
}

func (m *MemStorage) Metric(w http.ResponseWriter, req *http.Request) {
	input := strings.Split(req.URL.Path, "/")[4]
	metric := strings.Split(req.URL.Path, "/")[3]
	metric_type := strings.Split(req.URL.Path, "/")[2]

	switch metric_type {
	case "gauge":
		if f64, err := strconv.ParseFloat(input, 64); err == nil {
			m.mapa[metric] = gauge(f64)
		}
		fmt.Printf("Storage: %v\n", m.mapa)
	case "counter":
		if i64, err := strconv.ParseInt(input, 10, 64); err == nil {
			if c, ok := m.mapa[metric].(counter); ok {
				c += counter(i64)
				m.mapa[metric] = c
			} else {
				m.mapa[metric] = counter(i64)
			}
		}
		fmt.Printf("Storage: %v\n", m.mapa)
	}

	w.WriteHeader(http.StatusOK)
}

func middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodPost {
			http.Error(w, "Only POST requests are allowed!", http.StatusMethodNotAllowed)
			return
		}

		req.Header.Set("Accept", "*/*")
		path := strings.Split(req.URL.Path, "/")
		if len(path) != 5 {
			http.Error(w, "Incorrect input!", http.StatusNotFound)
			return
		}
		val := path[4]

		if (strings.Compare(path[2], "counter") != 0 || !IsCounter(val)) &&
			(strings.Compare(path[2], "gauge") != 0 || !IsGauge(val)) {
			http.Error(w, "Bad Request!", http.StatusBadRequest)
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

func main() {

	m := NewMemStorage()
	mux := http.NewServeMux()

	mux.Handle("/update/", middleware(http.HandlerFunc(m.Metric)))

	if err := http.ListenAndServe(":8080", mux); err != nil {
		panic(err)
	}
}
