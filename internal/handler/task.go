package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (h *Handler) GetTask(c *gin.Context) {
	id := c.Param("id")

	task, err := h.service.GetTask(
		c.Request.Context(),
		id,
	)

	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(
		http.StatusOK,
		toTaskResponse(task),
	)
}

func (h *Handler) ListTasks(c *gin.Context) {
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

	tasks, err := h.service.ListTasks(
		c.Request.Context(),
		limit,
		offset,
	)

	if err != nil {
		handleError(c, err)
		return
	}

	items := make([]TaskResponse, 0, len(tasks))

	for i := range tasks {
		items = append(
			items,
			toTaskResponse(&tasks[i]),
		)
	}

	c.JSON(http.StatusOK, gin.H{
		"items":  items,
		"limit":  limit,
		"offset": offset,
		"count":  len(items),
	})
}
