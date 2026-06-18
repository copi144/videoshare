package middleware

import (
	"net/http"

	"github.com/gorilla/csrf"
)

// CSRFProtection returns middleware that protects POST/PUT/DELETE routes.
// key is the 32-byte CSRF auth key. secure should be true in production.
func CSRFProtection(key []byte, secure bool) func(http.Handler) http.Handler {
	return csrf.Protect(key,
		csrf.Secure(secure),
		csrf.Path("/"),
		csrf.SameSite(csrf.SameSiteLaxMode),
	)
}
