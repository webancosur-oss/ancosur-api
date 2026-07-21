package auth

import (
	"errors"
	"os"
	"strconv"
)

func JWTKey() ([]byte, error) {
	key := os.Getenv("JWT_SECRET")

	if key == "" {
		return nil, errors.New("JWT_SECRET no está configurado")
	}

	return []byte(key), nil
}

func JWTIssuer() string {
	issuer := os.Getenv("JWT_ISSUER")

	if issuer == "" {
		return "ancosur-dashboard"
	}

	return issuer
}

func JWTExpiresHours() int {
	value := os.Getenv("JWT_EXPIRES_HOURS")

	if value == "" {
		return 24
	}

	hours, err := strconv.Atoi(value)

	if err != nil || hours <= 0 {
		return 24
	}

	return hours
}