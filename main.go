package main

import (
	"ancosur-api/config"
	"ancosur-api/handlers"
	"ancosur-api/routes"
	"context"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	// Carga las variables de entorno.
	cfg := config.Load()

	// Verifica la configuración de PostgreSQL.
	if cfg.DatabaseURL == "" {
		log.Fatal(
			"DATABASE_URL no está configurada.",
		)
	}

	// Verifica la clave utilizada para firmar JWT.
	if cfg.JWTSecret == "" {
		log.Fatal(
			"JWT_SECRET no está configurada.",
		)
	}

	ctx := context.Background()

	// Crea el pool de conexiones.
	dbPool, err := pgxpool.New(
		ctx,
		cfg.DatabaseURL,
	)
	if err != nil {
		log.Fatal(
			"No se pudo conectar a PostgreSQL: ",
			err,
		)
	}

	defer dbPool.Close()

	// Comprueba que PostgreSQL esté disponible.
	if err := dbPool.Ping(ctx); err != nil {
		log.Fatal(
			"No se pudo verificar la conexión con PostgreSQL: ",
			err,
		)
	}

	router := gin.Default()

	// Evita usar proxies de confianza no configurados.
	if err := router.SetTrustedProxies(nil); err != nil {
		log.Fatal(
			"No se pudieron configurar los proxies: ",
			err,
		)
	}

	// Estado de la API.
	router.GET(
		"/health",
		func(c *gin.Context) {
			c.JSON(
				http.StatusOK,
				gin.H{
					"status":  "ok",
					"service": "ancosur-api",
				},
			)
		},
	)

	// Ruta principal.
	router.GET(
		"/",
		func(c *gin.Context) {
			c.JSON(
				http.StatusOK,
				gin.H{
					"success": true,
					"message": "API ANCOSUR funcionando.",
				},
			)
		},
	)

	// Inicializa los handlers.
	authHandler := handlers.NewAuthHandler(
		dbPool,
		cfg.JWTSecret,
		cfg.JWTIssuer,
		cfg.JWTExpiresHours,
	)

	leadHandler := handlers.NewLeadHandler(
		dbPool,
	)

	benefitsHandler := handlers.NewBenefitsHandler(
		dbPool,
	)

	investmentHandler := handlers.NewInvestmentHandler(
		dbPool,
	)

	// Grupo principal de la API.
	api := router.Group("/api")

	// Rutas de autenticación.
	routes.RegisterAuthRoutes(
		api,
		authHandler,
	)

	// Rutas de leads.
	routes.RegisterLeadRoutes(
		api,
		leadHandler,
	)

	// Rutas de beneficios.
	routes.RegisterBenefitsRoutes(
		api,
		benefitsHandler,
	)

	// Rutas de solicitudes de inversión.
	routes.RegisterInvestmentRoutes(
		api,
		investmentHandler,
	)

	// Railway asigna PORT automáticamente.
	port := os.Getenv("PORT")

	if port == "" {
		port = cfg.Port
	}

	if port == "" {
		port = "5000"
	}

	address := "0.0.0.0:" + port

	log.Printf(
		"API ANCOSUR ejecutándose en http://localhost:%s",
		port,
	)

	if err := router.Run(address); err != nil {
		log.Fatal(
			"No se pudo iniciar el servidor: ",
			err,
		)
	}
}