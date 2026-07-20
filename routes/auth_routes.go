package routes

import (
	"ancosur-api/handlers"

	"github.com/gin-gonic/gin"
)

// RegisterAuthRoutes registra las rutas públicas de autenticación.
func RegisterAuthRoutes(
	api *gin.RouterGroup,
	authHandler *handlers.AuthHandler,
) {
	auth := api.Group("/auth")
	{
		auth.POST(
			"/login",
			authHandler.Login,
		)
	}
}