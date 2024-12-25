package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/JohnRobertFord/go-plant/internal/config"
	"github.com/JohnRobertFord/go-plant/internal/metrics"
)

type (
	gauge float64
	// counter    int64

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

func (m *MemStorage) Read4File(filename string) {
	log.Printf("Restore from: %s", filename)
	dataFile, err := os.Open(filename)
	if err != nil {
		fmt.Printf("err.Error()")
	}
	var in []metrics.Element
	jsonParser := json.NewDecoder(dataFile)
	if err = jsonParser.Decode(&in); err != nil {
		fmt.Println("Wrong format metric data")
	}

	for _, el := range in {
		if el.MType == "gauge" && el.Value != nil {
			m.mapa[el.ID] = *el.Value
		} else if el.MType == "counter" && el.Delta != nil {
			m.mapa[el.ID] = *el.Delta
		} else {
			log.Printf("error read \"%s\" metric", el.ID)
			continue
		}
	}
}

func Write2File(m *MemStorage) error {

	file, err := os.OpenFile(m.cfg.FilePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		return err
	}
	defer file.Close()

	var buf []string
	for k, v := range m.mapa {
		switch v.(type) {
		case int64:
			buf = append(buf, fmt.Sprintf("{\"id\":\"%s\",\"type\":\"counter\",\"delta\":%v}", k, v))
		case float64:
			buf = append(buf, fmt.Sprintf("{\"id\":\"%s\",\"type\":\"gauge\",\"value\":%v}", k, v))
		default:
			log.Printf("unknown type %s\n", v)
		}
	}
	_, err = fmt.Fprintf(file, "[%s]", strings.Join(buf, ","))
	return err
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

func (m *MemStorage) WriteJSONMetric(w http.ResponseWriter, req *http.Request) {
	m.mu.Lock()
	defer m.mu.Unlock()

	data, err := io.ReadAll(req.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	defer req.Body.Close()

	if data[0] != '[' {
		var in metrics.Element
		err = json.Unmarshal(data, &in)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		out := metrics.Element{
			ID:    in.ID,
			MType: in.MType,
		}
		if in.MType == "gauge" {
			m.mapa[in.ID] = *in.Value
			out.Value = in.Value
		} else if in.MType == "counter" {
			if c, ok := m.mapa[in.ID].(int64); ok {
				c += *in.Delta
				m.mapa[in.ID] = c
				out.Delta = &c
			} else {
				m.mapa[in.ID] = *in.Delta
				out.Delta = in.Delta
			}
		}

		o, _ := json.Marshal(out)

		if m.cfg.StoreInterval == 0 {
			err = Write2File(m)
			if err != nil {
				fmt.Println(err)
			}
		}

		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, fmt.Sprintf("%s\n", o))
	} else if data[0] == '[' {
		var in []metrics.Element
		err = json.Unmarshal(data, &in)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
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

		o, _ := json.Marshal(out)

		if m.cfg.StoreInterval == 0 {
			err = Write2File(m)
			if err != nil {
				log.Println(err)
			}
		}

		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, fmt.Sprintf("%s\n", o))
	}
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
	var in metrics.Element
	err := decoder.Decode(&in)
	if err != nil {
		if !errors.Is(err, io.EOF) {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Println(err)
		}
		return
	}
	defer req.Body.Close()

	w.Header().Set("Content-Type", "application/json")

	out := metrics.Element{
		ID:    in.ID,
		MType: in.MType,
	}

	if in.MType == "gauge" {
		if f, ok := m.mapa[in.ID].(float64); ok {
			out.Value = &f
		} else {
			http.Error(w, "Metric Not Found", http.StatusNotFound)
			return
		}
	} else if in.MType == "counter" {
		if c, ok := m.mapa[in.ID].(int64); ok {
			out.Delta = &c
		} else {
			http.Error(w, "Metric Not Found", http.StatusNotFound)
			return
		}
	}

	w.WriteHeader(http.StatusOK)
	o, _ := json.Marshal(out)
	io.WriteString(w, fmt.Sprintf("%s\n", o))
}

func (m *MemStorage) GetAll(w http.ResponseWriter, req *http.Request) {
	m.mu.Lock()
	defer m.mu.Unlock()
	w.Header().Set("Content-Type", "text/html")

	var list []string
	for k := range m.mapa {
		list = append(list, k)
	}

	w.WriteHeader(http.StatusOK)
	io.WriteString(w, strings.Join(list, ", "))
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
