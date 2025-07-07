package utils

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type Response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   *AppError   `json:"error,omitempty"`
	Meta    *Meta       `json:"meta,omitempty"`
}

type Meta struct {
	Page       int   `json:"page,omitempty"`
	PerPage    int   `json:"per_page,omitempty"`
	Total      int64 `json:"total,omitempty"`
	TotalPages int   `json:"total_pages,omitempty"`
}

func SendSuccess(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Success: true,
		Data:    data,
	})
}

func SendSuccessWithMeta(c *gin.Context, data interface{}, meta *Meta) {
	c.JSON(http.StatusOK, Response{
		Success: true,
		Data:    data,
		Meta:    meta,
	})
}

func SendError(c *gin.Context, statusCode int, err *AppError) {
	c.JSON(statusCode, Response{
		Success: false,
		Error:   err,
	})
}

func SendValidationError(c *gin.Context, message string, details string) {
	SendError(c, http.StatusBadRequest, NewAppError(ErrCodeValidation, message, details))
}

func SendNotFound(c *gin.Context, message string) {
	SendError(c, http.StatusNotFound, NewAppError(ErrCodeNotFound, message))
}

func SendUnauthorized(c *gin.Context, message string) {
	SendError(c, http.StatusUnauthorized, NewAppError(ErrCodeUnauthorized, message))
}

func SendForbidden(c *gin.Context, message string) {
	SendError(c, http.StatusForbidden, NewAppError(ErrCodeForbidden, message))
}

func SendInternalError(c *gin.Context, message string) {
	SendError(c, http.StatusInternalServerError, NewAppError(ErrCodeInternal, message))
}

func SendConflict(c *gin.Context, message string) {
	SendError(c, http.StatusConflict, NewAppError(ErrCodeConflict, message))
}
