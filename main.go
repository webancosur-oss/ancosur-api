package main

import (
	"ancosur-api/config"
	"ancosur-api/handlers"
	"ancosur-api/routes"
	"context"
	"log"
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	cfg := config.Load()

	if cfg.DatabaseURL == "" {
		log.Fatal(
			"DATABASE_URL no está configurada. " +
				"Configúrala en Railway usando " +
				"${{Postgres.DATABASE_URL}}",
		)
	}

	dbContext, cancel := context.WithTimeout(
		context.Background(),
		15*time.Second,
	)
	defer cancel()

	db, err := pgxpool.New(
		dbContext,
		cfg.DatabaseURL,
	)
	if err != nil {
		log.Fatal(
			"Error configurando la conexión a PostgreSQL: ",
			err,
		)
	}
	defer db.Close()

	if err := db.Ping(dbContext); err != nil {
		log.Fatal(
			"No se pudo conectar a PostgreSQL: ",
			err,
		)
	}

	router := gin.Default()

	router.Use(cors.New(cors.Config{
		AllowOrigins: cfg.FrontendURLs,

		AllowMethods: []string{
			http.MethodGet,
			http.MethodPost,
			http.MethodPut,
			http.MethodPatch,
			http.MethodDelete,
			http.MethodOptions,
		},

		AllowHeaders: []string{
			"Origin",
			"Content-Type",
			"Authorization",
			"Accept",
		},

		ExposeHeaders: []string{
			"Content-Length",
		},

		AllowCredentials: true,

		MaxAge: 12 * time.Hour,
	}))

	router.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "API de ANCOSUR funcionando correctamente",
			"app_url": cfg.AppURL,
		})
	})

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "ok",
			"service": "ancosur-api",
		})
	})

	leadHandler := handlers.NewLeadHandler(db)

	api := router.Group("/api")
	{
		routes.RegisterLeadRoutes(
			api,
			leadHandler,
		)
	}

	log.Printf(
		"API de ANCOSUR iniciada en el puerto %s",
		cfg.Port,
	)

	log.Printf(
		"URL pública de la API: %s",
		cfg.AppURL,
	)

	log.Printf(
		"Orígenes permitidos: %v",
		cfg.FrontendURLs,
	)

	if err := router.Run(":" + cfg.Port); err != nil {
		log.Fatal(
			"Error iniciando el servidor: ",
			err,
		)
	}
}
