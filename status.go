package httpsy

import (
	"github.com/askeladdk/httpsyproblem"
)

// HTTP status codes as registered with IANA.
// See: https://www.iana.org/assignments/http-status-codes/http-status-codes.xhtml
//
// HTTP status codes are errors and can be passed to the error handler:
//  httpsy.Error(w, r, httpsy.StatusForbidden)
var (
	StatusContinue                      = httpsyproblem.Wrap(100, nil) // RFC 7231, 6.2.1
	StatusSwitchingProtocols            = httpsyproblem.Wrap(101, nil) // RFC 7231, 6.2.2
	StatusProcessing                    = httpsyproblem.Wrap(102, nil) // RFC 2518, 10.1
	StatusEarlyHints                    = httpsyproblem.Wrap(103, nil) // RFC 8297
	StatusOK                            = httpsyproblem.Wrap(200, nil) // RFC 7231, 6.3.1
	StatusCreated                       = httpsyproblem.Wrap(201, nil) // RFC 7231, 6.3.2
	StatusAccepted                      = httpsyproblem.Wrap(202, nil) // RFC 7231, 6.3.3
	StatusNonAuthoritativeInfo          = httpsyproblem.Wrap(203, nil) // RFC 7231, 6.3.4
	StatusNoContent                     = httpsyproblem.Wrap(204, nil) // RFC 7231, 6.3.5
	StatusResetContent                  = httpsyproblem.Wrap(205, nil) // RFC 7231, 6.3.6
	StatusPartialContent                = httpsyproblem.Wrap(206, nil) // RFC 7233, 4.1
	StatusMultiStatus                   = httpsyproblem.Wrap(207, nil) // RFC 4918, 11.1
	StatusAlreadyReported               = httpsyproblem.Wrap(208, nil) // RFC 5842, 7.1
	StatusIMUsed                        = httpsyproblem.Wrap(226, nil) // RFC 3229, 10.4.1
	StatusMultipleChoices               = httpsyproblem.Wrap(300, nil) // RFC 7231, 6.4.1
	StatusMovedPermanently              = httpsyproblem.Wrap(301, nil) // RFC 7231, 6.4.2
	StatusFound                         = httpsyproblem.Wrap(302, nil) // RFC 7231, 6.4.3
	StatusSeeOther                      = httpsyproblem.Wrap(303, nil) // RFC 7231, 6.4.4
	StatusNotModified                   = httpsyproblem.Wrap(304, nil) // RFC 7232, 4.1
	StatusUseProxy                      = httpsyproblem.Wrap(305, nil) // RFC 7231, 6.4.5
	StatusTemporaryRedirect             = httpsyproblem.Wrap(307, nil) // RFC 7231, 6.4.7
	StatusPermanentRedirect             = httpsyproblem.Wrap(308, nil) // RFC 7538, 3
	StatusBadRequest                    = httpsyproblem.Wrap(400, nil) // RFC 7231, 6.5.1
	StatusUnauthorized                  = httpsyproblem.Wrap(401, nil) // RFC 7235, 3.1
	StatusPaymentRequired               = httpsyproblem.Wrap(402, nil) // RFC 7231, 6.5.2
	StatusForbidden                     = httpsyproblem.Wrap(403, nil) // RFC 7231, 6.5.3
	StatusNotFound                      = httpsyproblem.Wrap(404, nil) // RFC 7231, 6.5.4
	StatusMethodNotAllowed              = httpsyproblem.Wrap(405, nil) // RFC 7231, 6.5.5
	StatusNotAcceptable                 = httpsyproblem.Wrap(406, nil) // RFC 7231, 6.5.6
	StatusProxyAuthRequired             = httpsyproblem.Wrap(407, nil) // RFC 7235, 3.2
	StatusRequestTimeout                = httpsyproblem.Wrap(408, nil) // RFC 7231, 6.5.7
	StatusConflict                      = httpsyproblem.Wrap(409, nil) // RFC 7231, 6.5.8
	StatusGone                          = httpsyproblem.Wrap(410, nil) // RFC 7231, 6.5.9
	StatusLengthRequired                = httpsyproblem.Wrap(411, nil) // RFC 7231, 6.5.10
	StatusPreconditionFailed            = httpsyproblem.Wrap(412, nil) // RFC 7232, 4.2
	StatusRequestEntityTooLarge         = httpsyproblem.Wrap(413, nil) // RFC 7231, 6.5.11
	StatusRequestURITooLong             = httpsyproblem.Wrap(414, nil) // RFC 7231, 6.5.12
	StatusUnsupportedMediaType          = httpsyproblem.Wrap(415, nil) // RFC 7231, 6.5.13
	StatusRequestedRangeNotSatisfiable  = httpsyproblem.Wrap(416, nil) // RFC 7233, 4.4
	StatusExpectationFailed             = httpsyproblem.Wrap(417, nil) // RFC 7231, 6.5.14
	StatusTeapot                        = httpsyproblem.Wrap(418, nil) // RFC 7168, 2.3.3
	StatusMisdirectedRequest            = httpsyproblem.Wrap(421, nil) // RFC 7540, 9.1.2
	StatusUnprocessableEntity           = httpsyproblem.Wrap(422, nil) // RFC 4918, 11.2
	StatusLocked                        = httpsyproblem.Wrap(423, nil) // RFC 4918, 11.3
	StatusFailedDependency              = httpsyproblem.Wrap(424, nil) // RFC 4918, 11.4
	StatusTooEarly                      = httpsyproblem.Wrap(425, nil) // RFC 8470, 5.2.
	StatusUpgradeRequired               = httpsyproblem.Wrap(426, nil) // RFC 7231, 6.5.15
	StatusPreconditionRequired          = httpsyproblem.Wrap(428, nil) // RFC 6585, 3
	StatusTooManyRequests               = httpsyproblem.Wrap(429, nil) // RFC 6585, 4
	StatusRequestHeaderFieldsTooLarge   = httpsyproblem.Wrap(431, nil) // RFC 6585, 5
	StatusUnavailableForLegalReasons    = httpsyproblem.Wrap(451, nil) // RFC 7725, 3
	StatusInternalServerError           = httpsyproblem.Wrap(500, nil) // RFC 7231, 6.6.1
	StatusNotImplemented                = httpsyproblem.Wrap(501, nil) // RFC 7231, 6.6.2
	StatusBadGateway                    = httpsyproblem.Wrap(502, nil) // RFC 7231, 6.6.3
	StatusServiceUnavailable            = httpsyproblem.Wrap(503, nil) // RFC 7231, 6.6.4
	StatusGatewayTimeout                = httpsyproblem.Wrap(504, nil) // RFC 7231, 6.6.5
	StatusHTTPVersionNotSupported       = httpsyproblem.Wrap(505, nil) // RFC 7231, 6.6.6
	StatusVariantAlsoNegotiates         = httpsyproblem.Wrap(506, nil) // RFC 2295, 8.1
	StatusInsufficientStorage           = httpsyproblem.Wrap(507, nil) // RFC 4918, 11.5
	StatusLoopDetected                  = httpsyproblem.Wrap(508, nil) // RFC 5842, 7.2
	StatusNotExtended                   = httpsyproblem.Wrap(510, nil) // RFC 2774, 7
	StatusNetworkAuthenticationRequired = httpsyproblem.Wrap(511, nil) // RFC 6585, 6
)
