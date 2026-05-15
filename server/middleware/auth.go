package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

type contextKey string

const (
	AppKey  contextKey = "app"
	UserKey contextKey = "user"
)

// AppAuth is a standard http middleware for public application access by ID.
func AppAuth(db *gorm.DB) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// In public mode, we just need the app_id from the URL or query.
			// The actual handlers often get it from the URL pattern /update/:app_id...
			// But for consistent context, we can try to find the app if ID is provided.
			// However, most public endpoints don't need this middleware if they are truly public.
			// For now, we'll keep it as a placeholder that always passes.
			next.ServeHTTP(w, r)
		})
	}
}

// GinAppAuth is a Gin middleware for public application access.
func GinAppAuth(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
	}
}

// EchoAppAuth is an Echo middleware for public application access.
func EchoAppAuth(db *gorm.DB) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			return next(c)
		}
	}
}

// UserAuth is a standard http middleware for user authentication (Admin API).
func UserAuth(db *gorm.DB, adminToken string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := r.Header.Get("X-Admin-Token")
			if token == "" {
				token = r.URL.Query().Get("token")
			}

			if token == "" || token != adminToken {
				http.Error(w, "unauthorized: invalid admin token", http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// GinUserAuth is a Gin middleware for user authentication.
func GinUserAuth(adminToken string) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.GetHeader("X-Admin-Token")
		if token == "" {
			token = c.Query("token")
		}

		if token == "" || token != adminToken {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid admin token"})
			return
		}
		c.Next()
	}
}

// EchoUserAuth is an Echo middleware for user authentication.
func EchoUserAuth(adminToken string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			token := c.Request().Header.Get("X-Admin-Token")
			if token == "" {
				token = c.QueryParam("token")
			}

			if token == "" || token != adminToken {
				return echo.NewHTTPError(http.StatusUnauthorized, "invalid admin token")
			}
			return next(c)
		}
	}
}
