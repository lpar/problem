package problem

import (
	"net/http"
)

// ValidationProblem is an example of extending the ProblemDetails structure
// as per the form validation example in section 3 of RFC 7807, to support
// reporting of server-side data validation errors.
type ValidationProblem struct {
	ProblemDetails
	ValidationErrors []ValidationError `json:"invalid-params,omitempty"`
}

// ValidationError indicates a server-side validation error for data submitted
// as JSON or via a web form.
type ValidationError struct {
	FieldName string `json:"name"`
	Error     string `json:"reason"`
}

// NewValidationProblem creates an object to represent a server-side validation error.
func NewValidationProblem() *ValidationProblem{
	return &ValidationProblem{
		ProblemDetails: ProblemDetails{ Status: http.StatusBadRequest, Detail: "Validation error"},
		ValidationErrors: []ValidationError{},
	}
}

// Add adds a validation error message for the specified field to the ValidationProblem.
func (vp *ValidationProblem) Add(field string, errmsg string) {
	ve := ValidationError{field, errmsg}
	vp.ValidationErrors = append(vp.ValidationErrors, ve)
}
