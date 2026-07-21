package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	Rol    string `json:"rol"`
	AsesorID string `json:"asesor_id"`

	jwt.RegisteredClaims
}

func GenerateToken(
	userID string,
	email string,
	rol string,
	asesorID string,
) (string, error) {
	key, err := JWTKey()

	if err != nil {
		return "", err
	}

	now := time.Now()

	claims := Claims{
		UserID:   userID,
		Email:    email,
		Rol:      rol,
		AsesorID: asesorID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(
				now.Add(
					time.Hour * time.Duration(
						JWTExpiresHours(),
					),
				),
			),
			IssuedAt: jwt.NewNumericDate(now),
			Issuer:   JWTIssuer(),
			Subject:  userID,
		},
	}

	token := jwt.NewWithClaims(
		jwt.SigningMethodHS256,
		claims,
	)

	return token.SignedString(key)
}

func ValidateToken(
	tokenString string,
) (*Claims, error) {
	key, err := JWTKey()

	if err != nil {
		return nil, err
	}

	claims := &Claims{}

	token, err := jwt.ParseWithClaims(
		tokenString,
		claims,
		func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, errors.New("método de firma inválido")
			}

			return key, nil
		},
	)

	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, errors.New("token inválido")
	}

	return claims, nil
}