package response

import (
	"time"

	"github.com/gin-gonic/gin"
)

type APIResponse struct {
	Success   bool      `json:"success"`
	Message   string    `json:"message"`
	Data      any       `json:"data,omitempty"`
	CreatedAt time.Time `json:"created_at,omitempty"`
}

func SendAPIResponse(c *gin.Context, code int, success bool, message string, data any) {
	resp := APIResponse{
		Success:   success,
		Message:   message,
		Data:      data,
		CreatedAt: time.Now(),
	}

	c.JSON(code, resp)
}
