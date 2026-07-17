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
		Ruta utilizada por Railway
		para verificar que la API está funcionando.
	*/
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

	/*
		Ruta principal opcional.
	*/
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

	leadHandler :=
		handlers.NewLeadHandler(
			dbPool,
		)

	benefitsHandler :=
		handlers.NewBenefitsHandler(
			dbPool,
		)

	api := router.Group("/api")

	routes.RegisterLeadRoutes(
		api,
		leadHandler,
	)

	routes.RegisterBenefitsRoutes(
		api,
		benefitsHandler,
	)

	port := cfg.Port

	if port == "" {
		port = os.Getenv("PORT")
	}

	if port == "" {
		port = "8080"
	}

	address := "0.0.0.0:" + port

	log.Printf(
		"API ANCOSUR ejecutándose en %s",
		address,
	)

	if err := router.Run(address); err != nil {
		log.Fatal(
			"No se pudo iniciar el servidor: ",
			err,
		)
	}
}