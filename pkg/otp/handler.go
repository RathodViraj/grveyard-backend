package otp

import (
	"net/http"

	"grveyard/pkg/response"

	"github.com/gin-gonic/gin"
)

type OTPHandler struct {
	service OTPService
}

func NewOTPHandler(service OTPService) *OTPHandler {
	return &OTPHandler{service: service}
}

func (h *OTPHandler) RegisterRoutes(router *gin.Engine) {
	router.POST("/getOTP", h.getOTP)
	router.POST("/verifyOTP", h.verifyOTP)
}

type getOTPRequest struct {
	Email string `json:"email" binding:"required,email"`
}

type verifyOTPRequest struct {
	Email string `json:"email" binding:"required,email"`
	Code  string `json:"code" binding:"required"`
}

// @Summary      Generate and send OTP
// @Description  Generate a one-time password and send it to the provided email
// @Tags         OTP
// @Accept       json
// @Produce      json
// @Param        request body getOTPRequest true "Email to send OTP to"
// @Success      200 {object} response.APIResponse
// @Failure      400 {object} response.APIResponse
// @Failure      500 {object} response.APIResponse
// @Router       /getOTP [post]
func (h *OTPHandler) getOTP(c *gin.Context) {
	var req getOTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.SendAPIResponse(c, http.StatusBadRequest, false, "Invalid request: "+err.Error(), nil)
		return
	}

	if err := h.service.GenerateAndSendOTP(c.Request.Context(), req.Email); err != nil {
		response.SendAPIResponse(c, http.StatusInternalServerError, false, "Failed to generate and send OTP: "+err.Error(), nil)
		return
	}

	response.SendAPIResponse(c, http.StatusOK, true, "OTP sent successfully to "+req.Email, nil)
}

// @Summary      Verify OTP
// @Description  Verify the one-time password for the provided email
// @Tags         OTP
// @Accept       json
// @Produce      json
// @Param        request body verifyOTPRequest true "Email and OTP code to verify"
// @Success      200 {object} response.APIResponse
// @Failure      400 {object} response.APIResponse
// @Failure      401 {object} response.APIResponse
// @Router       /verifyOTP [post]
func (h *OTPHandler) verifyOTP(c *gin.Context) {
	var req verifyOTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.SendAPIResponse(c, http.StatusBadRequest, false, "Invalid request: "+err.Error(), nil)
		return
	}

	valid, err := h.service.VerifyOTP(c.Request.Context(), req.Email, req.Code)
	if err != nil {
		response.SendAPIResponse(c, http.StatusUnauthorized, false, "OTP verification failed: "+err.Error(), nil)
		return
	}

	if !valid {
		response.SendAPIResponse(c, http.StatusUnauthorized, false, "Invalid OTP", nil)
		return
	}

	response.SendAPIResponse(c, http.StatusOK, true, "OTP verified successfully", gin.H{"verified": true})
}
