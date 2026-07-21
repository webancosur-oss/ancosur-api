package routes

import (
	"ancosur-api/controller"
	middleware "ancosur-api/middlewares"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

type UserRoutes struct {
	DB *pgxpool.Pool
}

type CreateDashboardUserRequest struct {
	AsesorID string `json:"asesor_id"`
	Nombre   string `json:"nombre"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Rol      string `json:"rol"`
	Activo   *bool  `json:"activo"`
}

type UpdateDashboardUserRequest struct {
	AsesorID *string `json:"asesor_id"`
	Nombre   *string `json:"nombre"`
	Email    *string `json:"email"`
	Password *string `json:"password"`
	Rol      *string `json:"rol"`
	Activo   *bool   `json:"activo"`
}

type UserResponse struct {
	ID           string `json:"id"`
	AsesorID     string `json:"asesor_id"`
	Nombre       string `json:"nombre"`
	Email        string `json:"email"`
	Rol          string `json:"rol"`
	Activo       bool   `json:"activo"`
	Asesor       string `json:"asesor"`
	UltimoAcceso string `json:"ultimo_acceso"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
}

type userValidationData struct {
	ID       string
	AsesorID string
	Rol      string
}

func RutasUsers(
	api *gin.RouterGroup,
	db *pgxpool.Pool,
) {
	userRoutes := &UserRoutes{
		DB: db,
	}

	protected := api.Group("/users")
	protected.Use(middleware.AuthMiddleware())
	protected.Use(middleware.RoleMiddleware("admin"))

	protected.GET("", userRoutes.GetAllUsers)
	protected.GET("/:id", userRoutes.GetUserByID)
	protected.POST("", userRoutes.CreateUser)
	protected.PATCH("/:id", userRoutes.UpdateUser)
	protected.PUT("/:id", userRoutes.UpdateUser)
	protected.DELETE("/:id", userRoutes.DarBajaUser)
	protected.PATCH("/:id/baja", userRoutes.DarBajaUser)
	protected.PATCH("/:id/reactivar", userRoutes.ReactivarUser)
}

func (h *UserRoutes) GetAllUsers(
	c *gin.Context,
) {
	query := userSelectQuery
	args := []any{}

	search := strings.TrimSpace(c.Query("search"))
	if search != "" {
		args = append(args, "%"+search+"%")
		param := len(args)

		query += fmt.Sprintf(`
			AND (
				u.nombre ILIKE $%d
				OR u.email ILIKE $%d
				OR u.rol ILIKE $%d
				OR a.nombres_completos ILIKE $%d
			)
		`, param, param, param, param)
	}

	rol := strings.ToLower(strings.TrimSpace(c.Query("rol")))
	if rol != "" {
		args = append(args, rol)
		query += fmt.Sprintf(`
			AND LOWER(u.rol) = LOWER($%d)
		`, len(args))
	}

	activo := strings.TrimSpace(c.Query("activo"))
	if activo != "" {
		if strings.EqualFold(activo, "true") {
			query += `
				AND COALESCE(u.activo, TRUE) = TRUE
			`
		}

		if strings.EqualFold(activo, "false") {
			query += `
				AND COALESCE(u.activo, TRUE) = FALSE
			`
		}
	}

	query += `
		ORDER BY u.created_at DESC
	`

	rows, err := h.DB.Query(
		c.Request.Context(),
		query,
		args...,
	)

	if err != nil {
		controller.Error(
			c,
			http.StatusInternalServerError,
			"users.list_error",
			"No se pudo obtener los usuarios.",
			nil,
		)
		return
	}

	defer rows.Close()

	users := []UserResponse{}

	for rows.Next() {
		user, err := scanUser(rows)

		if err != nil {
			controller.Error(
				c,
				http.StatusInternalServerError,
				"users.scan_error",
				"No se pudo leer un usuario.",
				nil,
			)
			return
		}

		users = append(users, user)
	}

	if err := rows.Err(); err != nil {
		controller.Error(
			c,
			http.StatusInternalServerError,
			"users.rows_error",
			"No se pudo completar la lectura de usuarios.",
			nil,
		)
		return
	}

	controller.Success(
		c,
		http.StatusOK,
		"users.list",
		"Usuarios obtenidos correctamente.",
		gin.H{
			"items": users,
			"total": len(users),
		},
	)
}

func (h *UserRoutes) GetUserByID(
	c *gin.Context,
) {
	id := strings.TrimSpace(c.Param("id"))

	if id == "" {
		controller.Error(
			c,
			http.StatusBadRequest,
			"users.invalid_id",
			"ID de usuario inválido.",
			nil,
		)
		return
	}

	row := h.DB.QueryRow(
		c.Request.Context(),
		userSelectQuery+`
			AND u.id = $1::uuid
			LIMIT 1
		`,
		id,
	)

	user, err := scanUserRow(row)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			controller.Error(
				c,
				http.StatusNotFound,
				"users.not_found",
				"Usuario no encontrado.",
				nil,
			)
			return
		}

		controller.Error(
			c,
			http.StatusInternalServerError,
			"users.get_error",
			"No se pudo obtener el usuario.",
			nil,
		)
		return
	}

	controller.Success(
		c,
		http.StatusOK,
		"users.detail",
		"Usuario obtenido correctamente.",
		gin.H{
			"item": user,
		},
	)
}

