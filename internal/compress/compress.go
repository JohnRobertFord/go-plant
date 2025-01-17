package compress

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"
)

type compressWriter struct {
	http.ResponseWriter
	Writer io.Writer
}

func (c compressWriter) Write(p []byte) (int, error) {
	c.Header().Set("Content-Type", "text/html")
	return c.Writer.Write(p)
}

func GzipMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {

		if strings.Contains(req.Header.Get("Content-Encoding"), "gzip") {
			zr, err := gzip.NewReader(req.Body)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			defer zr.Close()
			req.Body = zr
		}
		if strings.Contains(req.Header.Get("Accept-Encoding"), "gzip") {
			zw, err := gzip.NewWriterLevel(w, gzip.BestCompression)
			if err != nil {
				io.WriteString(w, err.Error())
				return
			}
			defer zw.Close()
			w.Header().Set("Content-Encoding", "gzip")
			w = compressWriter{w, zw}
		}
		next.ServeHTTP(w, req)
	})
}
