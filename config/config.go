package config

import (
	"net"
	"net/url"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	Port         string
	AppURL       string
	FrontendURLs []string

	DBHost     string
	DBPort     string
	DBName     string
	DBUser     string
	DBPassword string
	DBSSLMode  string

	DatabaseURL string
}

func Load() Config {
	// Útil en desarrollo local.
	// En Railway las variables se leen directamente del entorno.
	_ = godotenv.Load()

	port := getEnv("PORT", "5000")

	appURL := strings.TrimRight(
		strings.TrimSpace(os.Getenv("APP_URL")),
		"/",
	)

	// Si APP_URL no está configurado, utiliza el dominio
	// público generado automáticamente por Railway.
	if appURL == "" {
		railwayPublicDomain := strings.TrimSpace(
			os.Getenv("RAILWAY_PUBLIC_DOMAIN"),
		)

		if railwayPublicDomain != "" {
			appURL = "https://" + strings.TrimRight(
				railwayPublicDomain,
				"/",
			)
		} else {
			appURL = "http://localhost:" + port
		}
	}

	frontendURLs := parseFrontendURLs()

	dbHost := strings.TrimSpace(os.Getenv("DB_HOST"))
	dbPort := getEnv("DB_PORT", "5432")
	dbName := strings.TrimSpace(os.Getenv("DB_NAME"))
	dbUser := strings.TrimSpace(os.Getenv("DB_USER"))
	dbPassword := os.Getenv("DB_PASSWORD")
	dbSSLMode := getEnv("DB_SSLMODE", "require")

	/*
		En Railway debe utilizarse directamente:

		DATABASE_URL=${{Postgres.DATABASE_URL}}

		Si DATABASE_URL no existe, se intenta construir
		desde las variables DB_* para mantener compatibilidad local.
	*/
	databaseURL := strings.TrimSpace(
		os.Getenv("DATABASE_URL"),
	)

	if databaseURL == "" {
		databaseURL = buildDatabaseURL(
			dbHost,
			dbPort,
			dbName,
			dbUser,
			dbPassword,
			dbSSLMode,
		)
	}

	return Config{
		Port:         port,
		AppURL:       appURL,
		FrontendURLs: frontendURLs,

		DBHost:     dbHost,
		DBPort:     dbPort,
		DBName:     dbName,
		DBUser:     dbUser,
		DBPassword: dbPassword,
		DBSSLMode:  dbSSLMode,

		DatabaseURL: databaseURL,
	}
}

func getEnv(key string, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))

	if value == "" {
		return fallback
	}

	return value
}

func parseFrontendURLs() []string {
	/*
		FRONTEND_URLS permite varios dominios separados
		por comas:

		http://localhost:3000,
		https://ancosur-web-production.up.railway.app,
		https://ancosur.com,
		https://www.ancosur.com
	*/
	rawOrigins := strings.TrimSpace(
		os.Getenv("FRONTEND_URLS"),
	)

	// Compatibilidad con la antigua variable FRONTEND_URL.
	if rawOrigins == "" {
		rawOrigins = strings.TrimSpace(
			os.Getenv("FRONTEND_URL"),
		)
	}

	if rawOrigins == "" {
		return []string{
			"http://localhost:3000",
		}
	}

	parts := strings.Split(rawOrigins, ",")
	origins := make([]string, 0, len(parts))

	for _, part := range parts {
		origin := strings.TrimRight(
			strings.TrimSpace(part),
			"/",
		)

		if origin != "" {
			origins = append(origins, origin)
		}
	}

	return origins
}

func buildDatabaseURL(
	host string,
	port string,
	name string,
	user string,
	password string,
	sslMode string,
) string {
	// Evita construir una URL inválida con datos vacíos.
	if host == "" ||
		name == "" ||
		user == "" ||
		password == "" {
		return ""
	}

	connectionURL := &url.URL{
		Scheme: "postgresql",
		User: url.UserPassword(
			user,
			password,
		),
		Host: net.JoinHostPort(
			host,
			port,
		),
		Path: "/" + name,
	}

	query := connectionURL.Query()
	query.Set("sslmode", sslMode)
	connectionURL.RawQuery = query.Encode()

	return connectionURL.String()
}