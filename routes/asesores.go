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
)

type AsesorRoutes struct {
	DB *pgxpool.Pool
}

type AsesorRequest struct {
	NombresCompletos string `json:"nombres_completos"`
	Telefono          string `json:"telefono"`
	Email             string `json:"email"`
	Activo            *bool  `json:"activo"`
}

type AsesorResponse struct {
	ID                string `json:"id"`
	NombresCompletos string `json:"nombres_completos"`
	Telefono          string `json:"telefono"`
	Email             string `json:"email"`
	Activo            bool   `json:"activo"`
	CreatedAt         string `json:"created_at"`
	UpdatedAt         string `json:"updated_at"`
}

func RutasAsesores(
	api *gin.RouterGroup,
	db *pgxpool.Pool,
) {
	asesorRoutes := &AsesorRoutes{
		DB: db,
	}

	protected := api.Group("/asesores")
	protected.Use(middleware.AuthMiddleware())

	protected.GET("", asesorRoutes.GetAllAsesores)
	protected.GET("/:id", asesorRoutes.GetAsesorByID)

	admin := api.Group("/asesores")
	admin.Use(middleware.AuthMiddleware())
	admin.Use(middleware.RoleMiddleware("admin"))

	admin.POST("", asesorRoutes.CreateAsesor)
	admin.PATCH("/:id/baja", asesorRoutes.DarBajaAsesor)
	admin.DELETE("/:id", asesorRoutes.DarBajaAsesor)
	admin.PATCH("/:id/reactivar", asesorRoutes.ReactivarAsesor)
}

