package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/JohnRobertFord/go-plant/internal/metrics"
	"go.uber.org/zap"
)

var sugar zap.SugaredLogger

type (
	gauge float64
	// counter    int64
	MemStorage struct {
		mapa map[string]any
		mu   sync.Mutex
	}
	responseData struct {
		size   int
		status int
	}

	loggingResponseWriter struct {
		http.ResponseWriter
		responseData *responseData
	}
)

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
			if c, ok := m.mapa[metric].(int64); ok {
				c += i
				m.mapa[metric] = c
			} else {
				m.mapa[metric] = i
			}
		}
	}
	w.WriteHeader(http.StatusOK)
}

func (m *MemStorage) WriteJSONMetrics(w http.ResponseWriter, req *http.Request) {
	m.mu.Lock()
	defer m.mu.Unlock()

	decoder := json.NewDecoder(req.Body)
	var in []metrics.Element
	err := decoder.Decode(&in)
	if err != nil {
		if !errors.Is(err, io.EOF) {
			fmt.Println(err)
		}
	}
	defer req.Body.Close()

	var out []metrics.Element
	for _, el := range in {
		temp := metrics.Element{
			ID:    el.ID,
			MType: el.MType,
		}
		if el.MType == "gauge" {
			m.mapa[el.ID] = *el.Value
			temp.Value = el.Value
		} else if el.MType == "counter" {
			if c, ok := m.mapa[el.ID].(int64); ok {
				c += *el.Delta
				m.mapa[el.ID] = c
				temp.Delta = &c
			} else {
				m.mapa[el.ID] = *el.Delta
				temp.Delta = el.Delta
			}
		} else {
			continue
		}
		out = append(out, temp)
	}

	w.WriteHeader(http.StatusOK)
	o, _ := json.Marshal(out)
	w.Header().Set("Content-Type", "application/json")
	io.WriteString(w, fmt.Sprintf("%s\n", o))

}

func (m *MemStorage) GetMetric(w http.ResponseWriter, req *http.Request) {
	m.mu.Lock()
	defer m.mu.Unlock()

	w.Header().Set("Content-Type", "text/plain")
	ID := strings.Split(req.URL.Path, "/")[3]
	res, ok := m.mapa[ID]
	if !ok {
		http.Error(w, "Metric Not Found", http.StatusNotFound)
		return
	}

	io.WriteString(w, fmt.Sprintf("%v\n", res))
}

func (m *MemStorage) GetJSONMetric(w http.ResponseWriter, req *http.Request) {
	m.mu.Lock()
	defer m.mu.Unlock()

	decoder := json.NewDecoder(req.Body)
	var in []metrics.Element
	err := decoder.Decode(&in)
	if err != nil {
		if !errors.Is(err, io.EOF) {
			fmt.Println(err)
		}
		return
	}
	defer req.Body.Close()
	var out []metrics.Element
	for _, el := range in {

		temp := metrics.Element{
			ID:    el.ID,
			MType: el.MType,
		}
		if el.MType == "gauge" {
			if f, ok := m.mapa[el.ID].(float64); ok {
				temp.Value = &f
			}
		} else if el.MType == "counter" {
			if c, ok := m.mapa[el.ID].(int64); ok {
				temp.Delta = &c
			}
		}
		out = append(out, temp)
	}

	w.WriteHeader(http.StatusOK)
	o, _ := json.Marshal(out)
	w.Header().Set("Content-Type", "application/json")
	io.WriteString(w, fmt.Sprintf("%s\n", o))
}

func (m *MemStorage) GetAll(w http.ResponseWriter, req *http.Request) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var list []string
	for k := range m.mapa {
		list = append(list, k)
	}

	io.WriteString(w, strings.Join(list, ", "))
	w.WriteHeader(http.StatusOK)
}

func (l *loggingResponseWriter) Write(b []byte) (int, error) {
	size, err := l.ResponseWriter.Write(b)
	l.responseData.size += size

	return size, err
}

func (l *loggingResponseWriter) WriteHeader(statusCode int) {
	l.ResponseWriter.WriteHeader(statusCode)
	l.responseData.status = statusCode
}

func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {

		path := strings.Split(req.URL.Path, "/")
		if req.Method == http.MethodPost && path[1] == "update" && len(path) == 3 {
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

func Logging(h http.Handler) http.Handler {
	logFn := func(w http.ResponseWriter, req *http.Request) {

		logger, err := zap.NewDevelopment()
		if err != nil {
			panic(err)
		}
		defer logger.Sync()

		sugar = *logger.Sugar()

		rd := &responseData{
			status: 200,
			size:   0,
		}

		lw := loggingResponseWriter{
			ResponseWriter: w,
			responseData:   rd,
		}

		start := time.Now()
		h.ServeHTTP(&lw, req)
		duration := time.Since(start)

		sugar.Infoln(
			"Method", req.Method,
			"URI", req.RequestURI,
			"Status", rd.status,
			"Size", rd.size,
			"Duration", duration,
		)
	}
	return http.HandlerFunc(logFn)
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
