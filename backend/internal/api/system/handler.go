package system

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"wcstransfer/backend/internal/buildinfo"
	"wcstransfer/backend/internal/config"
	"wcstransfer/backend/internal/platform"
)

type Handler struct {
	config       config.Config
	dependencies *platform.Dependencies
}

func NewHandler(cfg config.Config, deps *platform.Dependencies) *Handler {
	return &Handler{
		config:       cfg,
		dependencies: deps,
	}
}

func (h *Handler) Healthz(c *gin.Context) {
	checks := map[string]platform.CheckResult{}
	if h.dependencies != nil {
		checks = h.dependencies.Health(c.Request.Context())
	}

	status := "ok"
	code := http.StatusOK
	for _, check := range checks {
		if check.Status == "down" {
			status = "degraded"
			code = http.StatusServiceUnavailable
			break
		}
	}

	c.JSON(code, gin.H{
		"status":       status,
		"service":      h.config.AppName,
		"environment":  h.config.Env,
		"dependencies": checks,
		"time":         time.Now().UTC().Format(time.RFC3339),
	})
}

func (h *Handler) Version(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"service":     h.config.AppName,
		"environment": h.config.Env,
		"version":     buildinfo.Version,
		"commit":      buildinfo.Commit,
		"build_time":  buildinfo.BuildTime,
	})
}
