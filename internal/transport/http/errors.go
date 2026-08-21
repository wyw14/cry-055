package httptransport

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/wyw14/cry-055/internal/domain"
	"github.com/wyw14/cry-055/internal/middleware"
)

type errorResponse struct {
	Code        string              `json:"code"`
	Message     string              `json:"message"`
	FieldErrors []domain.FieldError `json:"field_errors"`
	RequestID   string              `json:"request_id"`
}

func writeError(c *gin.Context, err error) {
	status, code, message := http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error"
	fields := []domain.FieldError{}
	var validation domain.ValidationError
	switch {
	case errors.As(err, &validation):
		status, code, message = http.StatusUnprocessableEntity, "VALIDATION_ERROR", "request validation failed"
		fields = validation.Fields
	case errors.Is(err, domain.ErrNotFound):
		status, code, message = http.StatusNotFound, "NOT_FOUND", "resource not found"
	case errors.Is(err, domain.ErrDuplicate):
		status, code, message = http.StatusConflict, "DUPLICATE_RESOURCE", "resource already exists"
	case errors.Is(err, domain.ErrConflict):
		status, code, message = http.StatusConflict, "VERSION_CONFLICT", "resource version has changed"
	case errors.Is(err, domain.ErrInvalidTransition):
		status, code, message = http.StatusConflict, "INVALID_TRANSITION", "state transition is not allowed"
	case errors.Is(err, domain.ErrInstrumentBlocked):
		status, code, message = http.StatusLocked, "INSTRUMENT_BLOCKED", "instrument cannot be used"
	case errors.Is(err, domain.ErrUnauthorized):
		status, code, message = http.StatusForbidden, "FORBIDDEN", "operation is not authorized"
	case errors.Is(err, context.DeadlineExceeded):
		status, code, message = http.StatusGatewayTimeout, "REQUEST_TIMEOUT", "request timed out"
	}
	c.AbortWithStatusJSON(status, errorResponse{Code: code, Message: message, FieldErrors: fields, RequestID: middleware.GetRequestID(c)})
}
