package main

import (
	"ancosur-api/config"
	"ancosur-api/handlers"
	"ancosur-api/routes"
	"context"
	"log"

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

	if err := router.Run(
		":" + cfg.Port,
	); err != nil {
		log.Fatal(
			"No se pudo iniciar el servidor: ",
			err,
		)
	}
}