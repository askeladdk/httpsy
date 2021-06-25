package httpsy

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"net/http"
)

// CSRF is a middleware that protects against Cross-Site Request Forgery and BREACH attacks
// by implementing the CSRF-HMAC algorithm.
//
// The middleware generates a signed token and stores in the X-CSRF-Token header for every response.
// The user-agent must reflect the X-CSRF-Token header when making a request.
// The user-agent may also store the signed token in a POST form,
// which must be specified by setting the FormKey field.
// The middleware then verifies that the token was signed with the secret key,
// and responds with 403 Forbidden if it is not.
// Token verification is skipped if the request method is safe (GET, HEAD, OPTIONS, TRACE)
// or if the URL path is exempted. Endpoints that are exempted therefore must not have side effects.
//
// The middleware also intercepts HTTP to HTTPS man-in-the-middle attacks by
// verifying that the request URL and Referer header have the same origin.
//
// Endpoints can access the signed token by reading the header from the http.ResponseWriter:
//  func(w http.ResponseWriter, r *http.Request) {
//      csrfToken := w.Header().Get("X-CSRF-Token")
//      ...
type CSRF struct {
	// ExemptPaths is a slice of URL paths that are exempt from CSRF validation.
	// The request URL path is matched against each element using path.Match.
	// This field is ignored if ExemptFunc is set.
	ExemptPaths []string `json:"exemptPaths" yaml:"exemptPaths"`

	// ExemptFunc is optional and reports whether the request should be exempt from CSRF validation.
	ExemptFunc func(*http.Request) bool `json:"-"`

	// FormKey is the name of the CSRF form value (optional).
	FormKey string `json:"formKey" yaml:"formKey"`

	// Secret is the secret key used to sign the CSRF token (required).
	Secret string `json:"secret" yaml:"secret"`
}

// Handler returns a middleware handler that applies the CSRF configuration.
func (csrf CSRF) Handler(next http.Handler) http.Handler {
	// sanity check
	if csrf.Secret == "" {
		panic("csrf: no secret")
	}

	secret := []byte(csrf.Secret)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !csrf.exempt(r) {
			// intercept http->https mitm attacks by comparing origin and referer headers with url
			if r.URL.Scheme == "https" {
				source := sourceOrigin(r, r.URL)
				target := targetOrigin(r, r.URL)
				if !sameOrigin(source, target) {
					Error(w, r, StatusForbidden)
					return
				}
			}

			// verify sent token
			token, _ := csrf.extractToken(r)
			if !csrfVerifyToken(secret, token) {
				Error(w, r, StatusForbidden)
				return
			}
		}

		// generate new token and hand it to the client
		w.Header().Set("X-CSRF-Token", csrfCreateToken(secret))

		next.ServeHTTP(w, r)
	})
}

func (csrf CSRF) exempt(r *http.Request) bool {
	if Safe(r) {
		return true
	} else if csrf.ExemptFunc != nil {
		return csrf.ExemptFunc(r)
	}
	return stringsMatch(csrf.ExemptPaths, r.URL.Path)
}

func (csrf CSRF) extractToken(r *http.Request) ([]byte, error) {
	if v := r.Header.Get("X-CSRF-Token"); v != "" {
		return base64.StdEncoding.DecodeString(v)
	} else if v := r.PostFormValue(csrf.FormKey); v != "" {
		return base64.StdEncoding.DecodeString(v)
	} else if r.MultipartForm != nil {
		values := r.MultipartForm.Value[csrf.FormKey]
		if len(values) != 0 {
			return base64.StdEncoding.DecodeString(values[0])
		}
	}
	return nil, nil
}

func csrfCreateToken(secret []byte) string {
	buf := make([]byte, 32)
	randomNoise(buf[:32])
	h := hmac.New(sha256.New, secret)
	h.Write(buf[:32])
	buf = h.Sum(buf)
	return base64.StdEncoding.EncodeToString(buf)
}

func csrfVerifyToken(secret, token []byte) bool {
	if len(token) != 64 {
		return false
	}
	h := hmac.New(sha256.New, secret)
	h.Write(token[:32])
	mac := h.Sum(nil)
	return hmac.Equal(token[32:], mac)
}