func (h *AsesorRoutes) GetAllAsesores(
	c *gin.Context,
) {
	userRol := c.GetString("user_rol")
	userAsesorID := strings.TrimSpace(
		c.GetString("user_asesor_id"),
	)

	query := asesorSelectQuery
	args := []any{}

	if userRol == "asesor" {
		if userAsesorID == "" {
			controller.Error(
				c,
				http.StatusForbidden,
				"asesores.not_linked",
				"Tu usuario no está vinculado a un asesor.",
				nil,
			)
			return
		}

		args = append(args, userAsesorID)

		query += fmt.Sprintf(`
			AND a.id = $%d::uuid
			AND a.deleted_at IS NULL
			AND COALESCE(a.activo, TRUE) = TRUE
		`, len(args))
	} else {
		includeInactive := strings.EqualFold(
			strings.TrimSpace(c.Query("include_inactive")),
			"true",
		)

		if !includeInactive {
			query += `
				AND a.deleted_at IS NULL
				AND COALESCE(a.activo, TRUE) = TRUE
			`
		}
	}

	search := strings.TrimSpace(c.Query("search"))

	if search != "" {
		args = append(args, "%"+search+"%")
		param := len(args)

		query += fmt.Sprintf(`
			AND (
				a.nombres_completos ILIKE $%d
				OR a.telefono ILIKE $%d
				OR a.email ILIKE $%d
			)
		`, param, param, param)
	}

	query += `
		ORDER BY a.nombres_completos ASC
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
			"asesores.list_error",
			"No se pudo obtener los asesores.",
			errorDataAsesor(err),
		)
		return
	}

	defer rows.Close()

	asesores := []AsesorResponse{}

	for rows.Next() {
		asesor, err := scanAsesor(rows)

		if err != nil {
			controller.Error(
				c,
				http.StatusInternalServerError,
				"asesores.scan_error",
				"No se pudo leer un asesor.",
				errorDataAsesor(err),
			)
			return
		}

		asesores = append(asesores, asesor)
	}

	if err := rows.Err(); err != nil {
		controller.Error(
			c,
			http.StatusInternalServerError,
			"asesores.rows_error",
			"No se pudo completar la lectura de asesores.",
			errorDataAsesor(err),
		)
		return
	}

	controller.Success(
		c,
		http.StatusOK,
		"asesores.list",
		"Asesores obtenidos correctamente.",
		gin.H{
			"items": asesores,
			"total": len(asesores),
		},
	)
}

func (h *AsesorRoutes) GetAsesorByID(
	c *gin.Context,
) {
	id := strings.TrimSpace(c.Param("id"))

	if id == "" {
		controller.Error(
			c,
			http.StatusBadRequest,
			"asesores.invalid_id",
			"ID de asesor inválido.",
			nil,
		)
		return
	}

	userRol := c.GetString("user_rol")
	userAsesorID := strings.TrimSpace(
		c.GetString("user_asesor_id"),
	)

	if userRol == "asesor" && id != userAsesorID {
		controller.Error(
			c,
			http.StatusForbidden,
			"asesores.forbidden",
			"No puedes ver información de otro asesor.",
			nil,
		)
		return
	}

	row := h.DB.QueryRow(
		c.Request.Context(),
		asesorSelectQuery+`
			AND a.id = $1::uuid
			LIMIT 1
		`,
		id,
	)

	asesor, err := scanAsesorRow(row)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			controller.Error(
				c,
				http.StatusNotFound,
				"asesores.not_found",
				"Asesor no encontrado.",
				nil,
			)
			return
		}

		controller.Error(
			c,
			http.StatusInternalServerError,
			"asesores.get_error",
			"No se pudo obtener el asesor.",
			errorDataAsesor(err),
		)
		return
	}

	controller.Success(
		c,
		http.StatusOK,
		"asesores.detail",
		"Asesor obtenido correctamente.",
		gin.H{
			"item": asesor,
		},
	)
}

func (h *AsesorRoutes) CreateAsesor(
	c *gin.Context,
) {
	var body AsesorRequest

	if !controller.CheckBody(c, &body) {
		return
	}

	nombresCompletos := strings.TrimSpace(body.NombresCompletos)
	telefono := strings.TrimSpace(body.Telefono)
	email := strings.ToLower(strings.TrimSpace(body.Email))

	if nombresCompletos == "" {
		controller.Error(
			c,
			http.StatusBadRequest,
			"asesores.required_name",
			"El nombre completo del asesor es obligatorio.",
			nil,
		)
		return
	}

	activo := true

	if body.Activo != nil {
		activo = *body.Activo
	}

	var asesor AsesorResponse

	err := h.DB.QueryRow(
		c.Request.Context(),
		`
			INSERT INTO asesores (
				nombres_completos,
				telefono,
				email,
				activo,
				created_at,
				updated_at
			)
			VALUES (
				$1,
				$2,
				$3,
				$4,
				NOW(),
				NOW()
			)
			RETURNING
				id::text,
				COALESCE(nombres_completos, ''),
				COALESCE(telefono, ''),
				COALESCE(email, ''),
				COALESCE(activo, TRUE),
				COALESCE(created_at::text, ''),
				COALESCE(updated_at::text, '')
		`,
		nombresCompletos,
		telefono,
		email,
		activo,
	).Scan(
		&asesor.ID,
		&asesor.NombresCompletos,
		&asesor.Telefono,
		&asesor.Email,
		&asesor.Activo,
		&asesor.CreatedAt,
		&asesor.UpdatedAt,
	)

	if err != nil {
		if isUniqueViolationAsesor(err) {
			controller.Error(
				c,
				http.StatusConflict,
				"asesores.email_exists",
				"Ya existe un asesor con ese correo.",
				nil,
			)
			return
		}

		controller.Error(
			c,
			http.StatusInternalServerError,
			"asesores.create_error",
			"No se pudo crear el asesor.",
			errorDataAsesor(err),
		)
		return
	}

	controller.Success(
		c,
		http.StatusCreated,
		"asesores.created",
		"Asesor creado correctamente.",
		gin.H{
			"item": asesor,
		},
	)
}

func (h *AsesorRoutes) DarBajaAsesor(
	c *gin.Context,
) {
	id := strings.TrimSpace(c.Param("id"))

	if id == "" {
		controller.Error(
			c,
			http.StatusBadRequest,
			"asesores.invalid_id",
			"ID de asesor inválido.",
			nil,
		)
		return
	}

	tx, err := h.DB.Begin(
		c.Request.Context(),
	)

	if err != nil {
		controller.Error(
			c,
			http.StatusInternalServerError,
			"asesores.tx_error",
			"No se pudo iniciar la operación.",
			errorDataAsesor(err),
		)
		return
	}

	defer tx.Rollback(c.Request.Context())

	commandTag, err := tx.Exec(
		c.Request.Context(),
		`
			UPDATE asesores
			SET
				activo = FALSE,
				deleted_at = NOW(),
				updated_at = NOW()
			WHERE id = $1::uuid
			  AND deleted_at IS NULL
		`,
		id,
	)

	if err != nil {
		controller.Error(
			c,
			http.StatusInternalServerError,
			"asesores.disable_error",
			"No se pudo dar de baja al asesor.",
			errorDataAsesor(err),
		)
		return
	}

	if commandTag.RowsAffected() == 0 {
		controller.Error(
			c,
			http.StatusNotFound,
			"asesores.not_found",
			"Asesor no encontrado o ya fue dado de baja.",
			nil,
		)
		return
	}

	_, _ = tx.Exec(
		c.Request.Context(),
		`
			UPDATE usuarios_dashboard
			SET
				activo = FALSE,
				updated_at = NOW()
			WHERE asesor_id = $1::uuid
		`,
		id,
	)

	if err := tx.Commit(c.Request.Context()); err != nil {
		controller.Error(
			c,
			http.StatusInternalServerError,
			"asesores.commit_error",
			"No se pudo confirmar la baja del asesor.",
			errorDataAsesor(err),
		)
		return
	}

	controller.Success(
		c,
		http.StatusOK,
		"asesores.disabled",
		"Asesor dado de baja correctamente.",
		gin.H{
			"id": id,
		},
	)
}

func (h *AsesorRoutes) ReactivarAsesor(
	c *gin.Context,
) {
	id := strings.TrimSpace(c.Param("id"))

	if id == "" {
		controller.Error(
			c,
			http.StatusBadRequest,
			"asesores.invalid_id",
			"ID de asesor inválido.",
			nil,
		)
		return
	}

	commandTag, err := h.DB.Exec(
		c.Request.Context(),
		`
			UPDATE asesores
			SET
				activo = TRUE,
				deleted_at = NULL,
				updated_at = NOW()
			WHERE id = $1::uuid
		`,
		id,
	)

	if err != nil {
		controller.Error(
			c,
			http.StatusInternalServerError,
			"asesores.reactivate_error",
			"No se pudo reactivar al asesor.",
			errorDataAsesor(err),
		)
		return
	}

	if commandTag.RowsAffected() == 0 {
		controller.Error(
			c,
			http.StatusNotFound,
			"asesores.not_found",
			"Asesor no encontrado.",
			nil,
		)
		return
	}

	controller.Success(
		c,
		http.StatusOK,
		"asesores.reactivated",
		"Asesor reactivado correctamente.",
		gin.H{
			"id": id,
		},
	)
}

const asesorSelectQuery = `
	SELECT
		a.id::text,
		COALESCE(a.nombres_completos, ''),
		COALESCE(a.telefono, ''),
		COALESCE(a.email, ''),
		COALESCE(a.activo, TRUE),
		COALESCE(a.created_at::text, ''),
		COALESCE(a.updated_at::text, '')
	FROM asesores a
	WHERE 1 = 1
`

type asesorScanner interface {
	Scan(dest ...any) error
}

func scanAsesorRow(
	row asesorScanner,
) (AsesorResponse, error) {
	var asesor AsesorResponse

	err := row.Scan(
		&asesor.ID,
		&asesor.NombresCompletos,
		&asesor.Telefono,
		&asesor.Email,
		&asesor.Activo,
		&asesor.CreatedAt,
		&asesor.UpdatedAt,
	)

	return asesor, err
}

func scanAsesor(
	rows pgx.Rows,
) (AsesorResponse, error) {
	return scanAsesorRow(rows)
}

func isUniqueViolationAsesor(
	err error,
) bool {
	var pgErr *pgconn.PgError

	return errors.As(
		err,
		&pgErr,
	) && pgErr.Code == "23505"
}

func errorDataAsesor(
	err error,
) any {
	if err == nil {
		return nil
	}

	return gin.H{
		"error": err.Error(),
	}
}