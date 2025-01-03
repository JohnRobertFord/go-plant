package cache

import (
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/JohnRobertFord/go-plant/internal/config"
	"github.com/JohnRobertFord/go-plant/internal/storage/metrics"
)

type (
	// gauge   float64
	// counter int64

	MemStorage struct {
		mapa map[string]any
		mu   sync.Mutex
		cfg  *config.Config
	}
)

func NewMemStorage(c *config.Config) *MemStorage {
	return &MemStorage{
		mapa: make(map[string]any),
		cfg:  c,
	}
}

func (m *MemStorage) Insert(el metrics.Element) metrics.Element {
	m.mu.Lock()
	defer m.mu.Unlock()

	out := metrics.Element{
		ID:    el.ID,
		MType: el.MType,
	}
	switch el.MType {
	case "gauge":
		m.mapa[el.ID] = *el.Value
		out.Value = el.Value
	case "counter":
		if c, ok := m.mapa[el.ID].(int64); ok {
			c += *el.Delta
			m.mapa[el.ID] = c
			out.Delta = &c
		} else {
			m.mapa[el.ID] = *el.Delta
			out.Delta = el.Delta
		}
	}
	return out
}

func (m *MemStorage) Select(el metrics.Element) (metrics.Element, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	out := metrics.Element{
		ID:    el.ID,
		MType: el.MType,
	}

	if el.MType == "gauge" {
		if f, ok := m.mapa[el.ID].(float64); ok {
			out.Value = &f
		} else {
			return metrics.Element{}, fmt.Errorf("metric not found")
		}
	} else if el.MType == "counter" {
		if c, ok := m.mapa[el.ID].(int64); ok {
			out.Delta = &c
		} else {
			return metrics.Element{}, fmt.Errorf("metric not found")
		}
	}

	return out, nil
}

func (m *MemStorage) SelectAll() []metrics.Element {
	m.mu.Lock()
	defer m.mu.Unlock()

	var list []metrics.Element

	for k, v := range m.mapa {
		temp := metrics.Element{
			ID: k,
		}
		switch vt := v.(type) {
		case int64:
			if c, ok := v.(int64); ok {
				temp.MType = "counter"
				temp.Delta = &c
			} else {
				fmt.Println("error counter delta")
				// return metrics.Element{}, fmt.Errorf("metric not found")
			}
		case float64:
			if f, ok := v.(float64); ok {
				temp.MType = "gauge"
				temp.Value = &f
			} else {
				fmt.Println("error gauge value")
				// return metrics.Element{}, fmt.Errorf("metric not found")
			}
		default:
			fmt.Printf("unknown type: %v", vt)
		}
		list = append(list, temp)
	}
	return list
}

func (m *MemStorage) GetConfig() config.Config {
	return *m.cfg
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
