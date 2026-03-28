package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jacinli/sky-guardwall/internal/response"
	"github.com/jacinli/sky-guardwall/internal/service"
)

type IptablesHandler struct {
	svc *service.IptablesService
}

func NewIptablesHandler(svc *service.IptablesService) *IptablesHandler {
	return &IptablesHandler{svc: svc}
}

func (h *IptablesHandler) GetRules(c *gin.Context) {
	chain := c.Query("chain")
	rules, err := h.svc.GetRules(chain)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	chains, _ := h.svc.GetChains()
	meta, _ := h.svc.LastSync()

	result := gin.H{
		"rules":  rules,
		"total":  len(rules),
		"chains": chains,
	}
	if meta != nil {
		result["last_synced_at"] = meta.SyncedAt
		result["sync_has_error"] = meta.HasError
		result["sync_error_msg"] = meta.ErrorMsg
	}

	response.Success(c, result)
}

func (h *IptablesHandler) TriggerSync(c *gin.Context) {
	result := h.svc.Sync(c.Request.Context())
	if result == nil {
		response.Error(c, http.StatusTooManyRequests, "sync already in progress")
		return
	}
	if result.HasError {
		response.Error(c, http.StatusInternalServerError, result.ErrorMsg)
		return
	}
	response.Success(c, result)
}
