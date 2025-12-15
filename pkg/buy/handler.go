package buy

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"grveyard/pkg/response"
)

type BuyHandler struct {
	service BuyService
}

func NewBuyHandler(service BuyService) *BuyHandler {
	return &BuyHandler{service: service}
}

func (h *BuyHandler) RegisterRoutes(router *gin.Engine) {
	router.PATCH("/assets/:id/mark-sold", h.markAssetSold)
	router.PATCH("/assets/:id/unlist", h.unlistAsset)
	router.PATCH("/startups/:id/mark-sold", h.markStartupSold)
	router.PATCH("/startups/:id/unlist", h.unlistStartup)
}

// @Summary      Mark asset as sold
// @Description  Marks an asset as sold (sets is_sold to true). Fails if asset is already sold or inactive.
// @Tags         buy
// @Produce      json
// @Param        id   path      int  true  "Asset ID"
// @Success      200  {object}  response.APIResponse "Asset marked as sold successfully"
// @Failure      400  {object}  response.APIResponse "Invalid asset ID"
// @Failure      404  {object}  response.APIResponse "Asset not found"
// @Failure      409  {object}  response.APIResponse "Asset already marked as sold"
// @Failure      500  {object}  response.APIResponse "Internal server error"
// @Router       /assets/{id}/mark-sold [patch]
func (h *BuyHandler) markAssetSold(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.SendAPIResponse(c, http.StatusBadRequest, false, "invalid asset id", nil)
		return
	}

	if err := h.service.MarkAssetSold(c.Request.Context(), id); err != nil {
		if err == ErrNotFound {
			response.SendAPIResponse(c, http.StatusNotFound, false, "asset not found", nil)
			return
		}
		if err == ErrAlreadySold {
			response.SendAPIResponse(c, http.StatusConflict, false, "asset already marked as sold", nil)
			return
		}
		response.SendAPIResponse(c, http.StatusInternalServerError, false, err.Error(), nil)
		return
	}

	response.SendAPIResponse(c, http.StatusOK, true, "asset marked as sold", nil)
}

// @Summary      Unlist an asset
// @Description  Soft deletes an asset by setting is_active to false. Asset won't appear in marketplace listings.
// @Tags         buy
// @Produce      json
// @Param        id   path      int  true  "Asset ID"
// @Success      200  {object}  response.APIResponse "Asset unlisted successfully"
// @Failure      400  {object}  response.APIResponse "Invalid asset ID"
// @Failure      404  {object}  response.APIResponse "Asset not found"
// @Failure      500  {object}  response.APIResponse "Internal server error"
// @Router       /assets/{id}/unlist [patch]
func (h *BuyHandler) unlistAsset(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.SendAPIResponse(c, http.StatusBadRequest, false, "invalid asset id", nil)
		return
	}

	if err := h.service.UnlistAsset(c.Request.Context(), id); err != nil {
		if err == ErrNotFound {
			response.SendAPIResponse(c, http.StatusNotFound, false, "asset not found", nil)
			return
		}
		response.SendAPIResponse(c, http.StatusInternalServerError, false, err.Error(), nil)
		return
	}

	response.SendAPIResponse(c, http.StatusOK, true, "asset unlisted", nil)
}

// @Summary      Mark startup as sold
// @Description  Marks a startup as sold (sets status to 'sold'). Fails if startup is already sold.
// @Tags         buy
// @Produce      json
// @Param        id   path      int  true  "Startup ID"
// @Success      200  {object}  response.APIResponse "Startup marked as sold successfully"
// @Failure      400  {object}  response.APIResponse "Invalid startup ID"
// @Failure      404  {object}  response.APIResponse "Startup not found"
// @Failure      409  {object}  response.APIResponse "Startup already marked as sold"
// @Failure      500  {object}  response.APIResponse "Internal server error"
// @Router       /startups/{id}/mark-sold [patch]
func (h *BuyHandler) markStartupSold(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.SendAPIResponse(c, http.StatusBadRequest, false, "invalid startup id", nil)
		return
	}

	if err := h.service.MarkStartupSold(c.Request.Context(), id); err != nil {
		if err == ErrNotFound {
			response.SendAPIResponse(c, http.StatusNotFound, false, "startup not found", nil)
			return
		}
		if err == ErrAlreadySold {
			response.SendAPIResponse(c, http.StatusConflict, false, "startup already marked as sold", nil)
			return
		}
		response.SendAPIResponse(c, http.StatusInternalServerError, false, err.Error(), nil)
		return
	}

	response.SendAPIResponse(c, http.StatusOK, true, "startup marked as sold", nil)
}

// @Summary      Unlist a startup
// @Description  Unlists a startup by setting status to 'failed'. Startup won't be prominently displayed.
// @Tags         buy
// @Produce      json
// @Param        id   path      int  true  "Startup ID"
// @Success      200  {object}  response.APIResponse "Startup unlisted successfully"
// @Failure      400  {object}  response.APIResponse "Invalid startup ID"
// @Failure      404  {object}  response.APIResponse "Startup not found"
// @Failure      500  {object}  response.APIResponse "Internal server error"
// @Router       /startups/{id}/unlist [patch]
func (h *BuyHandler) unlistStartup(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.SendAPIResponse(c, http.StatusBadRequest, false, "invalid startup id", nil)
		return
	}

	if err := h.service.UnlistStartup(c.Request.Context(), id); err != nil {
		if err == ErrNotFound {
			response.SendAPIResponse(c, http.StatusNotFound, false, "startup not found", nil)
			return
		}
		response.SendAPIResponse(c, http.StatusInternalServerError, false, err.Error(), nil)
		return
	}

	response.SendAPIResponse(c, http.StatusOK, true, "startup unlisted", nil)
}
