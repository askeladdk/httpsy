package httpsy

import (
	"fmt"
	"net/http"
)

var statusJSON = func() map[int][]byte {
	m := make(map[int][]byte)
	for _, status := range []int{
		http.StatusContinue,
		http.StatusSwitchingProtocols,
		http.StatusProcessing,
		http.StatusEarlyHints,
		http.StatusOK,
		http.StatusCreated,
		http.StatusAccepted,
		http.StatusNonAuthoritativeInfo,
		http.StatusNoContent,
		http.StatusResetContent,
		http.StatusPartialContent,
		http.StatusMultiStatus,
		http.StatusAlreadyReported,
		http.StatusIMUsed,
		http.StatusMultipleChoices,
		http.StatusMovedPermanently,
		http.StatusFound,
		http.StatusSeeOther,
		http.StatusNotModified,
		http.StatusUseProxy,
		http.StatusTemporaryRedirect,
		http.StatusPermanentRedirect,
		http.StatusBadRequest,
		http.StatusUnauthorized,
		http.StatusPaymentRequired,
		http.StatusForbidden,
		http.StatusNotFound,
		http.StatusMethodNotAllowed,
		http.StatusNotAcceptable,
		http.StatusProxyAuthRequired,
		http.StatusRequestTimeout,
		http.StatusConflict,
		http.StatusGone,
		http.StatusLengthRequired,
		http.StatusPreconditionFailed,
		http.StatusRequestEntityTooLarge,
		http.StatusRequestURITooLong,
		http.StatusUnsupportedMediaType,
		http.StatusRequestedRangeNotSatisfiable,
		http.StatusExpectationFailed,
		http.StatusTeapot,
		http.StatusMisdirectedRequest,
		http.StatusUnprocessableEntity,
		http.StatusLocked,
		http.StatusFailedDependency,
		http.StatusTooEarly,
		http.StatusUpgradeRequired,
		http.StatusPreconditionRequired,
		http.StatusTooManyRequests,
		http.StatusRequestHeaderFieldsTooLarge,
		http.StatusUnavailableForLegalReasons,
		http.StatusInternalServerError,
		http.StatusNotImplemented,
		http.StatusBadGateway,
		http.StatusServiceUnavailable,
		http.StatusGatewayTimeout,
		http.StatusHTTPVersionNotSupported,
		http.StatusVariantAlsoNegotiates,
		http.StatusInsufficientStorage,
		http.StatusLoopDetected,
		http.StatusNotExtended,
		http.StatusNetworkAuthenticationRequired,
	} {
		m[status] = []byte(fmt.Sprintf(`{"status":%d,"title":"%s"}`, status, http.StatusText(status)))
	}
	return m
}()

type httpStatus int

func (status httpStatus) Error() string                { return http.StatusText(int(status)) }
func (status httpStatus) MarshalJSON() ([]byte, error) { return statusJSON[int(status)], nil }
func (status httpStatus) ProblemDetailer() bool        { return true }
func (status httpStatus) StatusCode() int              { return int(status) }

