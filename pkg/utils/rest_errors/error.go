package rest_errors

import "net/http"

type RestError interface {
	Message() string
	Status() int
	Error() string
	Causes() []interface{}
}

type restError struct {
	ErrorMessage string        `json:"message"`
	ErrorStatus  int           `json:"status"`
	ErrorError   string        `json:"error"`
	ErrorCauses  []interface{} `json:"causes"`
}

func (r restError) Message() string {
	return r.ErrorMessage
}

func (r restError) Status() int {
	return r.ErrorStatus
}

func (r restError) Error() string {
	return r.ErrorError
}

func (r restError) Causes() []interface{} {
	return r.ErrorCauses
}

func NewRestError(message string, status int, err string, causes []interface{}) RestError {
	return restError{
		ErrorMessage: message,
		ErrorStatus:  status,
		ErrorError:   err,
		ErrorCauses:  causes,
	}
}

func NewBadRequestError(message string) RestError {
	return restError{
		ErrorMessage: message,
		ErrorStatus:  http.StatusBadRequest,
		ErrorError:   "bad_request",
	}
}

func NewNotFoundError(message string) RestError {
	return restError{
		ErrorMessage: message,
		ErrorStatus:  http.StatusNotFound,
		ErrorError:   "not_found",
	}
}

func NewUnauthorizedError(message string) RestError {
	return restError{
		ErrorMessage: message,
		ErrorStatus:  http.StatusUnauthorized,
		ErrorError:   "unauthorized",
	}
}

func NewInternalServerError(message string, err error) RestError {
	result := restError{
		ErrorMessage: message,
		ErrorStatus:  http.StatusInternalServerError,
		ErrorError:   "internal_server_error",
	}
	if err != nil {
		result.ErrorCauses = append(result.ErrorCauses, err.Error())
	}
	return result
}