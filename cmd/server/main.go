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
	g    gauge
	c    counter
}

func NewMemStorage() (h *MemStorage) {
	return &MemStorage{
		g:    0.0,
		c:    0,
		mapa: make(map[string]any),
	}
}

func (m *MemStorage) Gauge(w http.ResponseWriter, req *http.Request) {
	input := strings.Split(req.URL.Path, "/")[4]
	metric := strings.Split(req.URL.Path, "/")[3]

	if f64, err := strconv.ParseFloat(input, 64); err == nil {
		m.g = gauge(f64)
	}
	m.mapa[metric] = m.g

	fmt.Printf("Gauge: %v\n", m.mapa[metric])
	w.WriteHeader(http.StatusOK)
}
func (m *MemStorage) Counter(w http.ResponseWriter, req *http.Request) {
	input := strings.Split(req.URL.Path, "/")[4]
	metric := strings.Split(req.URL.Path, "/")[3]

	if i64, err := strconv.ParseInt(input, 10, 64); err == nil {
		m.c += counter(i64)
	}
	m.mapa[metric] = m.c
	fmt.Printf("Counter: %v\n", m.mapa[metric])
	w.WriteHeader(http.StatusOK)
}

func middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodPost {
			http.Error(w, "Only POST requests are allowed!", http.StatusMethodNotAllowed)
			return
		}

		req.Header.Set("Accept", "*/*")
		// if req.Header.Get("Content-type") != "text/plain" {
		// 	http.Error(w, "Use text/plain data", http.StatusNotAcceptable)
		// 	return
		// }
		path := strings.Split(req.URL.Path, "/")
		if len(path) != 5 {
			http.Error(w, "Incorrect input!", http.StatusNotFound)
			return
		}
		val := path[4]

		if strings.Compare(path[2], "counter") != 0 && !IsCounter(val) {
			if strings.Compare(path[2], "gauge") != 0 && !IsGauge(val) {
				http.Error(w, "Bad Request!", http.StatusBadRequest)
				return
			}
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

	// var m1 MemStorage
	// ml := make(map[string]any)
	m := NewMemStorage()
	mux := http.NewServeMux()

	// mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
	// 	w.Write([]byte("PLANT"))
	// 	w.WriteHeader(http.StatusOK)
	// })
	mux.Handle("/update/gauge/", middleware(http.HandlerFunc(m.Gauge)))
	mux.Handle("/update/counter/", middleware(http.HandlerFunc(m.Counter)))

	if err := http.ListenAndServe(":8080", mux); err != nil {
		panic(err)
	}
}
