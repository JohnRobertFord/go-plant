package logger

import (
	"net/http"
	"sync"
	"time"

	"go.uber.org/zap"
)

var sugar zap.SugaredLogger
var once sync.Once

type (
	loggingResponseWriter struct {
		http.ResponseWriter
		responseData *responseData
	}
	responseData struct {
		size   int
		status int
	}
)

func (l *loggingResponseWriter) Write(b []byte) (int, error) {
	size, err := l.ResponseWriter.Write(b)
	l.responseData.size += size

	return size, err
}

func (l *loggingResponseWriter) WriteHeader(statusCode int) {
	l.ResponseWriter.WriteHeader(statusCode)
	l.responseData.status = statusCode
}

func getLogger() *zap.SugaredLogger {
	once.Do(func() {
		logger, err := zap.NewProduction()
		if err != nil {
			panic(err)
		}

		sugar = *logger.Sugar()
	})
	return &sugar
}

func Logging(h http.Handler) http.Handler {
	logFn := func(w http.ResponseWriter, req *http.Request) {

		s := getLogger()

		rd := &responseData{
			status: 0,
			size:   0,
		}
		lw := loggingResponseWriter{
			ResponseWriter: w,
			responseData:   rd,
		}

		start := time.Now()
		h.ServeHTTP(&lw, req)
		duration := time.Since(start)

		s.Infoln(
			"Method", req.Method,
			"URI", req.RequestURI,
			"Status", rd.status,
			"Size", rd.size,
			"Duration", duration,
		)
	}
	return http.HandlerFunc(logFn)
}
