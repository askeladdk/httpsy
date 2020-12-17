package httpsy

import (
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"strings"
)

// ErrBadCSRF is returned when CSRF validation failed.
const ErrBadCSRF = errorString("csrf validation failed")

const csrfTokenLength = 32

var csrfTokenKey = NewContextKey("csrf-token")

// CSRFToken represents a CSRF token to be included in a response.
type CSRFToken struct {
	value     string
	fieldName string
}

// GetCSRFToken gets the CSRF token from the request if there is one.
func GetCSRFToken(r *http.Request) (CSRFToken, bool) {
	t, ok := GetContextValue(r, csrfTokenKey).(CSRFToken)
	return t, ok
}

// String implements fmt.Stringer.
func (t CSRFToken) String() string {
	return t.value
}

// FormField returns an HTML input element that can be included in a form element in a template.
func (t CSRFToken) FormField() template.HTML {
	fragment := fmt.Sprintf(`<input type="hidden" name="%s" value="%s">`, t.fieldName, t.value)
	return template.HTML(fragment)
}

// CSRF is a middleware that protects against Cross-Site Request Forgery and BREACH attacks.
// It implements the double submit cookie algorithm.
//
// The middleware generates and stores a token in a cookie as well as in the X-CSRF-Token header.
// The user agent must send the cookie and header with every request.
// The middleware generates a 403 Forbidden if the cookie and header do not match
// unless the request method is idempotent (GET, HEAD, OPTIONS, TRACE)
// or the URL path is exempted.
// Instead of using the X-CSRF-Token header, the endpoint can also store
// the token in an HTML form element using GetCSRFToken.
//
// A typical configuration might look like:
//  httpsy.CSRF {
//      CookieName: "__Host-csrf",
//      FieldName: "csrf",
//      Path: "/",
//      Secure: true,
//      HTTPOnly: true,
//      SameSite: http.SameSiteStrictMode,
//  }
type CSRF struct {
	// Secret is an optional secret key used to encrypt the CSRF token.
	Secret [csrfTokenLength]byte

	// ExemptPaths is a slice of URL paths that are exempt from CSRF validation.
	// The request URL path is matched against each element using path.Match.
	// This field is ignored if ExemptFunc is set.
	ExemptPaths []string

	// ExemptFunc is optional and reports whether the request should be exempt from CSRF validation.
	ExemptFunc PredicateFunc

	// CookieName is the name of the CSRF cookie (required).
	CookieName string

	// FieldName is the name of the CSRF form value (required).
	FieldName string

	// Path is the cookie Path attribute.
	Path string

	// Domain is the cookie Domain attribute.
	Domain string

	// Secure is the cookie Secure attribute.
	Secure bool

	// HTTPOnly is the cookie HttpOnly attribute.
	HTTPOnly bool

	// SameSite is the cookie SameSite attribute.
	SameSite http.SameSite
}

// Handler returns a middleware handler that applies the CSRF configuration.
func (csrf CSRF) Handler(next http.Handler) http.Handler {
	// sanity checks
	if csrf.CookieName == "" {
		panic("csrf: no cookie name")
	} else if csrf.FieldName == "" {
		panic("csrf: no field name")
	} else if strings.HasPrefix(csrf.CookieName, "__Host-") &&
		(!csrf.Secure || csrf.Domain != "" || csrf.Path != "/") {
		panic("csrf: __Host- prefixed cookie has incorrect attributes")
	} else if strings.HasPrefix(csrf.CookieName, "__Secure-") && !csrf.Secure {
		panic("csrf: __Secure- prefixed cookie does not have secure flag set")
	}

	secret := csrf.Secret[:]

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var (
			b64       = base64.URLEncoding
			rbuf      [2 * csrfTokenLength]byte
			realToken string
		)

		w.Header().Add("Vary", "Cookie")

		if cookie, err := r.Cookie(csrf.CookieName); err == nil {
			realToken = cookie.Value
		}

		cookieToken := rbuf[csrfTokenLength:]

		// decode the cookie token and regenerate it if it is invalid
		if n, _ := b64.Decode(cookieToken, []byte(realToken)); n != csrfTokenLength {
			randomNoise(cookieToken)
			realToken = b64.EncodeToString(cookieToken)

			http.SetCookie(w, &http.Cookie{
				Name:     csrf.CookieName,
				Value:    realToken,
				Path:     csrf.Path,
				Domain:   csrf.Domain,
				MaxAge:   365 * 24 * 60 * 60,
				Secure:   csrf.Secure,
				HttpOnly: csrf.HTTPOnly,
				SameSite: csrf.SameSite,
			})
		}

		// mask the real token and store it in the request context
		csrfMask(secret, rbuf[:])
		maskedRealToken := b64.EncodeToString(rbuf[:])
		csrfToken := CSRFToken{maskedRealToken, csrf.FieldName}
		r = SetContextValue(r, csrfTokenKey, csrfToken)

		exempt := csrf.exempt(r)

		// intercept http->https mitm attacks by comparing referer with url
		if !exempt && r.URL.Scheme == "https" {
			referer, err := url.Parse(r.Header.Get("Referer"))
			if err != nil || referer.String() == "" || !sameOrigin(referer, r.URL) {
				Forbidden(w, r)
				return
			}
		}

		sentToken := csrf.extractToken(r)
		if !exempt && !csrfVerify(secret, realToken, sentToken) {
			Error(w, r, http.StatusForbidden, ErrBadCSRF)
			return
		}

		w.Header().Set("X-CSRF-Token", maskedRealToken)
		next.ServeHTTP(w, r)
	})
}

func (csrf CSRF) exempt(r *http.Request) bool {
	if Safe(r) {
		return true
	} else if csrf.ExemptFunc != nil {
		return csrf.ExemptFunc.Predicate(r)
	}
	return stringsMatch(csrf.ExemptPaths, r.URL.Path)
}

func (csrf CSRF) extractToken(r *http.Request) string {
	if v := r.Header.Get("X-CSRF-Token"); v != "" {
		return v
	} else if v := r.PostFormValue(csrf.FieldName); v != "" {
		return v
	} else if r.MultipartForm != nil {
		values := r.MultipartForm.Value[csrf.FieldName]
		if len(values) != 0 {
			return values[0]
		}
	}
	return ""
}

func oneTimePad(pad, key []byte) {
	for i := range key {
		pad[i] ^= key[i]
	}
}

func csrfMask(secret, unmasked []byte) {
	nonce := unmasked[:csrfTokenLength]
	token := unmasked[csrfTokenLength:]
	randomNoise(nonce)
	oneTimePad(token, nonce)
	oneTimePad(token, secret)
}

func csrfUnmask(secret, masked []byte) {
	nonce := masked[:csrfTokenLength]
	token := masked[csrfTokenLength:]
	oneTimePad(token, secret)
	oneTimePad(token, nonce)
}

func csrfVerify(secret []byte, unmaskedRealToken, maskedSentToken string) bool {
	var (
		b64  = base64.URLEncoding
		rbuf [csrfTokenLength]byte
		sbuf [2 * csrfTokenLength]byte
	)
	if n, _ := b64.Decode(rbuf[:], []byte(unmaskedRealToken)); n != len(rbuf) {
		return false
	} else if n, _ := b64.Decode(sbuf[:], []byte(maskedSentToken)); n != len(sbuf) {
		return false
	}
	csrfUnmask(secret, sbuf[:])
	return subtle.ConstantTimeCompare(rbuf[:], sbuf[csrfTokenLength:]) == 1
}
