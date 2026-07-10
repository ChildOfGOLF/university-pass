package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"university-pass/internal/model"
	"university-pass/internal/repository"
)

type contextKey string

const AccessPointCtxKey contextKey = "access_point"

func RequireScannerKey(apRepo *repository.AccessPointRepository) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := r.Header.Get("X-Scanner-Key")
			if key == "" {
				writeJSONError(w, http.StatusUnauthorized, "missing scanner key")
				return
			}

			ap, err := apRepo.GetByAPIKey(r.Context(), key)
			if err != nil {
				writeJSONError(w, http.StatusInternalServerError, "failed to verify scanner key")
				return
			}
			if ap == nil {
				writeJSONError(w, http.StatusUnauthorized, "invalid scanner key")
				return
			}

			ctx := context.WithValue(r.Context(), AccessPointCtxKey, ap)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func AccessPointFromContext(ctx context.Context) *model.AccessPoint {
	ap, _ := ctx.Value(AccessPointCtxKey).(*model.AccessPoint)
	return ap
}

func writeJSONError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}
