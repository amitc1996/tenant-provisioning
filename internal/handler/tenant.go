package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/amitc1996/tenant-provisioning/internal/application"
)

func (h *Handler) CreateTenant(c *gin.Context) {
	var request CreateTenantRequest

	if !decodeJSON(c, &request) {
		return
	}

	result, err := h.service.CreateTenant(
		c.Request.Context(),
		application.CreateTenantInput{
			Slug: request.Slug,
			Name: request.Name,
		},
	)

	if err != nil {
		handleError(c, err)
		return
	}

	c.Header(
		"Location",
		"/api/v1/tenants/"+result.Tenant.ID,
	)

	c.JSON(
		http.StatusCreated,
		toMutationResponse(result),
	)
}

func (h *Handler) GetTenant(c *gin.Context) {
	id := c.Param("id")

	tenant, err := h.service.GetTenant(
		c.Request.Context(),
		id,
	)

	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(
		http.StatusOK,
		toTenantResponse(tenant),
	)
}

func (h *Handler) ListTenants(c *gin.Context) {
	limit, offset, err := parsePagination(c)

	if err != nil {
		writeError(
			c,
			http.StatusBadRequest,
			"invalid_pagination",
			"Invalid pagination parameters",
		)
		return
	}

	tenants, err := h.service.ListTenants(
		c.Request.Context(),
		limit,
		offset,
	)

	if err != nil {
		handleError(c, err)
		return
	}

	items := make([]TenantResponse, 0, len(tenants))

	for i := range tenants {
		items = append(
			items,
			toTenantResponse(&tenants[i]),
		)
	}

	c.JSON(http.StatusOK, gin.H{
		"items":  items,
		"limit":  limit,
		"offset": offset,
		"count":  len(items),
	})
}

func (h *Handler) UpdateTenant(c *gin.Context) {
	id := c.Param("id")

	var request UpdateTenantRequest

	if !decodeJSON(c, &request) {
		return
	}

	result, err := h.service.UpdateTenant(
		c.Request.Context(),
		id,
		application.UpdateTenantInput{
			Slug:    request.Slug,
			Name:    request.Name,
			Version: request.Version,
		},
	)

	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(
		http.StatusAccepted,
		toMutationResponse(result),
	)
}

func (h *Handler) DeleteTenant(c *gin.Context) {
	id := c.Param("id")

	result, err := h.service.DeleteTenant(
		c.Request.Context(),
		id,
	)

	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(
		http.StatusAccepted,
		toMutationResponse(result),
	)
}
