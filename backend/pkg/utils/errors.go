package utils

import (
	"errors"
	"fmt"
)

var (
	ErrNotFound           = errors.New("resource not found")
	ErrInvalidInput       = errors.New("invalid input")
	ErrUnauthorized       = errors.New("unauthorized")
	ErrForbidden          = errors.New("forbidden")
	ErrInternalServer     = errors.New("internal server error")
	ErrConflict           = errors.New("resource conflict")
	ErrBadRequest         = errors.New("bad request")
	ErrOptimizationFailed = errors.New("optimization failed")
	ErrSimulationFailed   = errors.New("simulation failed")
)

type AppError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

func NewAppError(code string, message string, details ...string) *AppError {
	err := &AppError{
		Code:    code,
		Message: message,
	}
	if len(details) > 0 {
		err.Details = details[0]
	}
	return err
}

func (e *AppError) Error() string {
	if e.Details != "" {
		return fmt.Sprintf("%s: %s - %s", e.Code, e.Message, e.Details)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Common error codes
const (
	ErrCodeValidation        = "VALIDATION_ERROR"
	ErrCodeNotFound          = "NOT_FOUND"
	ErrCodeUnauthorized      = "UNAUTHORIZED"
	ErrCodeForbidden         = "FORBIDDEN"
	ErrCodeInternal          = "INTERNAL_ERROR"
	ErrCodeConflict          = "CONFLICT"
	ErrCodeOptimization      = "OPTIMIZATION_ERROR"
	ErrCodeSimulation        = "SIMULATION_ERROR"
	ErrCodeSalaryCapExceeded = "SALARY_CAP_EXCEEDED"
	ErrCodeInvalidLineup     = "INVALID_LINEUP"
)
