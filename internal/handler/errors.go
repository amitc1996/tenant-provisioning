package handler

import (
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/amitc1996/tenant-provisioning/internal/domain"
)

type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

type ErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func writeError(
	c *gin.Context,
	status int,
	code string,
	message string,
) {
	c.JSON(status, ErrorResponse{
		Error: ErrorDetail{
			Code:    code,
			Message: message,
		},
	})
}

func handleError(c *gin.Context, err error) {
	switch {

	case errors.Is(err, domain.ErrTenantNotFound):
		writeError(
			c,
			http.StatusNotFound,
			"tenant_not_found",
			"Tenant not found",
		)

	case errors.Is(err, domain.ErrTenantAlreadyExists):
		writeError(
			c,
			http.StatusConflict,
			"tenant_already_exists",
			"Tenant slug already exists",
		)

	case errors.Is(err, domain.ErrTenantUpdateNotAllowed):
		writeError(
			c,
			http.StatusConflict,
			"tenant_update_not_allowed",
			"Operation not allowed in current tenant status",
		)

	case errors.Is(err, domain.ErrTenantVersionConflict):
		writeError(
			c,
			http.StatusConflict,
			"tenant_version_conflict",
			"Tenant version does not match current version",
		)

	case errors.Is(err, domain.ErrTaskNotFound):
		writeError(
			c,
			http.StatusNotFound,
			"task_not_found",
			"Task not found",
		)

	case errors.Is(err, domain.ErrInvalidSlug):
		writeError(
			c,
			http.StatusBadRequest,
			"invalid_slug",
			"Invalid tenant slug",
		)

	case errors.Is(err, domain.ErrInvalidName):
		writeError(
			c,
			http.StatusBadRequest,
			"invalid_name",
			"Tenant name must not be empty",
		)

	case errors.Is(err, domain.ErrInvalidVersion):
		writeError(
			c,
			http.StatusBadRequest,
			"invalid_version",
			"A positive tenant version is required",
		)

	case errors.Is(err, domain.ErrEmptyTenantUpdate):
		writeError(
			c,
			http.StatusBadRequest,
			"empty_tenant_update",
			"At least one editable field is required",
		)

	case errors.Is(err, domain.ErrInvalidPagination):
		writeError(
			c,
			http.StatusBadRequest,
			"invalid_pagination",
			"Invalid pagination parameters",
		)

	default:
		log.Printf("request failed: %T", err)

		writeError(
			c,
			http.StatusInternalServerError,
			"internal_server_error",
			"An unexpected error occurred",
		)
	}
}
