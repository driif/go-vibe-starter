package errs

import (
	"encoding/json"
	"net/http"
)

type response struct {
	Status int    `json:"status"`
	Title  string `json:"title"`
	Detail string `json:"detail,omitempty"`
}

type validationResponse struct {
	response
	Errors []string `json:"errors"`
}

// Write writes err as a JSON problem response with the given HTTP status code.
// Use the sentinel errors already defined in auth or keycloak packages as err.
func Write(w http.ResponseWriter, status int, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(response{
		Status: status,
		Title:  http.StatusText(status),
		Detail: err.Error(),
	})
}

// WriteValidation writes a 400 response with a list of validation error strings.
func WriteValidation(w http.ResponseWriter, err error, errs []string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	_ = json.NewEncoder(w).Encode(validationResponse{
		response: response{
			Status: http.StatusBadRequest,
			Title:  http.StatusText(http.StatusBadRequest),
			Detail: err.Error(),
		},
		Errors: errs,
	})
}
