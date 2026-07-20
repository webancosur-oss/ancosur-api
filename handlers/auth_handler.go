package handlers

import (
	"errors"
	"net/http"
	"net/mail"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

type AuthHandler struct {
	DB              *pgxpool.Pool
	JWTSecret       string
	JWTIssuer       string
	JWTExpiresHours int
}

func NewAuthHandler(
	db *pgxpool.Pool,
	jwtSecret string,
	jwtIssuer string,
	jwtExpiresHours int,
) *AuthHandler {
	return &AuthHandler{
		DB:              db,
		JWTSecret:       jwtSecret,
		JWTIssuer:       jwtIssuer,
		JWTExpiresHours: jwtExpiresHours,
	}
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type AuthUserResponse struct {
	ID     string `json:"id"`
	Nombre string `json:"nombre"`
	Email  string `json:"email"`
	Rol    string `json:"rol"`
}

type authClaims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	Rol    string `json:"rol"`

	jwt.RegisteredClaims
}

// authResponse devuelve respuestas con una estructura uniforme.
func authResponse(
	c *gin.Context,
	status int,
	success bool,
	response string,
	message string,
	data any,
) {
	c.JSON(
		status,
		gin.H{
			"success":  success,
			"response": response,
			"message":  message,
			"data":     data,
		},
	)
}

// authValidEmail valida el formato del correo.
func authValidEmail(value string) bool {
	address, err := mail.ParseAddress(value)

	if err != nil {
		return false
	}

	return strings.EqualFold(
		address.Address,
		value,
	)
}

// generateToken crea un JWT firmado con HS256.
func (h *AuthHandler) generateToken(
	user AuthUserResponse,
) (string, int64, error) {
	now := time.Now()

	expiresAt := now.Add(
		time.Duration(
			h.JWTExpiresHours,
		) * time.Hour,
	)

	claims := authClaims{
		UserID: user.ID,
		Email:  user.Email,
		Rol:    user.Rol,

		RegisteredClaims: jwt.RegisteredClaims{
			Issuer: h.JWTIssuer,

			Subject: user.ID,

			ID: uuid.NewString(),

			IssuedAt: jwt.NewNumericDate(
				now,
			),

			NotBefore: jwt.NewNumericDate(
				now,
			),

			ExpiresAt: jwt.NewNumericDate(
				expiresAt,
			),
		},
	}

	token := jwt.NewWithClaims(
		jwt.SigningMethodHS256,
		claims,
	)

	signedToken, err := token.SignedString(
		[]byte(h.JWTSecret),
	)

	if err != nil {
		return "", 0, err
	}

	expiresIn := int64(
		time.Until(expiresAt).Seconds(),
	)

	return signedToken, expiresIn, nil
}

// Login valida las credenciales y devuelve un JWT.
func (h *AuthHandler) Login(
	c *gin.Context,
) {
	var body LoginRequest

	if err := c.ShouldBindJSON(&body); err != nil {
		authResponse(
			c,
			http.StatusBadRequest,
			false,
			"auth.invalid_json",
			"El JSON enviado no es válido.",
			nil,
		)
		return
	}

	email := strings.ToLower(
		strings.TrimSpace(body.Email),
	)

	password := body.Password

	if email == "" {
		authResponse(
			c,
			http.StatusBadRequest,
			false,
			"auth.email_required",
			"El correo electrónico es obligatorio.",
			nil,
		)
		return
	}

	if !authValidEmail(email) {
		authResponse(
			c,
			http.StatusBadRequest,
			false,
			"auth.invalid_email",
			"El correo electrónico no es válido.",
			nil,
		)
		return
	}

	if password == "" {
		authResponse(
			c,
			http.StatusBadRequest,
			false,
			"auth.password_required",
			"La contraseña es obligatoria.",
			nil,
		)
		return
	}

	ctx := c.Request.Context()

	var user AuthUserResponse
	var passwordHash string
	var activo bool

	err := h.DB.QueryRow(
		ctx,
		`
			SELECT
				id::text,
				nombre,
				email,
				password_hash,
				rol,
				activo
			FROM usuarios_dashboard
			WHERE LOWER(email) = LOWER($1)
			LIMIT 1
		`,
		email,
	).Scan(
		&user.ID,
		&user.Nombre,
		&user.Email,
		&passwordHash,
		&user.Rol,
		&activo,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			authResponse(
				c,
				http.StatusUnauthorized,
				false,
				"auth.invalid_credentials",
				"El correo o la contraseña no son correctos.",
				nil,
			)
			return
		}

		authResponse(
			c,
			http.StatusInternalServerError,
			false,
			"auth.user_query_error",
			"No se pudo consultar el usuario.",
			nil,
		)
		return
	}

	if !activo {
		authResponse(
			c,
			http.StatusForbidden,
			false,
			"auth.user_inactive",
			"El usuario se encuentra inactivo.",
			nil,
		)
		return
	}

	err = bcrypt.CompareHashAndPassword(
		[]byte(passwordHash),
		[]byte(password),
	)

	if err != nil {
		authResponse(
			c,
			http.StatusUnauthorized,
			false,
			"auth.invalid_credentials",
			"El correo o la contraseña no son correctos.",
			nil,
		)
		return
	}

	token, expiresIn, err :=
		h.generateToken(user)

	if err != nil {
		authResponse(
			c,
			http.StatusInternalServerError,
			false,
			"auth.token_error",
			"No se pudo generar el token de acceso.",
			nil,
		)
		return
	}

	_, err = h.DB.Exec(
		ctx,
		`
			UPDATE usuarios_dashboard
			SET
				ultimo_acceso = NOW(),
				updated_at = NOW()
			WHERE id = $1::uuid
		`,
		user.ID,
	)

	if err != nil {
		authResponse(
			c,
			http.StatusInternalServerError,
			false,
			"auth.update_access_error",
			"No se pudo actualizar el último acceso.",
			nil,
		)
		return
	}

	authResponse(
		c,
		http.StatusOK,
		true,
		"auth.login_success",
		"Inicio de sesión correcto.",
		gin.H{
			"token":      token,
			"token_type": "Bearer",
			"expires_in": expiresIn,
			"user":       user,
		},
	)
}
