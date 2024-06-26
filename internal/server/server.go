package server

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
)

type gauge float64
type counter int64
type MemStorage struct {
	mapa map[string]any
	mu   sync.Mutex
}

func NewMemStorage() *MemStorage {
	return &MemStorage{
		mapa: make(map[string]any),
	}
}

func (m *MemStorage) WriteMetric(w http.ResponseWriter, req *http.Request) {
	m.mu.Lock()
	defer m.mu.Unlock()

	input := strings.Split(req.URL.Path, "/")[4]
	metric := strings.Split(req.URL.Path, "/")[3]
	metrictype := strings.Split(req.URL.Path, "/")[2]

	switch metrictype {
	case "gauge":
		if f, err := strconv.ParseFloat(input, 64); err == nil {
			m.mapa[metric] = gauge(f)
		}
	case "counter":
		if i, err := strconv.ParseInt(input, 10, 64); err == nil {
			if c, ok := m.mapa[metric].(counter); ok {
				c += counter(i)
				m.mapa[metric] = c
			} else {
				m.mapa[metric] = counter(i)
			}
		}
	}
	w.WriteHeader(http.StatusOK)
}

func (m *MemStorage) GetMetric(w http.ResponseWriter, req *http.Request) {
	m.mu.Lock()
	defer m.mu.Unlock()

	metric := strings.Split(req.URL.Path, "/")[3]
	res, ok := m.mapa[metric]
	if !ok {
		http.Error(w, "Metric Not Found", http.StatusNotFound)
		return
	}

	io.WriteString(w, fmt.Sprintf("%v", res))
}

func (m *MemStorage) GetAll(w http.ResponseWriter, req *http.Request) {
	m.mu.Lock()
	defer m.mu.Unlock()
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
		// fmt.Println(time.Now())
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