func (h *UserRoutes) CreateUser(
	c *gin.Context,
) {
	var body CreateDashboardUserRequest

	if !controller.CheckBody(c, &body) {
		return
	}

	asesorID := strings.TrimSpace(body.AsesorID)
	nombre := strings.TrimSpace(body.Nombre)
	email := strings.ToLower(strings.TrimSpace(body.Email))
	password := strings.TrimSpace(body.Password)
	rol := strings.ToLower(strings.TrimSpace(body.Rol))

	if rol == "" {
		if asesorID != "" {
			rol = "asesor"
		} else {
			rol = "lectura"
		}
	}

	if nombre == "" {
		controller.Error(
			c,
			http.StatusBadRequest,
			"users.required_name",
			"El nombre es obligatorio.",
			nil,
		)
		return
	}

	if email == "" {
		controller.Error(
			c,
			http.StatusBadRequest,
			"users.required_email",
			"El correo es obligatorio.",
			nil,
		)
		return
	}

	if password == "" {
		controller.Error(
			c,
			http.StatusBadRequest,
			"users.required_password",
			"La contraseña es obligatoria.",
			nil,
		)
		return
	}

	if len(password) < 6 {
		controller.Error(
			c,
			http.StatusBadRequest,
			"users.weak_password",
			"La contraseña debe tener mínimo 6 caracteres.",
			nil,
		)
		return
	}

	if !isValidUserRol(rol) {
		controller.Error(
			c,
			http.StatusBadRequest,
			"users.invalid_role",
			"El rol enviado no es válido.",
			gin.H{
				"roles_permitidos": []string{
					"admin",
					"marketing",
					"ventas",
					"asesor",
					"lectura",
				},
			},
		)
		return
	}

	if rol == "asesor" && asesorID == "" {
		controller.Error(
			c,
			http.StatusBadRequest,
			"users.required_asesor_id",
			"Para crear un usuario asesor debes enviar asesor_id.",
			nil,
		)
		return
	}

	if rol != "asesor" && asesorID != "" {
		controller.Error(
			c,
			http.StatusBadRequest,
			"users.invalid_asesor_id",
			"Solo los usuarios con rol asesor deben tener asesor_id.",
			nil,
		)
		return
	}

	activo := true

	if body.Activo != nil {
		activo = *body.Activo
	}

	passwordHash, err := bcrypt.GenerateFromPassword(
		[]byte(password),
		bcrypt.DefaultCost,
	)

	if err != nil {
		controller.Error(
			c,
			http.StatusInternalServerError,
			"users.hash_error",
			"No se pudo proteger la contraseña.",
			nil,
		)
		return
	}

	var user UserResponse

	err = h.DB.QueryRow(
		c.Request.Context(),
		`
			INSERT INTO usuarios_dashboard (
				asesor_id,
				nombre,
				email,
				password_hash,
				rol,
				activo,
				created_at,
				updated_at
			)
			VALUES (
				NULLIF($1, '')::uuid,
				$2,
				$3,
				$4,
				$5,
				$6,
				NOW(),
				NOW()
			)
			RETURNING id::text
		`,
		asesorID,
		nombre,
		email,
		string(passwordHash),
		rol,
		activo,
	).Scan(&user.ID)

	if err != nil {
		h.handleUserCreateOrUpdateError(c, err)
		return
	}

	user, err = h.findUserResponseByID(c, user.ID)

	if err != nil {
		controller.Error(
			c,
			http.StatusInternalServerError,
			"users.get_created_error",
			"Usuario creado, pero no se pudo obtener su información.",
			nil,
		)
		return
	}

	controller.Success(
		c,
		http.StatusCreated,
		"users.created",
		"Usuario creado correctamente.",
		gin.H{
			"item": user,
		},
	)
}

