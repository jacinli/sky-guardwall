package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type R struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data"`
}

func Success(c *gin.Context, data any) {
	c.JSON(http.StatusOK, R{Code: 0, Message: "success", Data: data})
}

func Error(c *gin.Context, status int, msg string) {
	c.JSON(status, R{Code: status, Message: msg, Data: nil})
}
