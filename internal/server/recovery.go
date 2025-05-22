package server

import (
	"log"
	"net/http"
	"runtime/debug"
)

// Recovery はハンドラ内で発生した panic をキャッチし、500 応答を返しつつサーバーのクラッシュを防ぎます。
func Recovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				log.Printf("panic recovered: %v\n%s", rec, debug.Stack())
				writeJSONError(w, http.StatusInternalServerError, "internal server error")
			}
		}()
		next.ServeHTTP(w, r)
	})
}
