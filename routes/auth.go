package routes

import (
	"ancosur-api/auth"
	"ancosur-api/controller"
	middleware "ancosur-api/middlewares"
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

type AuthRoutes struct {
	DB *pgxpool.Pool
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type DashboardUser struct {
	ID       string `json:"id"`
	AsesorID string `json:"asesor_id"`
	Nombre   string `json:"nombre"`
	Email    string `json:"email"`
	Rol      string `json:"rol"`
	Activo   bool   `json:"-"`
}

type PublicUser struct {
	Nombre string `json:"nombre"`
	Email  string `json:"email"`
	Rol    string `json:"rol"`
}

func RutasAuth(
	api *gin.RouterGroup,
	db *pgxpool.Pool,
) {
	authRoutes := &AuthRoutes{
		DB: db,
	}

	s := api.Group("/auth")

	s.GET("/", authRoutes.Authentication)
	s.POST("/login", authRoutes.Login)
	s.PUT("/login", authRoutes.Login)

	protected := s.Group("")
	protected.Use(middleware.AuthMiddleware())

	protected.GET("/verify", authRoutes.VerifyLogin)
	protected.GET("/authentication", authRoutes.VerifyLogin)
	protected.POST("/logout", authRoutes.Logout)
}

func (h *AuthRoutes) Authentication(
	c *gin.Context,
) {
	controller.Success(
		c,
		http.StatusOK,
		"auth.available",
		"Servicio de autenticación disponible.",
		gin.H{},
	)
}

func (h *AuthRoutes) Login(
	c *gin.Context,
) {
	var body LoginRequest

	if !controller.CheckBody(c, &body) {
		return
	}

	email := strings.ToLower(
		strings.TrimSpace(body.Email),
	)

	password := strings.TrimSpace(
		body.Password,
	)

	if email == "" || password == "" {
		controller.Error(
			c,
			http.StatusBadRequest,
			"auth.required_fields",
			"El correo y la contraseña son obligatorios.",
			nil,
		)
		return
	}

	user, passwordHash, err := h.findUserByEmail(
		c,
		email,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			controller.Error(
				c,
				http.StatusUnauthorized,
				"auth.invalid_credentials",
				"Usuario y/o contraseña incorrectos.",
				nil,
			)
			return
		}

		controller.Error(
			c,
			http.StatusInternalServerError,
			"auth.login_error",
			"No se pudo iniciar sesión.",
			nil,
		)
		return
	}

	if !user.Activo {
		controller.Error(
			c,
			http.StatusForbidden,
			"auth.user_inactive",
			"El usuario está inactivo.",
			nil,
		)
		return
	}

	err = bcrypt.CompareHashAndPassword(
		[]byte(passwordHash),
		[]byte(password),
	)

	if err != nil {
		controller.Error(
			c,
			http.StatusUnauthorized,
			"auth.invalid_credentials",
			"Usuario y/o contraseña incorrectos.",
			nil,
		)
		return
	}

	token, err := auth.GenerateToken(
		user.ID,
		user.Email,
		user.Rol,
		user.AsesorID,
	)

	if err != nil {
		controller.Error(
			c,
			http.StatusInternalServerError,
			"auth.token_error",
			"No se pudo generar el token.",
			nil,
		)
		return
	}

	_, _ = h.DB.Exec(
		c.Request.Context(),
		`
			UPDATE usuarios_dashboard
			SET
				ultimo_acceso = NOW(),
				updated_at = NOW()
			WHERE id = $1::uuid
		`,
		user.ID,
	)

	controller.Success(
		c,
		http.StatusOK,
		"auth.login",
		"Inicio de sesión correcto.",
		gin.H{
			"user":  toPublicUser(user),
			"token": token,
		},
	)
}

func (h *AuthRoutes) VerifyLogin(
	c *gin.Context,
) {
	userID := strings.TrimSpace(
		c.GetString("user_id"),
	)

	if userID == "" {
		controller.Error(
			c,
			http.StatusUnauthorized,
			"auth.invalid_session",
			"Sesión inválida.",
			nil,
		)
		return
	}

	user, err := h.findUserByID(
		c,
		userID,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			controller.Error(
				c,
				http.StatusUnauthorized,
				"auth.user_not_found",
				"El usuario de la sesión no existe.",
				nil,
			)
			return
		}

		controller.Error(
			c,
			http.StatusInternalServerError,
			"auth.verify_error",
			"No se pudo verificar la sesión.",
			nil,
		)
		return
	}

	if !user.Activo {
		controller.Error(
			c,
			http.StatusForbidden,
			"auth.user_inactive",
			"El usuario está inactivo.",
			nil,
		)
		return
	}

	controller.Success(
		c,
		http.StatusOK,
		"auth.valid",
		"Sesión válida.",
		gin.H{
			"user": toPublicUser(user),
		},
	)
}

func (h *AuthRoutes) Logout(
	c *gin.Context,
) {
	controller.Success(
		c,
		http.StatusOK,
		"auth.logout_success",
		"Sesión cerrada correctamente.",
		gin.H{},
	)
}

func (h *AuthRoutes) findUserByEmail(
	c *gin.Context,
	email string,
) (DashboardUser, string, error) {
	var user DashboardUser
	var passwordHash string

	err := h.DB.QueryRow(
		c.Request.Context(),
		`
			SELECT
	id::text,
	COALESCE(asesor_id::text, ''),
	COALESCE(nombre, ''),
	email,
	password_hash,
	COALESCE(rol, 'admin'),
	COALESCE(activo, TRUE)
FROM usuarios_dashboard
WHERE LOWER(email) = LOWER($1)
LIMIT 1
		`,
		email,
	).Scan(
		&user.ID,
		&user.AsesorID,
		&user.Nombre,
		&user.Email,
		&passwordHash,
		&user.Rol,
		&user.Activo,
	)

	return user, passwordHash, err
}

func (h *AuthRoutes) findUserByID(
	c *gin.Context,
	userID string,
) (DashboardUser, error) {
	var user DashboardUser

	err := h.DB.QueryRow(
		c.Request.Context(),
		`
			SELECT
				id::text,
				COALESCE(nombre, ''),
				email,
				COALESCE(rol, 'admin'),
				COALESCE(activo, TRUE)
			FROM usuarios_dashboard
			WHERE id = $1::uuid
			LIMIT 1
		`,
		userID,
	).Scan(
		&user.ID,
		&user.Nombre,
		&user.Email,
		&user.Rol,
		&user.Activo,
	)

	return user, err
}

func toPublicUser(
	user DashboardUser,
) PublicUser {
	return PublicUser{
		// ID:     user.ID,
		Nombre: user.Nombre,
		Email:  user.Email,
		Rol:    user.Rol,
	}
}
