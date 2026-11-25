package handler

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/amitc1996/tenant-provisioning/internal/application"
)

type Handler struct {
	service *application.Service
}

func New(service *application.Service) *Handler {
	return &Handler{
		service: service,
	}
}

func (h *Handler) RegisterRoutes(router *gin.Engine) {
	api := router.Group("/api/v1")

	tenants := api.Group("/tenants")
	{
		tenants.POST("", h.CreateTenant)
		tenants.GET("", h.ListTenants)
		tenants.GET("/:id", h.GetTenant)
		tenants.PATCH("/:id", h.UpdateTenant)
		tenants.DELETE("/:id", h.DeleteTenant)
	}

	tasks := api.Group("/tasks")
	{
		tasks.GET("", h.ListTasks)
		tasks.GET("/:id", h.GetTask)
	}

	router.HandleMethodNotAllowed = true

	router.NoRoute(func(c *gin.Context) {
		writeError(
			c,
			http.StatusNotFound,
			"route_not_found",
			"Route not found",
		)
	})

	router.NoMethod(func(c *gin.Context) {
		writeError(
			c,
			http.StatusMethodNotAllowed,
			"method_not_allowed",
			"Method not allowed",
		)
	})
}

func parsePagination(c *gin.Context) (int, int, error) {
	limit, err := strconv.Atoi(
		c.DefaultQuery("limit", "20"),
	)
	if err != nil {
		return 0, 0, err
	}

	offset, err := strconv.Atoi(
		c.DefaultQuery("offset", "0"),
	)
	if err != nil {
		return 0, 0, err
	}

	return limit, offset, nil
}

func decodeJSON(c *gin.Context, dst any) bool {
	c.Request.Body = http.MaxBytesReader(
		c.Writer,
		c.Request.Body,
		1<<20,
	)

	decoder := json.NewDecoder(c.Request.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(dst); err != nil {
		writeError(
			c,
			http.StatusBadRequest,
			"invalid_request",
			"Invalid request body",
		)
		return false
	}

	if err := decoder.Decode(new(any)); err != io.EOF {
		writeError(
			c,
			http.StatusBadRequest,
			"invalid_request",
			"Request must contain one JSON object",
		)
		return false
	}

	return true
}
