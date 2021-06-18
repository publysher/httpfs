package httpfs

import (
	"fmt"
	"io/fs"
	"net/http"
)

type StatusError struct {
	StatusCode int
	Status     string
	Err        error
}

// AsStatusError converts an HTTP status code to a StatusError. If the status code is in the 2xx range,
// it returns nil.
func AsStatusError(statusCode int, status string) *StatusError {
	switch statusCode {
	case http.StatusOK:
		return nil
	case http.StatusNotFound:
		return &StatusError{
			StatusCode: statusCode,
			Status:     status,
			Err:        fs.ErrNotExist,
		}
	case http.StatusUnauthorized, http.StatusForbidden:
		return &StatusError{
			StatusCode: statusCode,
			Status:     status,
			Err:        fs.ErrPermission,
		}
	default:
		return &StatusError{
			StatusCode: statusCode,
			Status:     status,
			Err:        fs.ErrInvalid,
		}
	}
}

func (s *StatusError) Error() string {
	return fmt.Sprintf("%s: %s", s.Err, s.Status)
}

func (s *StatusError) Unwrap() error {
	return s.Err
}