func (h *UserRoutes) UpdateUser(
	c *gin.Context,
) {
	id := strings.TrimSpace(c.Param("id"))

	if id == "" {
		controller.Error(
			c,
			http.StatusBadRequest,
			"users.invalid_id",
			"ID de usuario inválido.",
			nil,
		)
		return
	}

	var body UpdateDashboardUserRequest

	if !controller.CheckBody(c, &body) {
		return
	}

	currentUser, err := h.findUserValidationByID(c, id)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			controller.Error(
				c,
				http.StatusNotFound,
				"users.not_found",
				"Usuario no encontrado.",
				nil,
			)
			return
		}

		controller.Error(
			c,
			http.StatusInternalServerError,
			"users.get_error",
			"No se pudo obtener el usuario.",
			nil,
		)
		return
	}

	newRol := currentUser.Rol
	if body.Rol != nil {
		newRol = strings.ToLower(strings.TrimSpace(*body.Rol))
	}

	if !isValidUserRol(newRol) {
		controller.Error(
			c,
			http.StatusBadRequest,
			"users.invalid_role",
			"El rol enviado no es válido.",
			gin.H{
				"roles_permitidos": []string{
					"admin",
					"marketing",
					"ventas",
					"asesor",
					"lectura",
				},
			},
		)
		return
	}

	newAsesorID := currentUser.AsesorID
	if body.AsesorID != nil {
		newAsesorID = strings.TrimSpace(*body.AsesorID)
	}

	if newRol == "asesor" && newAsesorID == "" {
		controller.Error(
			c,
			http.StatusBadRequest,
			"users.required_asesor_id",
			"Para un usuario asesor debes enviar asesor_id.",
			nil,
		)
		return
	}

	if newRol != "asesor" && newAsesorID != "" {
		controller.Error(
			c,
			http.StatusBadRequest,
			"users.invalid_asesor_id",
			"Solo los usuarios con rol asesor deben tener asesor_id.",
			nil,
		)
		return
	}

	var passwordHash any = nil

	if body.Password != nil {
		password := strings.TrimSpace(*body.Password)

		if password == "" {
			controller.Error(
				c,
				http.StatusBadRequest,
				"users.empty_password",
				"La contraseña no puede estar vacía.",
				nil,
			)
			return
		}

		if len(password) < 6 {
			controller.Error(
				c,
				http.StatusBadRequest,
				"users.weak_password",
				"La contraseña debe tener mínimo 6 caracteres.",
				nil,
			)
			return
		}

		hash, err := bcrypt.GenerateFromPassword(
			[]byte(password),
			bcrypt.DefaultCost,
		)

		if err != nil {
			controller.Error(
				c,
				http.StatusInternalServerError,
				"users.hash_error",
				"No se pudo proteger la contraseña.",
				nil,
			)
			return
		}

		passwordHash = string(hash)
	}

	commandTag, err := h.DB.Exec(
		c.Request.Context(),
		`
			UPDATE usuarios_dashboard
			SET
				asesor_id = CASE
					WHEN $2::text IS NULL THEN asesor_id
					ELSE NULLIF($2::text, '')::uuid
				END,
				nombre = COALESCE($3::text, nombre),
				email = COALESCE($4::text, email),
				password_hash = COALESCE($5::text, password_hash),
				rol = COALESCE($6::text, rol),
				activo = COALESCE($7::boolean, activo),
				updated_at = NOW()
			WHERE id = $1::uuid
		`,
		id,
		userOptionalString(&newAsesorID),
		userOptionalString(body.Nombre),
		userOptionalEmail(body.Email),
		passwordHash,
		newRol,
		userOptionalBool(body.Activo),
	)

	if err != nil {
		h.handleUserCreateOrUpdateError(c, err)
		return
	}

	if commandTag.RowsAffected() == 0 {
		controller.Error(
			c,
			http.StatusNotFound,
			"users.not_found",
			"Usuario no encontrado.",
			nil,
		)
		return
	}

	user, err := h.findUserResponseByID(c, id)

	if err != nil {
		controller.Error(
			c,
			http.StatusInternalServerError,
			"users.get_updated_error",
			"Usuario actualizado, pero no se pudo obtener su información.",
			nil,
		)
		return
	}

	controller.Success(
		c,
		http.StatusOK,
		"users.updated",
		"Usuario actualizado correctamente.",
		gin.H{
			"item": user,
		},
	)
}

