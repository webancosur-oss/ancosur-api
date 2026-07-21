package middleware

import (
	"ancosur-api/auth"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := getTokenFromRequest(c)

		if tokenString == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success":  false,
				"response": "auth.token_required",
				"message":  "Token requerido.",
				"data":     nil,
			})

			c.Abort()
			return
		}

		claims, err := auth.ValidateToken(tokenString)

		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success":  false,
				"response": "auth.invalid_token",
				"message":  "Token inválido o expirado.",
				"data": gin.H{
					"error": err.Error(),
				},
			})

			c.Abort()
			return
		}

		c.Set("user_id", claims.UserID)
		c.Set("user_email", claims.Email)
		c.Set("user_rol", claims.Rol)
		c.Set("user_asesor_id", claims.AsesorID)

		c.Next()
	}
}

func getTokenFromRequest(
	c *gin.Context,
) string {
	authorization := strings.TrimSpace(
		c.GetHeader("Authorization"),
	)

	if authorization != "" {
		parts := strings.SplitN(
			authorization,
			" ",
			2,
		)

		if len(parts) == 2 &&
			strings.EqualFold(parts[0], "Bearer") {
			return strings.TrimSpace(parts[1])
		}
	}

	accessToken := strings.TrimSpace(
		c.GetHeader("Access-Token"),
	)

	if accessToken != "" {
		return accessToken
	}

	return ""
}
