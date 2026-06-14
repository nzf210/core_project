package webhook

import (
	"net/http"

	"core_project/shared/sdk/config"
)

// ValidateN8NSignature checks if the request is signed by N8N
func ValidateN8NSignature(r *http.Request) bool {
	secret := config.GlobalConfig.N8N.WebhookSecret
	if secret == "" {
		return false // Or perhaps handle missing secret configuration
	}

	// Example signature header check.
	// We'll just check X-Webhook-Secret for now as instructed in the spec,
	// though standard N8N implementations may differ.
	headerSecret := r.Header.Get("X-Webhook-Secret")
	if headerSecret == "" {
		return false
	}

	return headerSecret == secret
}

// RequireN8NSecret is a middleware to enforce N8N Webhook authentication
func RequireN8NSecret(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !ValidateN8NSignature(r) {
			http.Error(w, "Unauthorized N8N Webhook Request", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}
