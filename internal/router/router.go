package router

import (
	"embed"
	"io/fs"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jacinli/sky-guardwall/internal/config"
	"github.com/jacinli/sky-guardwall/internal/handler"
	"github.com/jacinli/sky-guardwall/internal/middleware"
	"github.com/jacinli/sky-guardwall/internal/service"
	"gorm.io/gorm"
)

func Setup(cfg *config.Config, db *gorm.DB, frontendFS embed.FS) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())

	// Services
	iptablesSvc := service.NewIptablesService(db)
	ruleSvc := service.NewRuleService(db)

	// Handlers
	authH := handler.NewAuthHandler(cfg)
	iptablesH := handler.NewIptablesHandler(iptablesSvc)
	ruleH := handler.NewRuleHandler(ruleSvc)

	// API routes
	api := r.Group("/api/v1")
	{
		api.GET("/health", handler.Health)
		api.POST("/auth/login", authH.Login)

		protected := api.Group("", middleware.JWTAuth(cfg.JWTSecret))
		{
			protected.GET("/iptables/rules", iptablesH.GetRules)
			protected.POST("/iptables/sync", iptablesH.TriggerSync)
			protected.GET("/managed-rules", ruleH.ListRules)
			protected.POST("/managed-rules", ruleH.AddRule)
			protected.DELETE("/managed-rules/:id", ruleH.DeleteRule)
		}
	}

	// Serve embedded React SPA for all other routes
	// The embed package uses //go:embed dist, so sub-path is "dist"
	distFS, err := fs.Sub(frontendFS, "dist")
	if err != nil {
		panic("failed to sub frontend/dist from embed: " + err.Error())
	}
	fileServer := http.FileServer(http.FS(distFS))

	r.NoRoute(func(c *gin.Context) {
		// Try serving static file; fall back to index.html for SPA routing
		path := c.Request.URL.Path
		if _, err := fs.Stat(distFS, path[1:]); err == nil {
			fileServer.ServeHTTP(c.Writer, c.Request)
			return
		}
		// Serve index.html for all unknown paths (React Router handles them)
		index, err := fs.ReadFile(distFS, "index.html")
		if err != nil {
			c.Status(http.StatusNotFound)
			return
		}
		c.Data(http.StatusOK, "text/html; charset=utf-8", index)
	})

	return r
}
