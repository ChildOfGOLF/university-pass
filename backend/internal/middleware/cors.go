package middleware

import (
	"net/http"
	"os"
	"strings"
)

type CORS struct {
	allowedOrigins map[string]bool
}

func NewCORS() *CORS {
	origins := map[string]bool{
		"http://localhost":      true,
		"http://localhost:5500": true,
		"http://127.0.0.1:5500": true,
	}

	if raw := os.Getenv("ALLOWED_ORIGINS"); raw != "" {
		for _, o := range strings.Split(raw, ",") {
			o = strings.TrimSpace(o)
			if o != "" {
				origins[o] = true
			}
		}
	}

	return &CORS{allowedOrigins: origins}
}

func (c *CORS) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")

		w.Header().Add("Vary", "Origin")

		if c.allowedOrigins[origin] {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, PUT, DELETE, OPTIONS")

			if h := r.Header.Get("Access-Control-Request-Headers"); h != "" {
				w.Header().Set("Access-Control-Allow-Headers", h)
			} else {
				w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Scanner-Key")
			}

			w.Header().Set("Access-Control-Max-Age", "86400")
		}

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}
