package httptransport

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/wyw14/cry-055/internal/domain"
)

var validate = validator.New(validator.WithRequiredStructEnabled())

func bindJSON(c *gin.Context, target any) error {
	if err := c.ShouldBindJSON(target); err != nil {
		return domain.NewValidationError("body", "request body must be valid JSON")
	}
	if err := validate.Struct(target); err != nil {
		if failures, ok := err.(validator.ValidationErrors); ok {
			fields := make([]domain.FieldError, 0, len(failures))
			for _, failure := range failures {
				fields = append(fields, domain.FieldError{Field: strings.ToLower(failure.Field()), Message: "failed " + failure.Tag() + " validation"})
			}
			return domain.ValidationError{Fields: fields}
		}
		return domain.NewValidationError("body", err.Error())
	}
	return nil
}
func parseVersion(c *gin.Context) (domain.Version, error) {
	raw := strings.Trim(strings.TrimSpace(c.GetHeader("If-Match")), `"`)
	if raw == "" {
		return 0, domain.NewValidationError("If-Match", "version header is required")
	}
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || value <= 0 {
		return 0, domain.NewValidationError("If-Match", "version header must be a positive integer")
	}
	return domain.Version(value), nil
}
func pageRequest(c *gin.Context) (domain.PageRequest, error) {
	page := queryIntOrDefault(c, "page", 1)
	size := queryIntOrDefault(c, "size", 20)
	sortField := c.DefaultQuery("sort", "created_at")
	desc := strings.EqualFold(c.Query("order"), "desc")
	filters := map[string]string{}
	for key, values := range c.Request.URL.Query() {
		if strings.HasPrefix(key, "filter[") && strings.HasSuffix(key, "]") && len(values) > 0 {
			filters[strings.TrimSuffix(strings.TrimPrefix(key, "filter["), "]")] = values[0]
		}
	}
	return domain.PageRequest{Page: page, Size: size, Sort: sortField, Desc: desc, Filters: filters}, nil
}

func queryIntOrDefault(c *gin.Context, name string, fallback int) int {
	value, err := strconv.Atoi(strings.TrimSpace(c.Query(name)))
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}
func requireParam(c *gin.Context, name string) (domain.ID, error) {
	value := strings.TrimSpace(c.Param(name))
	if value == "" {
		return "", domain.NewValidationError(name, fmt.Sprintf("%s is required", name))
	}
	return domain.ID(value), nil
}
