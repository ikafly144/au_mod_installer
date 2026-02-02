package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
)

func TestAuthMiddleware(t *testing.T) {
	secret := "secret"
	mw := NewAuthMiddleware(secret)

	// Create a valid token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": 1,
		"exp": time.Now().Add(time.Hour).Unix(),
	})
	tokenString, _ := token.SignedString([]byte(secret))

	tests := []struct {
		name           string
		header         string
		expectedStatus int
	}{
		{
			name:           "Valid Token",
			header:         "Bearer " + tokenString,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Missing Header",
			header:         "",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "Invalid Format",
			header:         "Token " + tokenString,
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "Invalid Token",
			header:         "Bearer invalid",
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			if tt.header != "" {
				req.Header.Set("Authorization", tt.header)
			}
			w := httptest.NewRecorder()

			handler := mw.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			handler.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}
