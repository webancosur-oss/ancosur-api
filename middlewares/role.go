package middleware

import (
	"ancosur-api/controller"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func RoleMiddleware(
	allowedRoles ...string,
) gin.HandlerFunc {
	return func(c *gin.Context) {
		userRol := strings.ToLower(
			strings.TrimSpace(
				c.GetString("user_rol"),
			),
		)

		if userRol == "" {
			controller.AbortError(
				c,
				http.StatusForbidden,
				"auth.role_required",
				"Rol de usuario requerido.",
				nil,
			)
			return
		}

		for _, allowedRol := range allowedRoles {
			allowedRol = strings.ToLower(
				strings.TrimSpace(allowedRol),
			)

			if userRol == allowedRol {
				c.Next()
				return
			}
		}

		controller.AbortError(
			c,
			http.StatusForbidden,
			"auth.forbidden",
			"No tienes permisos para realizar esta acción.",
			nil,
		)
	}
}