// Package httpsyproblem implements RFC 7807 which specifies a way to carry machine-
// readable details of errors in a HTTP response to avoid the need to
// define new error response formats for HTTP APIs.
package httpsyproblem

import (
	"fmt"
	"net/http"
)

// Detailer identifies errors that implement RFC 7807.
type Detailer interface {
	error
	ProblemDetailer() bool
}

// Details implements the RFC 7807 model.
//
// Additional fields can be added by embedding Details inside another struct:
//
//  type MyDetails {
//      httpsyproblem.Details
//      MyCode int `json:"myCode,omitempty"`
//  }
type Details struct {
	// A human-readable explanation specific to this occurrence of the problem.
	Detail string `json:"detail,omitempty"`

	// A URI reference that identifies the specific occurrence of the problem.
	// It may or may not yield further information if dereferenced.
	Instance string `json:"instance,omitempty"`

	// The HTTP status code ([RFC7231], Section 6)
	// generated by the origin server for this occurrence of the problem.
	Status int `json:"status,omitempty"`

	// A short, human-readable summary of the problem
	// type. It SHOULD NOT change from occurrence to occurrence of the
	// problem, except for purposes of localization (e.g., using
	// proactive content negotiation; see [RFC7231], Section 3.4).
	Title string `json:"title,omitempty"`

	// A URI reference [RFC3986] that identifies the
	// problem type. This specification encourages that, when
	// dereferenced, it provide human-readable documentation for the
	// problem type (e.g., using HTML [W3C.REC-html5-20141028]). When
	// this member is not present, its value is assumed to be
	// "about:blank".
	Type string `json:"type,omitempty"`

	wrappedError error
}

// Wrap associates an error with a status code and wraps it in a Details.
// The Detail field is set to err.Error().
// The Status and Title fields are set according to statusCode.
func Wrap(err error, statusCode int) (details Details) {
	var ok bool

	if details, ok = err.(Details); !ok {
		if err != nil {
			details.Detail = err.Error()
		}
	}

	details.Status = statusCode
	details.Title = http.StatusText(statusCode)
	details.wrappedError = err
	return
}

// Detailf set the Detail field to a formatted string.
func (details *Details) Detailf(format string, a ...interface{}) {
	details.Detail = fmt.Sprintf(format, a...)
}

// Error implements the error interface and returns the Title field.
func (details Details) Error() string { return details.Title }

// StatusCode implements the httpsy.StatusCoder and returns the Status field.
func (details Details) StatusCode() int { return details.Status }

// Unwrap implements the interface used by errors.Unwrap() and returns the wrapped error.
func (details Details) Unwrap() error { return details.wrappedError }

// ProblemDetailer implements Detailer.
func (details Details) ProblemDetailer() bool { return true }
