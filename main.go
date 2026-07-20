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
	cfg := config.Load()

	ctx := context.Background()

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

	if err := dbPool.Ping(ctx); err != nil {
		log.Fatal(
			"No se pudo verificar la conexión con PostgreSQL: ",
			err,
		)
	}

	router := gin.Default()

	/*
		Evita la advertencia de Gin sobre
		proxies de confianza.
	*/
	if err := router.SetTrustedProxies(nil); err != nil {
		log.Fatal(
			"No se pudieron configurar los proxies: ",
			err,
		)
	}

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

	/*
		Inicialización de handlers.
	*/
	leadHandler := handlers.NewLeadHandler(
		dbPool,
	)

	benefitsHandler := handlers.NewBenefitsHandler(
		dbPool,
	)

	investmentHandler := handlers.NewInvestmentHandler(
		dbPool,
	)

	/*
		Grupo principal de rutas.
	*/
	api := router.Group("/api")

	routes.RegisterLeadRoutes(
		api,
		leadHandler,
	)

	routes.RegisterBenefitsRoutes(
		api,
		benefitsHandler,
	)

	routes.RegisterInvestmentRoutes(
		api,
		investmentHandler,
	)

	/*
		Railway entrega PORT automáticamente.
		En local se utiliza el puerto configurado
		o 5000 como respaldo.
	*/
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