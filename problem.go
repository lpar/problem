package problem

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
)

// ContentProblemDetails is the correct MIME type to use when returning a
// problem details object as JSON.
const ContentProblemDetails = "application/problem+json"

// ProblemDetails provides a standard encapsulation for problems encountered
// in web applications and REST APIs.
type ProblemDetails struct {
	Status       int    `json:"status,omitempty"`
	Title        string `json:"title,omitempty"`
	Detail       string `json:"detail,omitempty"`
	Type         string `json:"type,omitempty"`
	Instance     string `json:"instance,omitempty"`
	wrappedError error
}

// HTTPError is the minimal interface needed to be able to Write a problem,
// defined so that ProblemDetails can be encapsulated and expanded as needed.
type HTTPError interface {
	GetStatus() int
}

// GetStatus implements the HTTPError interface
func (pd ProblemDetails) GetStatus() int {
	return pd.Status
}

// New implements the error interface, so ProblemDetails objects can be used
// as regular error return values.
func (pd ProblemDetails) Error() string {
	return pd.Title
}

// Unwrap implements the Go 1.13+ unwrapping interface for errors.
func (pd ProblemDetails) Unwrap() error {
	return pd.wrappedError
}

const rfcBase = "https://tools.ietf.org/html/"

var typeForStatus = map[int]string{
	http.StatusBadRequest:                    "Bad Request",
	http.StatusUnauthorized:                  "Unauthorized",
	http.StatusPaymentRequired:               "Payment Required",
	http.StatusForbidden:                     "Forbidden",
	http.StatusNotFound:                      "Not Found",
	http.StatusMethodNotAllowed:              "Method Not Allowed",
	http.StatusNotAcceptable:                 "Not Acceptable",
	http.StatusProxyAuthRequired:             "Proxy Authentication Required",
	http.StatusRequestTimeout:                "Request Timeout",
	http.StatusConflict:                      "Conflict",
	http.StatusGone:                          "Gone",
	http.StatusLengthRequired:                "Length Required",
	http.StatusPreconditionFailed:            "Precondition Failed",
	http.StatusRequestEntityTooLarge:         "Payload Too Large",
	http.StatusRequestURITooLong:             "URI Too Long",
	http.StatusUnsupportedMediaType:          "Unsupported Media Type",
	http.StatusRequestedRangeNotSatisfiable:  "Range Not Satisfiable",
	http.StatusExpectationFailed:             "Expectation Failed",
	http.StatusTeapot:                        "I'm a teapot",
	http.StatusMisdirectedRequest:            "Misdirected Request",
	http.StatusUnprocessableEntity:           "Unprocessable Entity",
	http.StatusLocked:                        "Locked",
	http.StatusFailedDependency:              "Failed Dependency",
	http.StatusTooEarly:                      "Too Early",
	http.StatusUpgradeRequired:               "Upgrade Required",
	http.StatusPreconditionRequired:          "Precondition Required",
	http.StatusTooManyRequests:               "Too Many Requests",
	http.StatusRequestHeaderFieldsTooLarge:   "Request Header Fields Too Large",
	http.StatusUnavailableForLegalReasons:    "Unavailable For Legal Reasons",
	499:                                      "Client Closed Request",
	http.StatusInternalServerError:           "Internal Server Error",
	http.StatusNotImplemented:                "Not Implemented",
	http.StatusBadGateway:                    "Bad Gateway",
	http.StatusServiceUnavailable:            "Service Unavailable",
	http.StatusGatewayTimeout:                "Gateway Timeout",
	http.StatusHTTPVersionNotSupported:       "HTTP Version Not Supported",
	http.StatusVariantAlsoNegotiates:         "Variant Also Negotiates",
	http.StatusInsufficientStorage:           "Insufficient Storage",
	http.StatusLoopDetected:                  "Loop Detected",
	http.StatusNotExtended:                   "Not Extended",
	http.StatusNetworkAuthenticationRequired: "Network Authentication Required",
	599:                                      "Network Connect Timeout New",
}

//// Fluent API

// New returns a ProblemDetails error object with the given HTTP status code.
func New(status int) *ProblemDetails {
	return &ProblemDetails{
		Status: status,
		Title:  typeForStatus[status],
		Type:   "https://httpstatuses.com/" + strconv.Itoa(status),
	}
}

// Errorf uses fmt.Errorf to add a detail message to the ProblemDetails object.
// It supports the %w verb.
func (pd *ProblemDetails) Errorf(fmtstr string, args ...interface{}) *ProblemDetails {
	err := fmt.Errorf(fmtstr, args...)
	pd.wrappedError = errors.Unwrap(err)
	pd.Detail = err.Error()
	return pd
}

// WithDetail adds the supplied detail message to the problem details.
func (pd *ProblemDetails) WithDetail(msg string) *ProblemDetails {
	pd.Detail = msg
	return pd
}

// WithErr adds an error value as a wrapped error. If the error detail message
// is currently blank, it is initialized from the error's New() message.
func (pd *ProblemDetails) WithErr(err error) *ProblemDetails {
	pd.wrappedError = err
	if pd.Detail == "" {
		pd.Detail = err.Error()
	}
	return pd
}

// rawWrite implements writing anything which satisfies HTTPError, as a JSON
// problem details object.
func rawWrite(w http.ResponseWriter, obj HTTPError) error {
	w.Header().Set(http.CanonicalHeaderKey("Content-Type"), ContentProblemDetails)
	w.WriteHeader(obj.GetStatus())
	return json.NewEncoder(w).Encode(obj)
}

// Write sets the HTTP response code from the ProblemDetails and then sends the
// entire object as JSON.
func (pd *ProblemDetails) Write(w http.ResponseWriter) error {
	return rawWrite(w, pd)
}

//// Non-fluent API

// Write writes the supplied error if it's a ProblemDetails, returning nil;
// otherwise it returns the error untouched for the caller to handle.
func Write(w http.ResponseWriter, err error) error {
	if err == nil {
		return nil
	}
	switch r := err.(type) {
	/* case ProblemDetails:
	return r.Write(w) */
	case HTTPError:
		return rawWrite(w, r)
	case error:
		return r
	default:
		return fmt.Errorf("can't write non-error type %T", err)
	}
}

// MustWrite is like Write, but if the error isn't a ProblemDetails object
// the error is written as a new problem details object, HTTP Internal Server
// Error.
func MustWrite(w http.ResponseWriter, err error) error {
	err = Write(w, err)
	if err != nil {
		return New(http.StatusInternalServerError).WithErr(err).Write(w)
	}
	return nil
}

// Errorf is used like fmt.Errorf to create and return errors. It takes an
// extra first argument of the HTTP status to use.
func Errorf(status int, fmtstr string, args ...interface{}) *ProblemDetails {
	return New(status).Errorf(fmtstr, args...)
}

// Error is used just like http.Error to create and immediately issue an error.
func Error(w http.ResponseWriter, msg string, status int) error {
	return New(status).WithDetail(msg).Write(w)
}
