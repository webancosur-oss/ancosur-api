package main

import (
	"ancosur-api/config"
	"ancosur-api/routes"
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	appConfig := config.Load()

	db, err := connectDatabase(appConfig.DatabaseURL)
	if err != nil {
		log.Fatal("Error conectando a PostgreSQL: ", err)
	}

	defer db.Close()

	router := gin.Default()

	enableCORS(router, appConfig.FrontendURLs)

	router.GET("/", HomeHandler)
	router.GET("/health", HealthHandler(db))

	router.Static("/public", "./public")

	api := router.Group("/api")

	// Rutas públicas y protegidas manejadas dentro de cada archivo
	routes.RutasAuth(api, db)
	routes.RutasLeads(api, db)
	routes.RutasUsers(api, db)
	routes.RutasAsesores(api, db)
	routes.RutasEstadoLeads(api, db)

	// routes.RutasProyectos(api, db)
	// routes.RutasCampanias(api, db)
	// routes.RutasInversiones(api, db)
	// routes.RutasTerrenos(api, db)
	// routes.RutasClientes(api, db)

	fmt.Println("Server on port " + appConfig.Port)

	if err := router.Run(":" + appConfig.Port); err != nil {
		log.Fatal("Error iniciando servidor: ", err)
	}
}

func connectDatabase(databaseURL string) (*pgxpool.Pool, error) {
	if databaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL no está configurada ni se pudo generar con DB_HOST, DB_PORT, DB_NAME, DB_USER y DB_PASSWORD")
	}

	configDB, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, err
	}

	configDB.MaxConns = 10
	configDB.MinConns = 1
	configDB.MaxConnLifetime = time.Hour
	configDB.MaxConnIdleTime = time.Minute * 30

	db, err := pgxpool.NewWithConfig(
		context.Background(),
		configDB,
	)

	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(
		context.Background(),
		5*time.Second,
	)

	defer cancel()

	if err := db.Ping(ctx); err != nil {
		db.Close()
		return nil, err
	}

	return db, nil
}

func HomeHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"api":     "API ANCOSUR Dashboard",
		"version": "1.0.0",
		"message": "Servicio disponible.",
	})
}

func HealthHandler(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(
			c.Request.Context(),
			3*time.Second,
		)

		defer cancel()

		if err := db.Ping(ctx); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"success": false,
				"api":     "API ANCOSUR Dashboard",
				"db":      "error",
				"message": "No hay conexión con la base de datos.",
				"error":   err.Error(),
			})

			return
		}

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"api":     "API ANCOSUR Dashboard",
			"db":      "ok",
			"message": "API y base de datos disponibles.",
		})
	}
}

func enableCORS(
	router *gin.Engine,
	frontendURLs []string,
) {
	router.Use(func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		if origin == "" {
			origin = "http://localhost:3000"
		}

		allowedOrigin := ""

		for _, allowed := range frontendURLs {
			if origin == allowed {
				allowedOrigin = origin
				break
			}
		}

		if allowedOrigin == "" && len(frontendURLs) > 0 {
			allowedOrigin = frontendURLs[0]
		}

		if allowedOrigin == "" {
			allowedOrigin = "http://localhost:3000"
		}

		c.Writer.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, Access-Token")

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	})
}