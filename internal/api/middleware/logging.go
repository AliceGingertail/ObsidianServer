package middleware

import (
	"log"
	"net/http"
	"time"
)

type responseWriter struct {
	http.ResponseWriter
	status int
	size   int
}

func (rw *responseWriter) WriteHeader(status int) {
	rw.status = status
	rw.ResponseWriter.WriteHeader(status)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	size, err := rw.ResponseWriter.Write(b)
	rw.size += size
	return size, err
}

func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Оборачиваем ResponseWriter
		rw := &responseWriter{
			ResponseWriter: w,
			status:         http.StatusOK,
		}

		// Вызываем следующий обработчик
		next.ServeHTTP(rw, r)

		// Логируем запрос
		duration := time.Since(start)
		log.Printf(
			"[%s] %s %s %d %d %s",
			time.Now().Format("2006-01-02 15:04:05"),
			r.Method,
			r.URL.Path,
			rw.status,
			rw.size,
			duration,
		)
	})
}
