package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/jacinli/sky-guardwall/internal/response"
)

func Health(c *gin.Context) {
	response.Success(c, gin.H{"status": "ok"})
}
