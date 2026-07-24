package openapi

import (
	"errors"

	"github.com/gofiber/fiber/v3"
)

type ErrorResponse struct {
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Details map[string]string `json:"details,omitempty"`
}

type HTTPError struct {
	Status  int
	Code    string
	Message string
	Details map[string]string
	Cause   error
}

func (e *HTTPError) Error() string {
	if e.Cause != nil {
		return e.Message + ": " + e.Cause.Error()
	}
	return e.Message
}

func (e *HTTPError) Unwrap() error {
	return e.Cause
}

func writeError(c fiber.Ctx, err error) error {
	var httpError *HTTPError
	if errors.As(err, &httpError) {
		return c.Status(httpError.Status).JSON(ErrorResponse{
			Code:    httpError.Code,
			Message: httpError.Message,
			Details: httpError.Details,
		})
	}

	return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
		Code:    "internal_error",
		Message: "An internal server error occurred",
	})
}
