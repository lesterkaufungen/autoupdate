package autoupdate

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/labstack/echo/v4"
	"github.com/lesterkaufungen/autoupdate/server"
)

// Gin registers the autoupdate server handlers on a Gin router.
func Gin(r gin.IRouter, config ...server.Config) (*server.Server, error) {
	return server.Gin(r, config...)
}

// Echo registers the autoupdate server handlers on an Echo router.
func Echo(e *echo.Group, config ...server.Config) (*server.Server, error) {
	return server.Echo(e, config...)
}

// HTTP registers the autoupdate server handlers on a standard http.ServeMux.
func HTTP(mux *http.ServeMux, config ...server.Config) (*server.Server, error) {
	return server.HTTP(mux, config...)
}
