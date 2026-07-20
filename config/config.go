package config

import (
	"net"
	"net/url"
	"os"
	"strconv"
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

	JWTSecret       string
	JWTIssuer       string
	JWTExpiresHours int
}

// Load carga las variables locales o las configuradas en Railway.
func Load() Config {
	_ = godotenv.Load()

	port := getEnv(
		"PORT",
		"5000",
	)

	appURL := getAppURL(port)
	frontendURLs := parseFrontendURLs()

	dbHost := strings.TrimSpace(
		os.Getenv("DB_HOST"),
	)

	dbPort := getEnv(
		"DB_PORT",
		"5432",
	)

	dbName := strings.TrimSpace(
		os.Getenv("DB_NAME"),
	)

	dbUser := strings.TrimSpace(
		os.Getenv("DB_USER"),
	)

	dbPassword := os.Getenv(
		"DB_PASSWORD",
	)

	dbSSLMode := getEnv(
		"DB_SSLMODE",
		"require",
	)

	databaseURL := strings.TrimSpace(
		os.Getenv("DATABASE_URL"),
	)

	// Permite conexión local usando variables DB_*.
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

		JWTSecret: strings.TrimSpace(
			os.Getenv("JWT_SECRET"),
		),

		JWTIssuer: getEnv(
			"JWT_ISSUER",
			"ancosur-api",
		),

		JWTExpiresHours: getPositiveIntEnv(
			"JWT_EXPIRES_HOURS",
			8,
		),
	}
}

// getEnv devuelve la variable o un valor predeterminado.
func getEnv(
	key string,
	fallback string,
) string {
	value := strings.TrimSpace(
		os.Getenv(key),
	)

	if value == "" {
		return fallback
	}

	return value
}

// getPositiveIntEnv obtiene un entero positivo del entorno.
func getPositiveIntEnv(
	key string,
	fallback int,
) int {
	value := strings.TrimSpace(
		os.Getenv(key),
	)

	if value == "" {
		return fallback
	}

	parsedValue, err := strconv.Atoi(
		value,
	)

	if err != nil || parsedValue < 1 {
		return fallback
	}

	return parsedValue
}

// getAppURL obtiene la URL pública de la API.
func getAppURL(port string) string {
	appURL := strings.TrimRight(
		strings.TrimSpace(
			os.Getenv("APP_URL"),
		),
		"/",
	)

	if appURL != "" {
		return appURL
	}

	railwayDomain := strings.TrimSpace(
		os.Getenv(
			"RAILWAY_PUBLIC_DOMAIN",
		),
	)

	if railwayDomain != "" {
		return "https://" +
			strings.TrimRight(
				railwayDomain,
				"/",
			)
	}

	return "http://localhost:" + port
}

// parseFrontendURLs carga los dominios permitidos para CORS.
func parseFrontendURLs() []string {
	rawOrigins := strings.TrimSpace(
		os.Getenv("FRONTEND_URLS"),
	)

	// Mantiene compatibilidad con FRONTEND_URL.
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

	parts := strings.Split(
		rawOrigins,
		",",
	)

	origins := make(
		[]string,
		0,
		len(parts),
	)

	for _, part := range parts {
		origin := strings.TrimRight(
			strings.TrimSpace(part),
			"/",
		)

		if origin != "" {
			origins = append(
				origins,
				origin,
			)
		}
	}

	return origins
}

// buildDatabaseURL crea la conexión usando las variables DB_*.
func buildDatabaseURL(
	host string,
	port string,
	name string,
	user string,
	password string,
	sslMode string,
) string {
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

	query.Set(
		"sslmode",
		sslMode,
	)

	connectionURL.RawQuery =
		query.Encode()

	return connectionURL.String()
}
