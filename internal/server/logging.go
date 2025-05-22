package server

import (
	"log"
	"net/http"
	"time"
)

// statusRecorder はステータスコードをキャプチャするためのラッパー
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

func newStatusRecorder(w http.ResponseWriter, s int) *statusRecorder {
	return &statusRecorder{ResponseWriter: w, status: s}
}

// RequestLogger は各リクエストのメソッド、URI、ステータス、処理時間をログ出力するミドルウェア
func RequestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := newStatusRecorder(w, http.StatusOK)
		next.ServeHTTP(rec, r)
		duration := time.Since(start)

		log.Printf(
			"%s %s → %d (%s)",
			r.Method,
			r.RequestURI,
			rec.status,
			duration,
		)
	})
}
