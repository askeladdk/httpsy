package httpsy

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"net/http"
	"time"
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
type CSRF struct {
	// ExemptPaths is a slice of URL paths that are exempt from CSRF validation.
	// The request URL path is matched against each element using path.Match.
	// This field is ignored if ExemptFunc is set.
	ExemptPaths []string `json:"exemptPaths,omitempty" yaml:"exemptPaths,omitempty"`

	// ExemptFunc is optional and reports whether the request should be exempt from CSRF validation.
	ExemptFunc func(*http.Request) bool `json:"-" yaml:"-"`

	// Expires is the duration that a CSRF token is valid (required).
	Expires time.Duration `json:"expires" yaml:"expires"`

	// FormKey is the name of the CSRF form value (optional).
	FormKey string `json:"formKey,omitempty" yaml:"formKey,omitempty"`

	// Secret is the secret key used to sign the CSRF token (required).
	Secret string `json:"secret" yaml:"secret"`

	// SessionFunc extracts the session ID from the request if there is one (required).
	// No token will be generated and validation will fail if there is no session ID.
	SessionFunc func(*http.Request) (sessionID string, ok bool) `json:"-" yaml:"-"`
}

// Handler returns a middleware handler that applies the CSRF configuration.
func (csrf CSRF) Handler(next http.Handler) http.Handler {
	// sanity checks
	if csrf.Secret == "" {
		panic("csrf: no secret")
	} else if csrf.Expires == 0 {
		panic("csrf: no expires")
	} else if csrf.SessionFunc == nil {
		panic("csrf: no session func")
	}

	secret := []byte(csrf.Secret)
	b64 := base64.StdEncoding

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sessionID, session := csrf.SessionFunc(r)

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

			// bail if there is no session id
			if !session {
				Error(w, r, StatusForbidden)
				return
			}

			// verify sent token
			token, _ := b64.DecodeString(csrf.extractToken(r))
			if !csrfVerifyToken(secret, token, sessionID) {
				Error(w, r, StatusForbidden)
				return
			}
		}

		// generate new token and hand it to the client
		if session {
			token := b64.EncodeToString(csrfCreateToken(secret, sessionID, csrf.Expires))
			w.Header().Set("X-CSRF-Token", token)
		}

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

func (csrf CSRF) extractToken(r *http.Request) (token string) {
	if v := r.Header.Get("X-CSRF-Token"); v != "" {
		token = v
	} else if v := r.PostFormValue(csrf.FormKey); v != "" {
		token = v
	} else if r.MultipartForm != nil {
		values := r.MultipartForm.Value[csrf.FormKey]
		if len(values) != 0 {
			token = values[0]
		}
	}
	return
}

func csrfCreateToken(secret []byte, sessionID string, d time.Duration) []byte {
	buf := make([]byte, 16, 48)

	endTime := time.Now().Add(d)
	binary.LittleEndian.PutUint64(buf[:8], uint64(endTime.Unix()))
	binary.LittleEndian.PutUint64(buf[8:16], uint64(endTime.UnixNano()))

	h := hmac.New(sha256.New, secret)
	h.Write(buf)
	h.Write([]byte(sessionID))
	buf = h.Sum(buf)
	return buf
}

func csrfVerifyToken(secret, token []byte, sessionID string) bool {
	if len(token) != 48 {
		return false
	}

	// validate token signature
	h := hmac.New(sha256.New, secret)
	h.Write(token[:16])
	h.Write([]byte(sessionID))
	mac := h.Sum(nil)
	if !hmac.Equal(token[16:], mac) {
		return false
	}

	// check if token expired
	secs := int64(binary.LittleEndian.Uint64(token[:8]))
	nsec := int64(binary.LittleEndian.Uint64(token[8:16]))
	endTime := time.Unix(secs, nsec)
	return time.Now().Before(endTime)
}
