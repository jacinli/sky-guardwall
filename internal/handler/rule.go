package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/jacinli/sky-guardwall/internal/response"
	"github.com/jacinli/sky-guardwall/internal/service"
)

type RuleHandler struct {
	svc *service.RuleService
}

func NewRuleHandler(svc *service.RuleService) *RuleHandler {
	return &RuleHandler{svc: svc}
}

func (h *RuleHandler) ListRules(c *gin.Context) {
	rules, err := h.svc.ListRules()
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, gin.H{"rules": rules, "total": len(rules)})
}

func (h *RuleHandler) AddRule(c *gin.Context) {
	var req service.AddRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request body")
		return
	}

	rule, err := h.svc.AddRule(c.Request.Context(), req)
	if err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	response.Success(c, rule)
}

func (h *RuleHandler) DeleteRule(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid rule id")
		return
	}

	if err := h.svc.DeleteRule(c.Request.Context(), uint(id)); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, gin.H{"deleted": id})
}