// HTTP status codes as registered with IANA.
// See: https://www.iana.org/assignments/http-status-codes/http-status-codes.xhtml
//
// HTTP status codes are errors and can be passed to the error handler:
//  httpsy.Error(w, r, httpsy.StatusForbidden)
const (
	StatusContinue                      httpStatus = 100 // RFC 7231, 6.2.1
	StatusSwitchingProtocols            httpStatus = 101 // RFC 7231, 6.2.2
	StatusProcessing                    httpStatus = 102 // RFC 2518, 10.1
	StatusEarlyHints                    httpStatus = 103 // RFC 8297
	StatusOK                            httpStatus = 200 // RFC 7231, 6.3.1
	StatusCreated                       httpStatus = 201 // RFC 7231, 6.3.2
	StatusAccepted                      httpStatus = 202 // RFC 7231, 6.3.3
	StatusNonAuthoritativeInfo          httpStatus = 203 // RFC 7231, 6.3.4
	StatusNoContent                     httpStatus = 204 // RFC 7231, 6.3.5
	StatusResetContent                  httpStatus = 205 // RFC 7231, 6.3.6
	StatusPartialContent                httpStatus = 206 // RFC 7233, 4.1
	StatusMultiStatus                   httpStatus = 207 // RFC 4918, 11.1
	StatusAlreadyReported               httpStatus = 208 // RFC 5842, 7.1
	StatusIMUsed                        httpStatus = 226 // RFC 3229, 10.4.1
	StatusMultipleChoices               httpStatus = 300 // RFC 7231, 6.4.1
	StatusMovedPermanently              httpStatus = 301 // RFC 7231, 6.4.2
	StatusFound                         httpStatus = 302 // RFC 7231, 6.4.3
	StatusSeeOther                      httpStatus = 303 // RFC 7231, 6.4.4
	StatusNotModified                   httpStatus = 304 // RFC 7232, 4.1
	StatusUseProxy                      httpStatus = 305 // RFC 7231, 6.4.5
	StatusTemporaryRedirect             httpStatus = 307 // RFC 7231, 6.4.7
	StatusPermanentRedirect             httpStatus = 308 // RFC 7538, 3
	StatusBadRequest                    httpStatus = 400 // RFC 7231, 6.5.1
	StatusUnauthorized                  httpStatus = 401 // RFC 7235, 3.1
	StatusPaymentRequired               httpStatus = 402 // RFC 7231, 6.5.2
	StatusForbidden                     httpStatus = 403 // RFC 7231, 6.5.3
	StatusNotFound                      httpStatus = 404 // RFC 7231, 6.5.4
	StatusMethodNotAllowed              httpStatus = 405 // RFC 7231, 6.5.5
	StatusNotAcceptable                 httpStatus = 406 // RFC 7231, 6.5.6
	StatusProxyAuthRequired             httpStatus = 407 // RFC 7235, 3.2
	StatusRequestTimeout                httpStatus = 408 // RFC 7231, 6.5.7
	StatusConflict                      httpStatus = 409 // RFC 7231, 6.5.8
	StatusGone                          httpStatus = 410 // RFC 7231, 6.5.9
	StatusLengthRequired                httpStatus = 411 // RFC 7231, 6.5.10
	StatusPreconditionFailed            httpStatus = 412 // RFC 7232, 4.2
	StatusRequestEntityTooLarge         httpStatus = 413 // RFC 7231, 6.5.11
	StatusRequestURITooLong             httpStatus = 414 // RFC 7231, 6.5.12
	StatusUnsupportedMediaType          httpStatus = 415 // RFC 7231, 6.5.13
	StatusRequestedRangeNotSatisfiable  httpStatus = 416 // RFC 7233, 4.4
	StatusExpectationFailed             httpStatus = 417 // RFC 7231, 6.5.14
	StatusTeapot                        httpStatus = 418 // RFC 7168, 2.3.3
	StatusMisdirectedRequest            httpStatus = 421 // RFC 7540, 9.1.2
	StatusUnprocessableEntity           httpStatus = 422 // RFC 4918, 11.2
	StatusLocked                        httpStatus = 423 // RFC 4918, 11.3
	StatusFailedDependency              httpStatus = 424 // RFC 4918, 11.4
	StatusTooEarly                      httpStatus = 425 // RFC 8470, 5.2.
	StatusUpgradeRequired               httpStatus = 426 // RFC 7231, 6.5.15
	StatusPreconditionRequired          httpStatus = 428 // RFC 6585, 3
	StatusTooManyRequests               httpStatus = 429 // RFC 6585, 4
	StatusRequestHeaderFieldsTooLarge   httpStatus = 431 // RFC 6585, 5
	StatusUnavailableForLegalReasons    httpStatus = 451 // RFC 7725, 3
	StatusInternalServerError           httpStatus = 500 // RFC 7231, 6.6.1
	StatusNotImplemented                httpStatus = 501 // RFC 7231, 6.6.2
	StatusBadGateway                    httpStatus = 502 // RFC 7231, 6.6.3
	StatusServiceUnavailable            httpStatus = 503 // RFC 7231, 6.6.4
	StatusGatewayTimeout                httpStatus = 504 // RFC 7231, 6.6.5
	StatusHTTPVersionNotSupported       httpStatus = 505 // RFC 7231, 6.6.6
	StatusVariantAlsoNegotiates         httpStatus = 506 // RFC 2295, 8.1
	StatusInsufficientStorage           httpStatus = 507 // RFC 4918, 11.5
	StatusLoopDetected                  httpStatus = 508 // RFC 5842, 7.2
	StatusNotExtended                   httpStatus = 510 // RFC 2774, 7
	StatusNetworkAuthenticationRequired httpStatus = 511 // RFC 6585, 6
)

// StatusCoder is an error that is associated with a HTTP status code.
type StatusCoder interface {
	error
	StatusCode() int
}

type timeouter interface {
	error
	Timeout() bool
}

type temporaryer interface {
	error
	Temporary() bool
}

// StatusCode returns the HTTP status code associated with the error.
func StatusCode(err error) int {
	if err == nil {
		return http.StatusOK
	} else if sc, ok := err.(StatusCoder); ok {
		return sc.StatusCode()
	} else if to, ok := err.(timeouter); ok && to.Timeout() {
		return http.StatusGatewayTimeout
	} else if te, ok := err.(temporaryer); ok && te.Temporary() {
		return http.StatusServiceUnavailable
	}
	return http.StatusInternalServerError
}
