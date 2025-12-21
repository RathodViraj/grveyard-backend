package assets

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"grveyard/pkg/response"
)

type AssetHandler struct {
	service AssetService
}

func NewAssetHandler(service AssetService) *AssetHandler {
	return &AssetHandler{service: service}
}

func isValidAssetType(assetType string) bool {
	switch assetType {
	case "research", "codebase", "domain", "product", "data", "other":
		return true
	default:
		return false
	}
}

func (h *AssetHandler) RegisterRoutes(router *gin.Engine) {
	router.POST("/assets", h.createAsset)
	router.PUT("/assets/:id", h.updateAsset)
	router.DELETE("/assets/:id", h.deleteAsset)
	router.DELETE("/assets", h.deleteAllAssets)
	router.GET("/assets", h.listAssets)
	router.GET("/assets/:id", h.getAssetByID)
	router.GET("/users/:uuid/assets", h.listAssetsByUser)
	router.DELETE("/users/:uuid/assets/delete-all", h.deleteAllAssetsByUserUUID)
}

type createAssetRequest struct {
	UserUUID     string  `json:"user_uuid" binding:"required"`
	Title        string  `json:"title" binding:"required"`
	Description  string  `json:"description"`
	AssetType    string  `json:"asset_type" binding:"required"`
	ImageURL     string  `json:"image_url"`
	Price        float64 `json:"price"`
	IsNegotiable bool    `json:"is_negotiable"`
	IsSold       bool    `json:"is_sold"`
}

type updateAssetRequest struct {
	Title        string  `json:"title" binding:"required"`
	Description  string  `json:"description"`
	AssetType    string  `json:"asset_type" binding:"required"`
	ImageURL     string  `json:"image_url"`
	Price        float64 `json:"price"`
	IsNegotiable bool    `json:"is_negotiable"`
	IsSold       bool    `json:"is_sold"`
}

// @Summary      Create a new asset
// @Description  Creates a new asset for sale under a startup
// @Tags         assets
// @Accept       json
// @Produce      json
// @Param        request body createAssetRequest true "Asset creation request"
// @Success      201  {object}  response.APIResponse{data=Asset} "Asset created successfully"
// @Failure      400  {object}  response.APIResponse "Invalid request payload"
// @Failure      500  {object}  response.APIResponse "Internal server error"
// @Router       /assets [post]
func (h *AssetHandler) createAsset(c *gin.Context) {
	var req createAssetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.SendAPIResponse(c, http.StatusBadRequest, false, "invalid request payload", nil)
		return
	}

	if req.UserUUID == "" {
		response.SendAPIResponse(c, http.StatusBadRequest, false, "user_uuid must be provided", nil)
		return
	}

	if !isValidAssetType(req.AssetType) {
		response.SendAPIResponse(c, http.StatusBadRequest, false, "invalid asset_type", nil)
		return
	}

	if req.Price < 0 {
		response.SendAPIResponse(c, http.StatusBadRequest, false, "price cannot be negative", nil)
		return
	}

	asset, err := h.service.CreateAsset(c.Request.Context(), Asset{
		UserUUID:     req.UserUUID,
		Title:        req.Title,
		Description:  req.Description,
		AssetType:    req.AssetType,
		ImageURL:     req.ImageURL,
		Price:        req.Price,
		IsNegotiable: req.IsNegotiable,
		IsSold:       req.IsSold,
		IsActive:     true,
	})
	if err != nil {
		response.SendAPIResponse(c, http.StatusInternalServerError, false, err.Error(), nil)
		return
	}

	response.SendAPIResponse(c, http.StatusCreated, true, "asset created", asset)
}

