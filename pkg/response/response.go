// Package response provides standardized API response envelopes.
// All API responses follow the WitFoo Way convention for consistent
// client-side handling.
package response

import (
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
)

// Response is the standard API response envelope.
type Response struct {
	Success   bool   `json:"success"`
	Message   string `json:"message,omitempty"`
	Data      any    `json:"data,omitempty"`
	Meta      *Meta  `json:"meta,omitempty"`
	Timestamp string `json:"timestamp"`
}

// Meta contains pagination metadata.
type Meta struct {
	Count    int  `json:"count"`
	Total    int  `json:"total,omitempty"`
	Page     int  `json:"page,omitempty"`
	PageSize int  `json:"page_size,omitempty"`
	HasMore  bool `json:"has_more,omitempty"`
}

// ErrorDetail provides structured error information.
type ErrorDetail struct {
	Code    string `json:"code,omitempty"`
	Field   string `json:"field,omitempty"`
	Message string `json:"message"`
}

// ErrorResponse is the standard error response envelope.
type ErrorResponse struct {
	Success   bool          `json:"success"`
	Error     string        `json:"error"`
	Details   []ErrorDetail `json:"details,omitempty"`
	Timestamp string        `json:"timestamp"`
}

func now() string {
	return time.Now().UTC().Format(time.RFC3339)
}

// OK returns a 200 success response with data.
func OK(c echo.Context, message string, data any) error {
	return c.JSON(http.StatusOK, Response{
		Success:   true,
		Message:   message,
		Data:      data,
		Timestamp: now(),
	})
}

// OKWithMeta returns a 200 success response with data and pagination metadata.
func OKWithMeta(c echo.Context, message string, data any, meta *Meta) error {
	return c.JSON(http.StatusOK, Response{
		Success:   true,
		Message:   message,
		Data:      data,
		Meta:      meta,
		Timestamp: now(),
	})
}

// Created returns a 201 created response.
func Created(c echo.Context, message string, data any) error {
	return c.JSON(http.StatusCreated, Response{
		Success:   true,
		Message:   message,
		Data:      data,
		Timestamp: now(),
	})
}

// NoContent returns a 204 no content response.
func NoContent(c echo.Context) error {
	return c.NoContent(http.StatusNoContent)
}

// BadRequest returns a 400 error response.
func BadRequest(c echo.Context, message string) error {
	return c.JSON(http.StatusBadRequest, ErrorResponse{
		Success:   false,
		Error:     message,
		Timestamp: now(),
	})
}

// BadRequestWithDetails returns a 400 error response with field-level details.
func BadRequestWithDetails(c echo.Context, message string, details []ErrorDetail) error {
	return c.JSON(http.StatusBadRequest, ErrorResponse{
		Success:   false,
		Error:     message,
		Details:   details,
		Timestamp: now(),
	})
}

// Unauthorized returns a 401 error response.
func Unauthorized(c echo.Context, message string) error {
	return c.JSON(http.StatusUnauthorized, ErrorResponse{
		Success:   false,
		Error:     message,
		Timestamp: now(),
	})
}

// Forbidden returns a 403 error response.
func Forbidden(c echo.Context, message string) error {
	return c.JSON(http.StatusForbidden, ErrorResponse{
		Success:   false,
		Error:     message,
		Timestamp: now(),
	})
}

// NotFound returns a 404 error response.
func NotFound(c echo.Context, message string) error {
	return c.JSON(http.StatusNotFound, ErrorResponse{
		Success:   false,
		Error:     message,
		Timestamp: now(),
	})
}

// Conflict returns a 409 error response.
func Conflict(c echo.Context, message string) error {
	return c.JSON(http.StatusConflict, ErrorResponse{
		Success:   false,
		Error:     message,
		Timestamp: now(),
	})
}

// TooLarge returns a 413 error response.
func TooLarge(c echo.Context, message string) error {
	return c.JSON(http.StatusRequestEntityTooLarge, ErrorResponse{
		Success:   false,
		Error:     message,
		Timestamp: now(),
	})
}

// InternalError returns a 500 error response.
// Never expose internal error details to clients.
func InternalError(c echo.Context) error {
	return c.JSON(http.StatusInternalServerError, ErrorResponse{
		Success:   false,
		Error:     "Internal server error",
		Timestamp: now(),
	})
}
