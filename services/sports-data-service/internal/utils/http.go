package utils

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// ErrorResponse represents an API error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
	Code    int    `json:"code"`
}

// SuccessResponse represents a successful API response
type SuccessResponse struct {
	Data    interface{} `json:"data"`
	Message string      `json:"message,omitempty"`
}

// SendError sends a generic error response
func SendError(c *gin.Context, statusCode int, message string) {
	c.JSON(statusCode, ErrorResponse{
		Error:   http.StatusText(statusCode),
		Message: message,
		Code:    statusCode,
	})
}

// SendInternalError sends a 500 internal server error
func SendInternalError(c *gin.Context, message string) {
	SendError(c, http.StatusInternalServerError, message)
}

// SendBadRequest sends a 400 bad request error
func SendBadRequest(c *gin.Context, message string) {
	SendError(c, http.StatusBadRequest, message)
}

// SendNotFound sends a 404 not found error
func SendNotFound(c *gin.Context, message string) {
	SendError(c, http.StatusNotFound, message)
}

// SendUnauthorized sends a 401 unauthorized error
func SendUnauthorized(c *gin.Context, message string) {
	SendError(c, http.StatusUnauthorized, message)
}

// SendForbidden sends a 403 forbidden error
func SendForbidden(c *gin.Context, message string) {
	SendError(c, http.StatusForbidden, message)
}

// SendSuccess sends a 200 success response
func SendSuccess(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, SuccessResponse{
		Data: data,
	})
}

// SendSuccessWithMessage sends a 200 success response with message
func SendSuccessWithMessage(c *gin.Context, data interface{}, message string) {
	c.JSON(http.StatusOK, SuccessResponse{
		Data:    data,
		Message: message,
	})
}

// SendCreated sends a 201 created response
func SendCreated(c *gin.Context, data interface{}) {
	c.JSON(http.StatusCreated, SuccessResponse{
		Data: data,
	})
}

// SendValidationError sends a 422 validation error
func SendValidationError(c *gin.Context, message string) {
	SendError(c, http.StatusUnprocessableEntity, message)
}