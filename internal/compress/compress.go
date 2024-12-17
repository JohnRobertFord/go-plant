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

// func (c compressWriter) WriteHeader(statusCode int) {
// 	c.w.Header().Set("Content-Type", "text/html")
// 	c.w.WriteHeader(statusCode)
// }

// func (c compressWriter) Header() http.Header {
// 	return c.w.Header()
// }

func GzipMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {

		if !strings.Contains(req.Header.Get("Accept-Encoding"), "gzip") {
			next.ServeHTTP(w, req)
			return
		}

		gz, err := gzip.NewWriterLevel(w, gzip.BestCompression)
		if err != nil {
			io.WriteString(w, err.Error())
			return
		}
		defer gz.Close()

		w.Header().Set("Content-Encoding", "gzip")
		next.ServeHTTP(compressWriter{w, gz}, req)

		// проверяем, что клиент отправил серверу сжатые данные в формате gzip
		// contentEncoding := req.Header.Get("Content-Encoding")
		// sendsGzip := strings.Contains(contentEncoding, "gzip")
		// if sendsGzip {
		// 	// оборачиваем тело запроса в io.Reader с поддержкой декомпрессии
		// 	cr, err := newCompressReader(req.Body)
		// 	if err != nil {
		// 		w.WriteHeader(http.StatusInternalServerError)
		// 		return
		// 	}
		// 	// меняем тело запроса на новое
		// 	req.Body = cr
		// 	defer cr.Close()
		// }

		// // передаём управление хендлеру
		// next.ServeHTTP(ow, req)
	})
}
