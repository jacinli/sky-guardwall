package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jacinli/sky-guardwall/internal/config"
	"github.com/jacinli/sky-guardwall/internal/middleware"
	"github.com/jacinli/sky-guardwall/internal/response"
)

type AuthHandler struct {
	cfg *config.Config
}

func NewAuthHandler(cfg *config.Config) *AuthHandler {
	return &AuthHandler{cfg: cfg}
}

type loginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "username and password are required")
		return
	}

	if req.Username != h.cfg.AdminUser || req.Password != h.cfg.AdminPass {
		response.Error(c, http.StatusUnauthorized, "invalid username or password")
		return
	}

	token, err := middleware.GenerateToken(req.Username, h.cfg.JWTSecret, h.cfg.JWTExpireHours)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to generate token")
		return
	}

	response.Success(c, gin.H{"token": token})
}
