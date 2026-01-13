package startups

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"grveyard/pkg/response"
)

type StartupHandler struct {
	service StartupService
}

func NewStartupHandler(service StartupService) *StartupHandler {
	return &StartupHandler{service: service}
}

func isValidStatus(status string) bool {
	if status == "" {
		return true
	}
	switch status {
	case "active", "failed", "sold":
		return true
	default:
		return false
	}
}

func (h *StartupHandler) RegisterRoutes(router *gin.Engine) {
	router.POST("/startups", h.createStartup)
	router.PUT("/startups/:id", h.updateStartup)
	router.DELETE("/startups/:id", h.deleteStartup)
	router.DELETE("/startups", h.deleteAllStartups)
	router.GET("/startups", h.listStartups)
	router.GET("/startups/user/:uuid", h.ListStartupsByUser)
	router.GET("/startups/:id", h.getStartupByID)
}

type createStartupRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	LogoURL     string `json:"logo_url"`
	OwnerUUID   string `json:"owner_uuid" binding:"required"`
	Status      string `json:"status"`
}

type updateStartupRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	LogoURL     string `json:"logo_url"`
	Status      string `json:"status"`
}

// @Summary      Create a new startup
// @Description  Creates a new startup with the provided details
// @Tags         startups
// @Accept       json
// @Produce      json
// @Param        request body createStartupRequest true "Startup creation request"
// @Success      201  {object}  response.APIResponse{data=Startup} "Startup created successfully"
// @Failure      400  {object}  response.APIResponse "Invalid request payload"
// @Failure      500  {object}  response.APIResponse "Internal server error"
// @Router       /startups [post]
func (h *StartupHandler) createStartup(c *gin.Context) {
	var req createStartupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.SendAPIResponse(c, http.StatusBadRequest, false, "invalid request payload", nil)
		return
	}

	if req.OwnerUUID == "" {
		response.SendAPIResponse(c, http.StatusBadRequest, false, "owner_uuid must be provided", nil)
		return
	}

	if !isValidStatus(req.Status) {
		response.SendAPIResponse(c, http.StatusBadRequest, false, "invalid status", nil)
		return
	}

	startup, err := h.service.CreateStartup(c.Request.Context(), Startup{
		Name:        req.Name,
		Description: req.Description,
		LogoURL:     req.LogoURL,
		OwnerUUID:   req.OwnerUUID,
		Status:      req.Status,
	})
	if err != nil {
		response.SendAPIResponse(c, http.StatusInternalServerError, false, err.Error(), nil)
		return
	}

	response.SendAPIResponse(c, http.StatusCreated, true, "startup created", startup)
}

// @Summary      Update a startup
// @Description  Updates an existing startup's details
// @Tags         startups
// @Accept       json
// @Produce      json
// @Param        id   path      int  true  "Startup ID"
// @Param        request body updateStartupRequest true "Startup update request"
// @Success      200  {object}  response.APIResponse{data=Startup} "Startup updated successfully"
// @Failure      400  {object}  response.APIResponse "Invalid request"
// @Failure      404  {object}  response.APIResponse "Startup not found"
// @Failure      500  {object}  response.APIResponse "Internal server error"
// @Router       /startups/{id} [put]
func (h *StartupHandler) updateStartup(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.SendAPIResponse(c, http.StatusBadRequest, false, "invalid startup id", nil)
		return
	}

	var req updateStartupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.SendAPIResponse(c, http.StatusBadRequest, false, "invalid request payload", nil)
		return
	}

	if !isValidStatus(req.Status) {
		response.SendAPIResponse(c, http.StatusBadRequest, false, "invalid status", nil)
		return
	}

	startup, err := h.service.UpdateStartup(c.Request.Context(), Startup{
		ID:          id,
		Name:        req.Name,
		Description: req.Description,
		LogoURL:     req.LogoURL,
		Status:      req.Status,
	})
	if err != nil {
		if err == ErrStartupNotFound {
			response.SendAPIResponse(c, http.StatusNotFound, false, "startup not found", nil)
			return
		}
		response.SendAPIResponse(c, http.StatusInternalServerError, false, err.Error(), nil)
		return
	}

	response.SendAPIResponse(c, http.StatusOK, true, "startup updated", startup)
}

