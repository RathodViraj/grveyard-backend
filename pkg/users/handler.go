package users

import (
	"net/http"
	"strconv"

	"grveyard/pkg/response"

	"github.com/gin-gonic/gin"
)

type UserHandler struct {
	service UserService
}

func NewUserHandler(service UserService) *UserHandler {
	return &UserHandler{service: service}
}

func (h *UserHandler) RegisterRoutes(router *gin.Engine) {
	router.POST("/users", h.createUser)
	router.POST("/users/login", h.login)
	router.PUT("/users/:id", h.updateUser)
	router.DELETE("/users/:id", h.deleteUser)
	router.GET("/users", h.listUsers)
	router.GET("/users/:id", h.getUserByID)
}

type createUserRequest struct {
	Name          string `json:"name" binding:"required"`
	Email         string `json:"email" binding:"required"`
	Role          string `json:"role" binding:"required"`
	Password      string `json:"password" binding:"required"`
	ProfilePicURL string `json:"profile_pic_url"`
	UUID          string `json:"uuid"`
}

type updateUserRequest struct {
	Name          string `json:"name" binding:"required"`
	Role          string `json:"role"`
	ProfilePicURL string `json:"profile_pic_url"`
	UUID          string `json:"uuid"`
}

type loginRequest struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
	UUID     string `json:"uuid" binding:"required"`
}

// @Summary      Create user
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        request body createUserRequest true "Create user request"
// @Success      201 {object} response.APIResponse{data=User}
// @Failure      400 {object} response.APIResponse
// @Failure      500 {object} response.APIResponse
// @Router       /users [post]
func (h *UserHandler) createUser(c *gin.Context) {
	var req createUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.SendAPIResponse(c, http.StatusBadRequest, false, "invalid request payload", nil)
		return
	}

	u, err := h.service.CreateUser(c.Request.Context(), req.Name, req.Email, req.Role, req.Password, req.ProfilePicURL, req.UUID)
	if err != nil {
		response.SendAPIResponse(c, http.StatusBadRequest, false, err.Error(), nil)
		return
	}
	response.SendAPIResponse(c, http.StatusCreated, true, "user created", u)
}

// @Summary      Update user
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        id path int true "User ID"
// @Param        request body updateUserRequest true "Update user request"
// @Success      200 {object} response.APIResponse{data=User}
// @Failure      400 {object} response.APIResponse
// @Failure      404 {object} response.APIResponse
// @Failure      500 {object} response.APIResponse
// @Router       /users/{id} [put]
func (h *UserHandler) updateUser(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.SendAPIResponse(c, http.StatusBadRequest, false, "invalid user id", nil)
		return
	}

	var req updateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.SendAPIResponse(c, http.StatusBadRequest, false, "invalid request payload", nil)
		return
	}

	u, err := h.service.UpdateUser(c.Request.Context(), User{
		ID:            id,
		Name:          req.Name,
		Role:          req.Role,
		ProfilePicURL: req.ProfilePicURL,
		UUID:          req.UUID,
	})
	if err != nil {
		if err == ErrUserNotFound {
			response.SendAPIResponse(c, http.StatusNotFound, false, "user not found", nil)
			return
		}
		response.SendAPIResponse(c, http.StatusBadRequest, false, err.Error(), nil)
		return
	}
	response.SendAPIResponse(c, http.StatusOK, true, "user updated", u)
}

// @Summary      Delete user
// @Tags         users
// @Produce      json
// @Param        id path int true "User ID"
// @Success      200 {object} response.APIResponse
// @Failure      400 {object} response.APIResponse
// @Failure      404 {object} response.APIResponse
// @Router       /users/{id} [delete]
func (h *UserHandler) deleteUser(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.SendAPIResponse(c, http.StatusBadRequest, false, "invalid user id", nil)
		return
	}

	if err := h.service.DeleteUser(c.Request.Context(), id); err != nil {
		if err == ErrUserNotFound {
			response.SendAPIResponse(c, http.StatusNotFound, false, "user not found", nil)
			return
		}
		response.SendAPIResponse(c, http.StatusBadRequest, false, err.Error(), nil)
		return
	}
	response.SendAPIResponse(c, http.StatusOK, true, "user deleted", nil)
}

// @Summary      Get user by ID
// @Tags         users
// @Produce      json
// @Param        id path int true "User ID"
// @Success      200 {object} response.APIResponse{data=User}
// @Failure      400 {object} response.APIResponse
// @Failure      404 {object} response.APIResponse
// @Router       /users/{id} [get]
func (h *UserHandler) getUserByID(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.SendAPIResponse(c, http.StatusBadRequest, false, "invalid user id", nil)
		return
	}

	u, err := h.service.GetUserByID(c.Request.Context(), id)
	if err != nil {
		if err == ErrUserNotFound {
			response.SendAPIResponse(c, http.StatusNotFound, false, "user not found", nil)
			return
		}
		response.SendAPIResponse(c, http.StatusBadRequest, false, err.Error(), nil)
		return
	}
	response.SendAPIResponse(c, http.StatusOK, true, "user fetched", u)
}

// @Summary      List users
// @Tags         users
// @Produce      json
// @Param        page  query int false "Page number" default(1)
// @Param        limit query int false "Items per page" default(10)
// @Success      200 {object} response.APIResponse{data=UserList}
// @Failure      500 {object} response.APIResponse
// @Router       /users [get]
func (h *UserHandler) listUsers(c *gin.Context) {
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

	items, total, err := h.service.ListUsers(c.Request.Context(), page, limit)
	if err != nil {
		response.SendAPIResponse(c, http.StatusInternalServerError, false, err.Error(), nil)
		return
	}
	data := UserList{Items: items, Total: total, Page: page, Limit: limit}
	response.SendAPIResponse(c, http.StatusOK, true, "users listed", data)
}

// @Summary      Login user (verify and update uuid)
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        request body loginRequest true "Login request with firebase uuid"
// @Success      200 {object} response.APIResponse{data=User}
// @Failure      400 {object} response.APIResponse
// @Failure      401 {object} response.APIResponse
// @Failure      500 {object} response.APIResponse
// @Router       /users/login [post]
func (h *UserHandler) login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.SendAPIResponse(c, http.StatusBadRequest, false, "invalid request payload", nil)
		return
	}
	u, err := h.service.Login(c.Request.Context(), req.Email, req.Password, req.UUID)
	if err != nil {
		if err.Error() == "invalid credentials" {
			response.SendAPIResponse(c, http.StatusUnauthorized, false, err.Error(), nil)
			return
		}
		response.SendAPIResponse(c, http.StatusBadRequest, false, err.Error(), nil)
		return
	}
	response.SendAPIResponse(c, http.StatusOK, true, "login successful", u)
}
