package auth

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"core_project/shared/sdk/config"
	"core_project/shared/sdk/response"
)

type contextKey string

const TenantIDKey contextKey = "tenantID"
const UserIDKey contextKey = "userID"
const RoleKey contextKey = "role"

// ValidateJWT verifies the JWT token and returns claims if valid
func ValidateJWT(tokenString string) (jwt.MapClaims, error) {
	if config.GlobalConfig == nil {
		return nil, errors.New("config not loaded")
	}

	secret := []byte(config.GlobalConfig.JWTSecret)
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return secret, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token")
}

// Middleware creates an HTTP middleware for JWT validation and tenant isolation
func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		var tokenString string
		
		if authHeader != "" {
			parts := strings.Split(authHeader, " ")
			if len(parts) == 2 && parts[0] == "Bearer" {
				tokenString = parts[1]
			}
		}

		if tokenString == "" {
			tokenString = r.URL.Query().Get("token")
			// If token came from query string, set it as a cookie for subsequent iframe asset requests
			if tokenString != "" {
				http.SetCookie(w, &http.Cookie{
					Name:     "access_token",
					Value:    tokenString,
					Path:     "/",
					MaxAge:   3600,
					HttpOnly: true,
					SameSite: http.SameSiteLaxMode,
				})
			}
		}

		if tokenString == "" {
			if cookie, err := r.Cookie("access_token"); err == nil {
				tokenString = cookie.Value
			}
		}

		if tokenString == "" {
			// Allow n8n static assets to bypass auth since they don't contain sensitive data
			// and might fail to send cookies in cross-origin iframes
			if strings.HasPrefix(r.URL.Path, "/api/superadmin/n8n/assets/") {
				next.ServeHTTP(w, r)
				return
			}
			response.Error(w, http.StatusUnauthorized, "Missing Authorization header or token query", nil)
			return
		}

		claims, err := ValidateJWT(tokenString)
		if err != nil {
			response.Error(w, http.StatusUnauthorized, "Invalid or expired token", err)
			return
		}

		// Extract tenant and user ID (support both snake_case and camelCase)
		tenantID, _ := claims["tenant_id"].(string)
		if tenantID == "" {
			tenantID, _ = claims["tenantId"].(string)
		}
		
		userID, _ := claims["user_id"].(string)
		if userID == "" {
			userID, _ = claims["userId"].(string)
		}

		role, _ := claims["role"].(string)
		if role != "" {
			r.Header.Set("X-User-Role", role)
		}

		if tenantID == "" {
			response.Error(w, http.StatusForbidden, "Token does not contain tenant context", nil)
			return
		}

		// Pass into context
		ctx := context.WithValue(r.Context(), TenantIDKey, tenantID)
		ctx = context.WithValue(ctx, UserIDKey, userID)
		ctx = context.WithValue(ctx, RoleKey, role)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