// @Summary      Delete a startup
// @Description  Deletes a startup by ID
// @Tags         startups
// @Produce      json
// @Param        id   path      int  true  "Startup ID"
// @Success      200  {object}  response.APIResponse "Startup deleted successfully"
// @Failure      400  {object}  response.APIResponse "Invalid startup ID"
// @Failure      404  {object}  response.APIResponse "Startup not found"
// @Failure      500  {object}  response.APIResponse "Internal server error"
// @Router       /startups/{id} [delete]
func (h *StartupHandler) deleteStartup(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.SendAPIResponse(c, http.StatusBadRequest, false, "invalid startup id", nil)
		return
	}

	if err := h.service.DeleteStartup(c.Request.Context(), id); err != nil {
		if err == ErrStartupNotFound {
			response.SendAPIResponse(c, http.StatusNotFound, false, "startup not found", nil)
			return
		}
		response.SendAPIResponse(c, http.StatusInternalServerError, false, err.Error(), nil)
		return
	}

	response.SendAPIResponse(c, http.StatusOK, true, "startup deleted", nil)
}

// @Summary      Get startup by ID
// @Description  Retrieves a single startup by its ID
// @Tags         startups
// @Produce      json
// @Param        id   path      int  true  "Startup ID"
// @Success      200  {object}  response.APIResponse{data=Startup} "Startup retrieved successfully"
// @Failure      400  {object}  response.APIResponse "Invalid startup ID"
// @Failure      404  {object}  response.APIResponse "Startup not found"
// @Failure      500  {object}  response.APIResponse "Internal server error"
// @Router       /startups/{id} [get]
func (h *StartupHandler) getStartupByID(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.SendAPIResponse(c, http.StatusBadRequest, false, "invalid startup id", nil)
		return
	}

	startup, err := h.service.GetStartupByID(c.Request.Context(), id)
	if err != nil {
		if err == ErrStartupNotFound {
			response.SendAPIResponse(c, http.StatusNotFound, false, "startup not found", nil)
			return
		}
		response.SendAPIResponse(c, http.StatusInternalServerError, false, err.Error(), nil)
		return
	}

	response.SendAPIResponse(c, http.StatusOK, true, "startup fetched", startup)
}

// @Summary      List all startups
// @Description  Retrieves a paginated list of all startups
// @Tags         startups
// @Produce      json
// @Param        page   query     int  false  "Page number" default(1)
// @Param        limit  query     int  false  "Items per page" default(10)
// @Success      200  {object}  response.APIResponse{data=StartupList} "Startups retrieved successfully"
// @Failure      500  {object}  response.APIResponse "Internal server error"
// @Router       /startups [get]
func (h *StartupHandler) listStartups(c *gin.Context) {
	page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
	if err != nil || page < 1 {
		page = 1
	}

	limit, err := strconv.Atoi(c.DefaultQuery("limit", "10"))
	if err != nil || limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}

	startupsList, total, err := h.service.ListStartups(c.Request.Context(), page, limit)
	if err != nil {
		response.SendAPIResponse(c, http.StatusInternalServerError, false, err.Error(), nil)
		return
	}

	data := StartupList{Items: startupsList, Total: total, Page: page, Limit: limit}
	response.SendAPIResponse(c, http.StatusOK, true, "startups listed", data)
}

// @Summary      Delete all startups
// @Description  Soft deletes all startups by setting is_deleted to true
// @Tags         startups
// @Produce      json
// @Success      200  {object}  response.APIResponse "All startups deleted successfully"
// @Failure      500  {object}  response.APIResponse "Internal server error"
// @Router       /startups [delete]
func (h *StartupHandler) deleteAllStartups(c *gin.Context) {
	if err := h.service.DeleteAllStartups(c.Request.Context()); err != nil {
		response.SendAPIResponse(c, http.StatusInternalServerError, false, err.Error(), nil)
		return
	}

	response.SendAPIResponse(c, http.StatusOK, true, "all startups deleted", nil)
}

// @Summary      Get startups by UUID
// @Description  Retrieves startups by user's UUID
// @Tags         startups
// @Produce      json
// @Param        uuid   path      string  true  "user UUID"
// @Success      200  {object}  response.APIResponse{data=StartupList} "Startups retrieved successfully"
// @Failure      500  {object}  response.APIResponse "Internal server error"
// @Router       /startups/user/{uuid} [get]
func (h *StartupHandler) ListStartupsByUser(c *gin.Context) {
	uuid := c.Param("uuid")

	startups, err := h.service.ListStartupsByUser(c.Request.Context(), uuid)
	if err != nil {
		response.SendAPIResponse(c, http.StatusInternalServerError, false, err.Error(), nil)
		return
	}

	StartupList := StartupList{Items: startups, Total: int64(len(startups))}
	response.SendAPIResponse(c, http.StatusOK, true, "startup fetched by uuid", StartupList)
}
