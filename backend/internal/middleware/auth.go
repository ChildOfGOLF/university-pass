package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

type ctxKey string

const (
	CtxUserID ctxKey = "user_id"
	CtxRole   ctxKey = "role"
)

// TODO: move to env
var jwtSecret = []byte("vBS0K4W5DRo2iTQI1JmnuqIouvnHaBbsyvXxqk1Ibhz")

type Claims struct {
	UserID int    `json:"user_id"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

func RequireRole(roles ...string) func(http.Handler) http.Handler {
	allowedRoles := make(map[string]bool, len(roles))
	for _, r := range roles {
		allowedRoles[r] = true
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if !strings.HasPrefix(authHeader, "Bearer ") {
				http.Error(w, `{"error":"missing bearer token"}`, http.StatusUnauthorized)
				return
			}

			tokenStr := strings.TrimPrefix(authHeader, "Bearer ")

			claims := &Claims{}
			token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
				return jwtSecret, nil
			})

			if err != nil || !token.Valid {
				http.Error(w, `{"error":"invalid token"}`, http.StatusUnauthorized)
				return
			}

			if !allowedRoles[claims.Role] {
				http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
				return
			}

			ctx := context.WithValue(r.Context(), CtxUserID, claims.UserID)
			ctx = context.WithValue(ctx, CtxRole, claims.Role)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
