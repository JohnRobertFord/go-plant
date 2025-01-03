package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/JohnRobertFord/go-plant/internal/storage/metrics"
	"github.com/JohnRobertFord/go-plant/internal/storage/metrics/diskfile"
	"github.com/jackc/pgx/v5"
)

func GetAll(ms metrics.Storage) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		// fmt.Println("GetAll")

		list := ms.SelectAll()
		var out []string
		for _, el := range list {
			out = append(out, el.ID)
		}
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		io.WriteString(w, strings.Join(out, ", "))

	})
}
func Ping(ms metrics.Storage) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {

		// fmt.Println("Ping")
		ctx := context.Background()
		cfg := ms.GetConfig()

		for i := 0; i < 3; i++ {
			conn, err := pgx.Connect(ctx, cfg.DatabaseDsn)
			if err == nil {
				w.WriteHeader(http.StatusOK)
				defer func(context context.Context) {
					err := conn.Close(context)
					if err != nil {
						log.Printf("[ERR][DB] error while closing conntection: %s\n", err)
					}
				}(ctx)
				return
			}

			time.Sleep(time.Duration(i+1) * time.Second)
			fmt.Printf("Connect to DB. Retry: %d\n", i+1)
		}
		log.Printf("[ERR][DB] failed to connect to %s\n", cfg.DatabaseDsn)
		w.WriteHeader(http.StatusInternalServerError)
	})
}
func WriteMetric(ms metrics.Storage) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {

		// fmt.Println("WriteMetric")

		metrictype := strings.Split(req.URL.Path, "/")[2]
		metric := strings.Split(req.URL.Path, "/")[3]
		input := strings.Split(req.URL.Path, "/")[4]

		el := metrics.Element{
			ID:    metric,
			MType: metrictype,
		}

		switch metrictype {
		case "gauge":
			if f, err := strconv.ParseFloat(input, 64); err == nil {
				el.Value = &f
			}
		case "counter":
			if i, err := strconv.ParseInt(input, 10, 64); err == nil {
				el.Delta = &i
			}
		}
		ms.Insert(el)

		w.WriteHeader(http.StatusOK)
	})
}
func WriteJSONMetric(ms metrics.Storage) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {

		// fmt.Println("WriteJSONMetric")

		data, err := io.ReadAll(req.Body)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		defer req.Body.Close()

		cfg := ms.GetConfig()
		if data[0] != '[' {
			var in metrics.Element
			err = json.Unmarshal(data, &in)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			o, _ := json.Marshal(ms.Insert(in))
			if cfg.StoreInterval == 0 {
				err = diskfile.Write2File(ms)
				if err != nil {
					log.Printf("[ERR][FILE] %s", err)
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
				out = append(out, ms.Insert(el))
			}

			o, _ := json.Marshal(out)

			if cfg.StoreInterval == 0 {
				err = diskfile.Write2File(ms)
				if err != nil {
					log.Printf("[ERR][FILE] %s", err)
				}
			}

			w.WriteHeader(http.StatusOK)
			w.Header().Set("Content-Type", "application/json")
			io.WriteString(w, fmt.Sprintf("%s\n", o))
		}
	})
}
func GetJSONMetric(ms metrics.Storage) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {

		// fmt.Println("GetJSONMetric")

		decoder := json.NewDecoder(req.Body)
		var in metrics.Element
		err := decoder.Decode(&in)
		if err != nil {
			if !errors.Is(err, io.EOF) {
				w.WriteHeader(http.StatusBadRequest)
				log.Printf("[ERR][JSON] %s", err)
			}
			return
		}
		defer req.Body.Close()

		w.Header().Set("Content-Type", "application/json")

		res, err := ms.Select(in)
		if err != nil {
			http.Error(w, "Metric Not Found", http.StatusNotFound)
			return
		}

		w.WriteHeader(http.StatusOK)
		o, _ := json.Marshal(res)
		io.WriteString(w, fmt.Sprintf("%s\n", o))
	})
}
func GetMetric(ms metrics.Storage) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {

		// fmt.Println("GetMetric")

		w.Header().Set("Content-Type", "text/plain")
		metrictype := strings.Split(req.URL.Path, "/")[2]
		ID := strings.Split(req.URL.Path, "/")[3]

		res, err := ms.Select(metrics.Element{ID: ID, MType: metrictype})
		if err != nil {
			http.Error(w, "Metric Not Found", http.StatusNotFound)
			return
		}
		var out any
		switch metrictype {
		case "gauge":
			out = *res.Value
		case "counter":
			out = *res.Delta
		}

		w.WriteHeader(http.StatusOK)
		io.WriteString(w, fmt.Sprintf("%v\n", out))
	})
}
