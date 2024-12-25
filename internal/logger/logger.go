package logger

import (
	"net/http"
	"time"

	"go.uber.org/zap"
)

var sugar zap.SugaredLogger

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

func Logging(h http.Handler) http.Handler {
	logFn := func(w http.ResponseWriter, req *http.Request) {

		logger, err := zap.NewDevelopment()
		if err != nil {
			panic(err)
		}
		defer logger.Sync()

		sugar = *logger.Sugar()

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
