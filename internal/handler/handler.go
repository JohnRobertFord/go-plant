package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/JohnRobertFord/go-plant/internal/storage/metrics"
	"github.com/JohnRobertFord/go-plant/internal/storage/metrics/diskfile"
	"github.com/go-chi/chi"
)

func Ping(ms metrics.Storage) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		err := ms.Ping(req.Context())
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	})
}
func GetAll(ms metrics.Storage) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {

		ctx := req.Context()
		list, err := ms.SelectAll(ctx)
		if err != nil {
			log.Println("[ERR][SELECTALL] failed to get all metrics")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		var out []string
		for _, el := range list {
			out = append(out, el.ID)
		}
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		io.WriteString(w, strings.Join(out, ", "))

	})
}
func WriteMetric(ms metrics.Storage) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {

		ctx := req.Context()
		metrictype := chi.URLParam(req, "MetricType")
		metric := chi.URLParam(req, "MetricID")
		input := chi.URLParam(req, "MetricValue")

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
		_, err := ms.Insert(ctx, el)
		if err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
	})
}
func WriteJSONMetric(ms metrics.Storage) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		ctx := req.Context()
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
			insrt, err := ms.Insert(ctx, in)
			if err != nil {
				log.Println(err)
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			o, _ := json.Marshal(insrt)
			if cfg.StoreInterval == 0 {
				err = diskfile.Write2File(ctx, ms)
				if err != nil {
					log.Printf("[ERR][FILE] %s", err)
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
			}

			w.WriteHeader(http.StatusOK)
			w.Header().Set("Content-Type", "application/json")
			io.WriteString(w, fmt.Sprintf("%s\n", o))
		} else {
			var in []metrics.Element
			err = json.Unmarshal(data, &in)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			var out []metrics.Element
			for _, el := range in {
				insrt, err := ms.Insert(ctx, el)
				if err != nil {
					log.Println(err)
					continue
				}
				out = append(out, insrt)
			}

			o, _ := json.Marshal(out)

			if cfg.StoreInterval == 0 {
				err = diskfile.Write2File(ctx, ms)
				if err != nil {
					log.Printf("[ERR][FILE] %s", err)
					w.WriteHeader(http.StatusInternalServerError)
					return
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

		ctx := req.Context()
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

		res, err := ms.Select(ctx, in)
		if err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusNotFound)
			return
		}

		w.WriteHeader(http.StatusOK)
		o, _ := json.Marshal(res)
		io.WriteString(w, fmt.Sprintf("%s\n", o))
	})
}
func GetMetric(ms metrics.Storage) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {

		ctx := req.Context()
		w.Header().Set("Content-Type", "text/plain")
		metrictype := chi.URLParam(req, "MetricType")
		ID := chi.URLParam(req, "MetricID")

		res, err := ms.Select(ctx, metrics.Element{ID: ID, MType: metrictype})
		if err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusNotFound)
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