// @Summary      Update an asset
// @Description  Updates an existing asset's details
// @Tags         assets
// @Accept       json
// @Produce      json
// @Param        id   path      int  true  "Asset ID"
// @Param        request body updateAssetRequest true "Asset update request"
// @Success      200  {object}  response.APIResponse{data=Asset} "Asset updated successfully"
// @Failure      400  {object}  response.APIResponse "Invalid request"
// @Failure      404  {object}  response.APIResponse "Asset not found"
// @Failure      500  {object}  response.APIResponse "Internal server error"
// @Router       /assets/{id} [put]
func (h *AssetHandler) updateAsset(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.SendAPIResponse(c, http.StatusBadRequest, false, "invalid asset id", nil)
		return
	}

	var req updateAssetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.SendAPIResponse(c, http.StatusBadRequest, false, "invalid request payload", nil)
		return
	}

	if !isValidAssetType(req.AssetType) {
		response.SendAPIResponse(c, http.StatusBadRequest, false, "invalid asset_type", nil)
		return
	}

	if req.Price < 0 {
		response.SendAPIResponse(c, http.StatusBadRequest, false, "price cannot be negative", nil)
		return
	}

	asset, err := h.service.UpdateAsset(c.Request.Context(), Asset{
		ID:           id,
		Title:        req.Title,
		Description:  req.Description,
		AssetType:    req.AssetType,
		ImageURL:     req.ImageURL,
		Price:        req.Price,
		IsNegotiable: req.IsNegotiable,
		IsSold:       req.IsSold,
	})
	if err != nil {
		if err == ErrAssetNotFound {
			response.SendAPIResponse(c, http.StatusNotFound, false, "asset not found", nil)
			return
		}
		response.SendAPIResponse(c, http.StatusInternalServerError, false, err.Error(), nil)
		return
	}

	response.SendAPIResponse(c, http.StatusOK, true, "asset updated", asset)
}

// @Summary      Delete an asset
// @Description  Deletes an asset by ID
// @Tags         assets
// @Produce      json
// @Param        id   path      int  true  "Asset ID"
// @Success      200  {object}  response.APIResponse "Asset deleted successfully"
// @Failure      400  {object}  response.APIResponse "Invalid asset ID"
// @Failure      404  {object}  response.APIResponse "Asset not found"
// @Failure      500  {object}  response.APIResponse "Internal server error"
// @Router       /assets/{id} [delete]
func (h *AssetHandler) deleteAsset(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.SendAPIResponse(c, http.StatusBadRequest, false, "invalid asset id", nil)
		return
	}

	if err := h.service.DeleteAsset(c.Request.Context(), id); err != nil {
		if err == ErrAssetNotFound {
			response.SendAPIResponse(c, http.StatusNotFound, false, "asset not found", nil)
			return
		}
		response.SendAPIResponse(c, http.StatusInternalServerError, false, err.Error(), nil)
		return
	}

	response.SendAPIResponse(c, http.StatusOK, true, "asset deleted", nil)
}

// @Summary      Get asset by ID
// @Description  Retrieves a single asset by its ID
// @Tags         assets
// @Produce      json
// @Param        id   path      int  true  "Asset ID"
// @Success      200  {object}  response.APIResponse{data=Asset} "Asset retrieved successfully"
// @Failure      400  {object}  response.APIResponse "Invalid asset ID"
// @Failure      404  {object}  response.APIResponse "Asset not found"
// @Failure      500  {object}  response.APIResponse "Internal server error"
// @Router       /assets/{id} [get]
func (h *AssetHandler) getAssetByID(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.SendAPIResponse(c, http.StatusBadRequest, false, "invalid asset id", nil)
		return
	}

	asset, err := h.service.GetAssetByID(c.Request.Context(), id)
	if err != nil {
		if err == ErrAssetNotFound {
			response.SendAPIResponse(c, http.StatusNotFound, false, "asset not found", nil)
			return
		}
		response.SendAPIResponse(c, http.StatusInternalServerError, false, err.Error(), nil)
		return
	}

	response.SendAPIResponse(c, http.StatusOK, true, "asset fetched", asset)
}