func (h *UserRoutes) DarBajaUser(
	c *gin.Context,
) {
	id := strings.TrimSpace(c.Param("id"))

	if id == "" {
		controller.Error(
			c,
			http.StatusBadRequest,
			"users.invalid_id",
			"ID de usuario inválido.",
			nil,
		)
		return
	}

	currentUserID := strings.TrimSpace(c.GetString("user_id"))

	if currentUserID == id {
		controller.Error(
			c,
			http.StatusBadRequest,
			"users.cannot_disable_self",
			"No puedes dar de baja tu propio usuario.",
			nil,
		)
		return
	}

	commandTag, err := h.DB.Exec(
		c.Request.Context(),
		`
			UPDATE usuarios_dashboard
			SET
				activo = FALSE,
				updated_at = NOW()
			WHERE id = $1::uuid
		`,
		id,
	)

	if err != nil {
		controller.Error(
			c,
			http.StatusInternalServerError,
			"users.disable_error",
			"No se pudo dar de baja el usuario.",
			nil,
		)
		return
	}

	if commandTag.RowsAffected() == 0 {
		controller.Error(
			c,
			http.StatusNotFound,
			"users.not_found",
			"Usuario no encontrado.",
			nil,
		)
		return
	}

	controller.Success(
		c,
		http.StatusOK,
		"users.disabled",
		"Usuario dado de baja correctamente.",
		gin.H{
			"id": id,
		},
	)
}

func (h *UserRoutes) ReactivarUser(
	c *gin.Context,
) {
	id := strings.TrimSpace(c.Param("id"))

	if id == "" {
		controller.Error(
			c,
			http.StatusBadRequest,
			"users.invalid_id",
			"ID de usuario inválido.",
			nil,
		)
		return
	}

	commandTag, err := h.DB.Exec(
		c.Request.Context(),
		`
			UPDATE usuarios_dashboard
			SET
				activo = TRUE,
				updated_at = NOW()
			WHERE id = $1::uuid
		`,
		id,
	)

	if err != nil {
		controller.Error(
			c,
			http.StatusInternalServerError,
			"users.reactivate_error",
			"No se pudo reactivar el usuario.",
			nil,
		)
		return
	}

	if commandTag.RowsAffected() == 0 {
		controller.Error(
			c,
			http.StatusNotFound,
			"users.not_found",
			"Usuario no encontrado.",
			nil,
		)
		return
	}

	controller.Success(
		c,
		http.StatusOK,
		"users.reactivated",
		"Usuario reactivado correctamente.",
		gin.H{
			"id": id,
		},
	)
}

