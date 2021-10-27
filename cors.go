package httpsy

import (
	"net/http"
	"net/textproto"
	"strconv"
	"strings"
)

// CORS is a middleware for Cross-Origin Resource Sharing.
// The middleware sets the appropriate HTTP headers and handles CORS preflight requests.
// It does not enforce CORS rules -- That is up to the user agent (browser).
//
// CORS is not a security mechanism. In fact, it reduces security in a controlled manner.
// Browsers normally only allow scripts to access resources that exist in
// the same domain (Same Origin Policy). CORS instructs the browser under what conditions
// to relax this policy and allow scripts to access resources across domains.
//
// A typical configuration might look like this:
//  httpsy.CORS{
// 	   AllowHeaders: []string{"X-Requested-With", "Content-Type", "Authorization"},
// 	   AllowMethods: []string{"GET", "PUT", "POST", "DELETE", "HEAD", "OPTIONS"},
// 	   AllowOrigins: []string{"https://example.com", "https://api.example.com"},
// 	   MaxAge:       10 * time.Minute,
//  }
type CORS struct {
	// AllowCredentials sets the Access-Control-Allow-Credentials header.
	// The header will not be set if Access-Control-Allow-Origin is "*",
	// as that is disallowed by the standard for security reasons.
	AllowCredentials bool `json:"allowCredentials" yaml:"allowCredentials"`

	// AllowHeaders sets the Access-Control-Allow-Headers header.
	// An empty slice reflects the Access-Control-Request-Headers header.
	AllowHeaders []string `json:"allowHeaders,omitempty" yaml:"allowHeaders,omitempty"`

	// AllowMethods sets the Access-Control-Allow-Methods header.
	// An empty slice reflects the Access-Control-Request-Method header.
	AllowMethods []string `json:"allowMethods,omitempty" yaml:"allowMethods,omitempty"`

	// AllowOrigins lists all origins that the user agent is allowed to fetch from.
	// The request Origin header is matched against each element using path.Match.
	// The Access-Control-Allow-Origin header is set to Origin if a match is found.
	// No CORS headers will be set if no match was found.
	// The ACAO header is set to "*" if the slice is empty (not recommended).
	// This field is ignored if AllowOriginFunc is set.
	AllowOrigins []string `json:"allowOrigins,omitempty" yaml:"allowOrigins,omitempty"`

	// AllowOriginFunc overrides the behaviour for origin matching.
	// It must return the value of Access-Control-Allow-Origin and whether there was a match.
	AllowOriginFunc func(r *http.Request) (origin string, ok bool) `json:"-" yaml:"-"`

	// ExposeHeaders sets the Access-Control-Expose-Headers header.
	ExposeHeaders []string `json:"exposeHeaders,omitempty" yaml:"exposeHeaders,omitempty"`

	// MaxAge (seconds) sets the Access-Control-Max-Age header.
	// It defaults to -1 if not set.
	MaxAge int `json:"maxAge" yaml:"maxAge"`

	// OptionsPassthrough specifies that the handler should continue to the next one
	// after the preflight CORS rules have been applied.
	OptionsPassthrough bool `json:"optionsPassthrough" yaml:"optionsPassthrough"`
}

// Handle returns a middleware handler that applies the CORS configuration.
func (cors *CORS) Handle(next http.Handler) http.Handler {
	var (
		allowMethods  = strings.Join(cors.AllowMethods, ", ")
		allowHeaders  = stringsJoinMap(cors.AllowHeaders, ", ", textproto.CanonicalMIMEHeaderKey)
		exposeHeaders = stringsJoinMap(cors.ExposeHeaders, ", ", textproto.CanonicalMIMEHeaderKey)
		maxAge        = "-1"
		allowOrigins  []string
	)

	if cors.MaxAge > 0 {
		maxAge = strconv.Itoa(int(cors.MaxAge))
	}

	for _, s := range cors.AllowOrigins {
		allowOrigins = append(allowOrigins, strings.ToLower(s))
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var (
			h           = w.Header()
			origin      = r.Header.Get("Origin")
			isCORS      = origin != ""
			isPreflight = r.Method == http.MethodOptions &&
				r.Header.Get("Access-Control-Request-Method") != ""
		)

		if isCORS {
			h.Add("Vary", "Origin")

			if cors.AllowOriginFunc != nil {
				origin, isCORS = cors.AllowOriginFunc(r)
			} else if len(allowOrigins) == 0 {
				origin = "*"
			} else {
				isCORS = stringsMatch(allowOrigins, strings.ToLower(origin))
			}
		}

		if !isCORS {
			next.ServeHTTP(w, r)
			return
		}

		h.Set("Access-Control-Allow-Origin", origin)

		if cors.AllowCredentials && origin != "*" {
			h.Set("Access-Control-Allow-Credentials", "true")
		}

		if !isPreflight && exposeHeaders != "" {
			h.Set("Access-Control-Expose-Headers", exposeHeaders)
		}

		if isPreflight {
			h.Add("Vary", "Access-Control-Request-Headers")
			h.Add("Vary", "Access-Control-Request-Method")

			if allowHeaders != "" {
				h.Set("Access-Control-Allow-Headers", allowHeaders)
			} else {
				h.Set("Access-Control-Allow-Headers", r.Header.Get("Access-Control-Request-Headers"))
			}

			if allowMethods != "" {
				h.Set("Access-Control-Allow-Methods", allowMethods)
			} else {
				h.Set("Access-Control-Allow-Methods", r.Header.Get("Access-Control-Request-Method"))
			}

			h.Set("Access-Control-Max-Age", maxAge)

			if !cors.OptionsPassthrough {
				w.Header().Add("Content-Length", "0")
				w.WriteHeader(http.StatusOK)
				return
			}
		}

		next.ServeHTTP(w, r)
	})
}