// @Summary      List all assets
// @Description  Retrieves a paginated list of active assets with optional filters
// @Tags         assets
// @Produce      json
// @Param        page        query     int     false  "Page number" default(1)
// @Param        limit       query     int     false  "Items per page" default(10)
// @Param        user_uuid   query     string  false  "Filter by user UUID"
// @Param        asset_type  query     string  false  "Filter by asset type" Enums(research, codebase, domain, product, data, other)
// @Param        is_sold     query     bool    false  "Filter by sold status"
// @Success      200  {object}  response.APIResponse{data=AssetList} "Assets retrieved successfully"
// @Failure      500  {object}  response.APIResponse "Internal server error"
// @Router       /assets [get]
func (h *AssetHandler) listAssets(c *gin.Context) {
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

	filters := AssetFilters{}

	if userUUID := c.Query("user_uuid"); userUUID != "" {
		filters.UserUUID = &userUUID
	}

	if assetType := c.Query("asset_type"); assetType != "" {
		if isValidAssetType(assetType) {
			filters.AssetType = &assetType
		}
	}

	if isSoldStr := c.Query("is_sold"); isSoldStr != "" {
		isSold, err := strconv.ParseBool(isSoldStr)
		if err == nil {
			filters.IsSold = &isSold
		}
	}

	assetsList, total, err := h.service.ListAssets(c.Request.Context(), filters, page, limit)
	if err != nil {
		response.SendAPIResponse(c, http.StatusInternalServerError, false, err.Error(), nil)
		return
	}

	data := AssetList{Items: assetsList, Total: total, Page: page, Limit: limit}
	response.SendAPIResponse(c, http.StatusOK, true, "assets listed", data)
}

// @Summary      List assets by user
// @Description  Retrieves a paginated list of active assets for a specific user
// @Tags         assets
// @Produce      json
// @Param        uuid   path      string  true   "User UUID"
// @Param        page   query     int  false  "Page number" default(1)
// @Param        limit  query     int  false  "Items per page" default(10)
// @Success      200  {object}  response.APIResponse{data=AssetList} "User assets retrieved successfully"
// @Failure      400  {object}  response.APIResponse "Invalid user UUID"
// @Failure      500  {object}  response.APIResponse "Internal server error"
// @Router       /users/{uuid}/assets [get]
func (h *AssetHandler) listAssetsByUser(c *gin.Context) {
	userUUID := c.Param("uuid")
	if userUUID == "" {
		response.SendAPIResponse(c, http.StatusBadRequest, false, "invalid user uuid", nil)
		return
	}

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

	assetsList, total, err := h.service.ListAssetsByUser(c.Request.Context(), userUUID, page, limit)
	if err != nil {
		response.SendAPIResponse(c, http.StatusInternalServerError, false, err.Error(), nil)
		return
	}

	data := AssetList{Items: assetsList, Total: total, Page: page, Limit: limit}
	response.SendAPIResponse(c, http.StatusOK, true, "startup assets listed", data)
}

// @Summary      Delete all assets
// @Description  Soft deletes all assets by setting is_deleted to true
// @Tags         assets
// @Produce      json
// @Success      200  {object}  response.APIResponse "All assets deleted successfully"
// @Failure      500  {object}  response.APIResponse "Internal server error"
// @Router       /assets [delete]
func (h *AssetHandler) deleteAllAssets(c *gin.Context) {
	if err := h.service.DeleteAllAssets(c.Request.Context()); err != nil {
		response.SendAPIResponse(c, http.StatusInternalServerError, false, err.Error(), nil)
		return
	}

	response.SendAPIResponse(c, http.StatusOK, true, "all assets deleted", nil)
}

// @Summary      Delete all assets by user UUID
// @Description  Soft deletes all assets for a specific user by setting is_deleted to true
// @Tags         assets
// @Produce      json
// @Param        uuid   path      string  true  "User UUID"
// @Success      200  {object}  response.APIResponse "All user assets deleted successfully"
// @Failure      400  {object}  response.APIResponse "Invalid user UUID"
// @Failure      500  {object}  response.APIResponse "Internal server error"
// @Router       /users/{uuid}/assets/delete-all [delete]
func (h *AssetHandler) deleteAllAssetsByUserUUID(c *gin.Context) {
	userUUID := c.Param("uuid")
	if userUUID == "" {
		response.SendAPIResponse(c, http.StatusBadRequest, false, "user uuid required", nil)
		return
	}

	if err := h.service.DeleteAllAssetsByUserUUID(c.Request.Context(), userUUID); err != nil {
		response.SendAPIResponse(c, http.StatusInternalServerError, false, err.Error(), nil)
		return
	}

	response.SendAPIResponse(c, http.StatusOK, true, "all user assets deleted", nil)
}