func (h *UserRoutes) findUserResponseByID(
	c *gin.Context,
	id string,
) (UserResponse, error) {
	row := h.DB.QueryRow(
		c.Request.Context(),
		userSelectQuery+`
			AND u.id = $1::uuid
			LIMIT 1
		`,
		id,
	)

	return scanUserRow(row)
}

func (h *UserRoutes) findUserValidationByID(
	c *gin.Context,
	id string,
) (userValidationData, error) {
	var user userValidationData

	err := h.DB.QueryRow(
		c.Request.Context(),
		`
			SELECT
				id::text,
				COALESCE(asesor_id::text, ''),
				COALESCE(rol, 'lectura')
			FROM usuarios_dashboard
			WHERE id = $1::uuid
			LIMIT 1
		`,
		id,
	).Scan(
		&user.ID,
		&user.AsesorID,
		&user.Rol,
	)

	return user, err
}

func (h *UserRoutes) handleUserCreateOrUpdateError(
	c *gin.Context,
	err error,
) {
	isUnique, constraintName := isUniqueViolationUser(err)

	if isUnique {
		message := "Ya existe un usuario con esos datos."

		if constraintName == "usuarios_dashboard_email_key" ||
			constraintName == "uq_usuarios_dashboard_email" {
			message = "Ya existe un usuario con ese correo."
		}

		if constraintName == "uq_usuarios_dashboard_asesor_id" {
			message = "Ese asesor ya tiene una cuenta de dashboard."
		}

		controller.Error(
			c,
			http.StatusConflict,
			"users.duplicated",
			message,
			nil,
		)
		return
	}

	if isForeignKeyViolationUser(err) {
		controller.Error(
			c,
			http.StatusBadRequest,
			"users.invalid_foreign_key",
			"El asesor seleccionado no existe.",
			nil,
		)
		return
	}

	controller.Error(
		c,
		http.StatusInternalServerError,
		"users.save_error",
		"No se pudo guardar el usuario.",
		nil,
	)
}

const userSelectQuery = `
	SELECT
		u.id::text,
		COALESCE(u.asesor_id::text, ''),
		COALESCE(u.nombre, ''),
		COALESCE(u.email, ''),
		COALESCE(u.rol, ''),
		COALESCE(u.activo, TRUE),
		COALESCE(a.nombres_completos, ''),
		COALESCE(u.ultimo_acceso::text, ''),
		COALESCE(u.created_at::text, ''),
		COALESCE(u.updated_at::text, '')
	FROM usuarios_dashboard u
	LEFT JOIN asesores a
		ON a.id = u.asesor_id
	WHERE 1 = 1
`

type userScanner interface {
	Scan(dest ...any) error
}

func scanUserRow(
	row userScanner,
) (UserResponse, error) {
	var user UserResponse

	err := row.Scan(
		&user.ID,
		&user.AsesorID,
		&user.Nombre,
		&user.Email,
		&user.Rol,
		&user.Activo,
		&user.Asesor,
		&user.UltimoAcceso,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	return user, err
}

func scanUser(
	rows pgx.Rows,
) (UserResponse, error) {
	return scanUserRow(rows)
}

func isValidUserRol(
	rol string,
) bool {
	validRoles := map[string]bool{
		"admin":     true,
		"marketing": true,
		"ventas":    true,
		"asesor":    true,
		"lectura":   true,
	}

	return validRoles[rol]
}

func userOptionalString(
	value *string,
) any {
	if value == nil {
		return nil
	}

	return strings.TrimSpace(*value)
}

func userOptionalEmail(
	value *string,
) any {
	if value == nil {
		return nil
	}

	return strings.ToLower(
		strings.TrimSpace(*value),
	)
}

func userOptionalBool(
	value *bool,
) any {
	if value == nil {
		return nil
	}

	return *value
}

func isUniqueViolationUser(
	err error,
) (bool, string) {
	var pgErr *pgconn.PgError

	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return true, pgErr.ConstraintName
	}

	return false, ""
}

func isForeignKeyViolationUser(
	err error,
) bool {
	var pgErr *pgconn.PgError

	return errors.As(err, &pgErr) && pgErr.Code == "23503"
}