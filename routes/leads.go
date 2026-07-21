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
	"github.com/jackc/pgx/v5/pgxpool"
)

type LeadRoutes struct {
	DB *pgxpool.Pool
}

type LeadRequest struct {
	AsesorID     string `json:"asesor_id"`
	EstadoLeadID string `json:"estado_lead_id"`
	ProyectoID   string `json:"proyecto_id"`
	CampaniaID   string `json:"campania_id"`

	NombresCompletos string `json:"nombres_completos"`
	TipoDocumento    string `json:"tipo_documento"`
	NumeroDocumento  string `json:"numero_documento"`
	Telefono         string `json:"telefono"`
	Email            string `json:"email"`
	Mensaje          string `json:"mensaje"`

	ProyectoInteres   string `json:"proyecto_interes"`
	CategoriaInteres  string `json:"categoria_interes"`
	FuenteProspeccion string `json:"fuente_prospeccion"`
	OrigenRuta        string `json:"origen_ruta"`
	OrigenComponente  string `json:"origen_componente"`

	Activo *bool `json:"activo"`
}

type LeadUpdateRequest struct {
	AsesorID     *string `json:"asesor_id"`
	EstadoLeadID *string `json:"estado_lead_id"`
	ProyectoID   *string `json:"proyecto_id"`
	CampaniaID   *string `json:"campania_id"`

	NombresCompletos *string `json:"nombres_completos"`
	TipoDocumento    *string `json:"tipo_documento"`
	NumeroDocumento  *string `json:"numero_documento"`
	Telefono         *string `json:"telefono"`
	Email            *string `json:"email"`
	Mensaje          *string `json:"mensaje"`

	ProyectoInteres   *string `json:"proyecto_interes"`
	CategoriaInteres  *string `json:"categoria_interes"`
	FuenteProspeccion *string `json:"fuente_prospeccion"`
	OrigenRuta        *string `json:"origen_ruta"`
	OrigenComponente  *string `json:"origen_componente"`

	Activo *bool `json:"activo"`
}

type LeadResponse struct {
	ID string `json:"id"`

	AsesorID     string `json:"asesor_id"`
	EstadoLeadID string `json:"estado_lead_id"`
	ProyectoID   string `json:"proyecto_id"`
	CampaniaID   string `json:"campania_id"`

	NombresCompletos string `json:"nombres_completos"`
	TipoDocumento    string `json:"tipo_documento"`
	NumeroDocumento  string `json:"numero_documento"`
	Telefono         string `json:"telefono"`
	Email            string `json:"email"`
	Mensaje          string `json:"mensaje"`

	ProyectoInteres   string `json:"proyecto_interes"`
	CategoriaInteres  string `json:"categoria_interes"`
	FuenteProspeccion string `json:"fuente_prospeccion"`
	OrigenRuta        string `json:"origen_ruta"`
	OrigenComponente  string `json:"origen_componente"`

	Activo bool `json:"activo"`

	EstadoLead string `json:"estado_lead"`
	Asesor     string `json:"asesor"`
	Proyecto   string `json:"proyecto"`

	UpdatedProyecto string `json:"proyecto"`
	Campania        string `json:"campania"`

	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

func RutasLeads(
	api *gin.RouterGroup,
	db *pgxpool.Pool,
) {
	leadRoutes := &LeadRoutes{
		DB: db,
	}

	// Público: formularios web
	api.POST("/leads", leadRoutes.CreateLead)

	// Protegido: dashboard
	protected := api.Group("/leads")
	protected.Use(middleware.AuthMiddleware())

	protected.GET("", leadRoutes.GetAllLeads)
	protected.GET("/:id", leadRoutes.GetLeadByID)
	protected.PATCH("/:id", leadRoutes.UpdateLead)
	protected.PUT("/:id", leadRoutes.UpdateLead)
	protected.DELETE("/:id", leadRoutes.DeleteLead)
}

func (h *LeadRoutes) GetAllLeads(c *gin.Context) {
	query := leadSelectQuery
	args := []any{}

	search := strings.TrimSpace(c.Query("search"))
	if search != "" {
		args = append(args, "%"+search+"%")
		query += fmt.Sprintf(`
			AND (
				l.nombres_completos ILIKE $%d
				OR l.telefono ILIKE $%d
				OR l.email ILIKE $%d
				OR l.numero_documento ILIKE $%d
				OR l.proyecto_interes ILIKE $%d
				OR l.mensaje ILIKE $%d
			)
		`, len(args), len(args), len(args), len(args), len(args), len(args))
	}

	estadoLeadID := strings.TrimSpace(c.Query("estado_lead_id"))
	if estadoLeadID != "" {
		args = append(args, estadoLeadID)
		query += fmt.Sprintf(`
			AND l.estado_lead_id = $%d::uuid
		`, len(args))
	}

	userRol := c.GetString("user_rol")

	asesorID := strings.TrimSpace(c.Query("asesor_id"))

	if userRol == "asesor" {
		if asesorID == "" {
			controller.Error(
				c,
				http.StatusForbidden,
				"leads.asesor_not_linked",
				"Tu usuario no está vinculado a un asesor.",
				nil,
			)
			return
		}

		args = append(args, asesorID)
		query += fmt.Sprintf(`
		AND l.asesor_id = $%d::uuid
	`, len(args))
	}

	if asesorID != "" {
		args = append(args, asesorID)
		query += fmt.Sprintf(`
			AND l.asesor_id = $%d::uuid
		`, len(args))
	}

	proyectoID := strings.TrimSpace(c.Query("proyecto_id"))
	if proyectoID != "" {
		args = append(args, proyectoID)
		query += fmt.Sprintf(`
			AND l.proyecto_id = $%d::uuid
		`, len(args))
	}

	categoriaInteres := strings.TrimSpace(c.Query("categoria_interes"))
	if categoriaInteres != "" {
		args = append(args, categoriaInteres)
		query += fmt.Sprintf(`
			AND LOWER(l.categoria_interes) = LOWER($%d)
		`, len(args))
	}

	query += `
		ORDER BY l.created_at DESC
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
			"leads.list_error",
			"No se pudo obtener los leads.",
			errorData(err),
		)
		return
	}

	defer rows.Close()

	leads := []LeadResponse{}

	for rows.Next() {
		lead, err := scanLead(rows)

		if err != nil {
			controller.Error(
				c,
				http.StatusInternalServerError,
				"leads.scan_error",
				"No se pudo leer un lead.",
				errorData(err),
			)
			return
		}

		leads = append(leads, lead)
	}

	if err := rows.Err(); err != nil {
		controller.Error(
			c,
			http.StatusInternalServerError,
			"leads.rows_error",
			"No se pudo completar la lectura de leads.",
			errorData(err),
		)
		return
	}

	controller.Success(
		c,
		http.StatusOK,
		"leads.list",
		"Leads obtenidos correctamente.",
		gin.H{
			"items": leads,
			"total": len(leads),
		},
	)
}

func (h *LeadRoutes) GetLeadByID(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))

	if id == "" {
		controller.Error(
			c,
			http.StatusBadRequest,
			"leads.invalid_id",
			"ID de lead inválido.",
			nil,
		)
		return
	}

	row := h.DB.QueryRow(
		c.Request.Context(),
		leadSelectQuery+`
			AND l.id = $1::uuid
			LIMIT 1
		`,
		id,
	)

	lead, err := scanLeadRow(row)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			controller.Error(
				c,
				http.StatusNotFound,
				"leads.not_found",
				"Lead no encontrado.",
				nil,
			)
			return
		}

		controller.Error(
			c,
			http.StatusInternalServerError,
			"leads.get_error",
			"No se pudo obtener el lead.",
			errorData(err),
		)
		return
	}

	controller.Success(
		c,
		http.StatusOK,
		"leads.detail",
		"Lead obtenido correctamente.",
		gin.H{
			"item": lead,
		},
	)
}

func (h *LeadRoutes) CreateLead(c *gin.Context) {
	var body LeadRequest

	if !controller.CheckBody(c, &body) {
		return
	}

	body.AsesorID = strings.TrimSpace(body.AsesorID)
	body.EstadoLeadID = strings.TrimSpace(body.EstadoLeadID)
	body.ProyectoID = strings.TrimSpace(body.ProyectoID)
	body.CampaniaID = strings.TrimSpace(body.CampaniaID)

	body.NombresCompletos = strings.TrimSpace(body.NombresCompletos)
	body.TipoDocumento = strings.TrimSpace(body.TipoDocumento)
	body.NumeroDocumento = strings.TrimSpace(body.NumeroDocumento)
	body.Telefono = strings.TrimSpace(body.Telefono)
	body.Email = strings.TrimSpace(body.Email)
	body.Mensaje = strings.TrimSpace(body.Mensaje)

	body.ProyectoInteres = strings.TrimSpace(body.ProyectoInteres)
	body.CategoriaInteres = strings.TrimSpace(body.CategoriaInteres)
	body.FuenteProspeccion = strings.TrimSpace(body.FuenteProspeccion)
	body.OrigenRuta = strings.TrimSpace(body.OrigenRuta)
	body.OrigenComponente = strings.TrimSpace(body.OrigenComponente)

	if body.NombresCompletos == "" {
		controller.Error(
			c,
			http.StatusBadRequest,
			"leads.required_name",
			"El nombre completo es obligatorio.",
			nil,
		)
		return
	}

	if body.Telefono == "" {
		controller.Error(
			c,
			http.StatusBadRequest,
			"leads.required_phone",
			"El teléfono es obligatorio.",
			nil,
		)
		return
	}

	if body.FuenteProspeccion == "" {
		body.FuenteProspeccion = "Web"
	}

	if body.CategoriaInteres == "" {
		body.CategoriaInteres = "General"
	}

	estadoLeadID := body.EstadoLeadID

	if estadoLeadID == "" {
		var err error

		estadoLeadID, err = h.getEstadoLeadIDByName(c, "Nuevo")

		if err != nil {
			controller.Error(
				c,
				http.StatusInternalServerError,
				"leads.estado_error",
				"No se encontró el estado Nuevo.",
				errorData(err),
			)
			return
		}
	}

	activo := true

	if body.Activo != nil {
		activo = *body.Activo
	}

	var leadID string

	err := h.DB.QueryRow(
		c.Request.Context(),
		`
			INSERT INTO leads (
				asesor_id,
				estado_lead_id,
				proyecto_id,
				campania_id,
				nombres_completos,
				tipo_documento,
				numero_documento,
				telefono,
				email,
				mensaje,
				proyecto_interes,
				categoria_interes,
				fuente_prospeccion,
				origen_ruta,
				origen_componente,
				activo,
				created_at,
				updated_at
			)
			VALUES (
				NULLIF($1, '')::uuid,
				NULLIF($2, '')::uuid,
				NULLIF($3, '')::uuid,
				NULLIF($4, '')::uuid,
				$5,
				$6,
				$7,
				$8,
				$9,
				$10,
				$11,
				$12,
				$13,
				$14,
				$15,
				$16,
				NOW(),
				NOW()
			)
			RETURNING id::text
		`,
		body.AsesorID,
		estadoLeadID,
		body.ProyectoID,
		body.CampaniaID,
		body.NombresCompletos,
		body.TipoDocumento,
		body.NumeroDocumento,
		body.Telefono,
		body.Email,
		body.Mensaje,
		body.ProyectoInteres,
		body.CategoriaInteres,
		body.FuenteProspeccion,
		body.OrigenRuta,
		body.OrigenComponente,
		activo,
	).Scan(&leadID)

	if err != nil {
		controller.Error(
			c,
			http.StatusInternalServerError,
			"leads.create_error",
			"No se pudo registrar el lead.",
			errorData(err),
		)
		return
	}

	controller.Success(
		c,
		http.StatusCreated,
		"leads.created",
		"Lead registrado correctamente.",
		gin.H{
			"id": leadID,
		},
	)
}

func (h *LeadRoutes) UpdateLead(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))

	if id == "" {
		controller.Error(
			c,
			http.StatusBadRequest,
			"leads.invalid_id",
			"ID de lead inválido.",
			nil,
		)
		return
	}

	var body LeadUpdateRequest

	if !controller.CheckBody(c, &body) {
		return
	}

	commandTag, err := h.DB.Exec(
		c.Request.Context(),
		`
			UPDATE leads
			SET
				asesor_id = CASE
					WHEN $2::text IS NULL THEN asesor_id
					ELSE NULLIF($2::text, '')::uuid
				END,

				estado_lead_id = CASE
					WHEN $3::text IS NULL THEN estado_lead_id
					ELSE NULLIF($3::text, '')::uuid
				END,

				proyecto_id = CASE
					WHEN $4::text IS NULL THEN proyecto_id
					ELSE NULLIF($4::text, '')::uuid
				END,

				campania_id = CASE
					WHEN $5::text IS NULL THEN campania_id
					ELSE NULLIF($5::text, '')::uuid
				END,

				nombres_completos = COALESCE($6::text, nombres_completos),
				tipo_documento = COALESCE($7::text, tipo_documento),
				numero_documento = COALESCE($8::text, numero_documento),
				telefono = COALESCE($9::text, telefono),
				email = COALESCE($10::text, email),
				mensaje = COALESCE($11::text, mensaje),
				proyecto_interes = COALESCE($12::text, proyecto_interes),
				categoria_interes = COALESCE($13::text, categoria_interes),
				fuente_prospeccion = COALESCE($14::text, fuente_prospeccion),
				origen_ruta = COALESCE($15::text, origen_ruta),
				origen_componente = COALESCE($16::text, origen_componente),
				activo = COALESCE($17::boolean, activo),
				updated_at = NOW()
			WHERE id = $1::uuid
			  AND deleted_at IS NULL
		`,
		id,
		optionalString(body.AsesorID),
		optionalString(body.EstadoLeadID),
		optionalString(body.ProyectoID),
		optionalString(body.CampaniaID),
		optionalString(body.NombresCompletos),
		optionalString(body.TipoDocumento),
		optionalString(body.NumeroDocumento),
		optionalString(body.Telefono),
		optionalString(body.Email),
		optionalString(body.Mensaje),
		optionalString(body.ProyectoInteres),
		optionalString(body.CategoriaInteres),
		optionalString(body.FuenteProspeccion),
		optionalString(body.OrigenRuta),
		optionalString(body.OrigenComponente),
		optionalBool(body.Activo),
	)

	if err != nil {
		controller.Error(
			c,
			http.StatusInternalServerError,
			"leads.update_error",
			"No se pudo actualizar el lead.",
			errorData(err),
		)
		return
	}

	if commandTag.RowsAffected() == 0 {
		controller.Error(
			c,
			http.StatusNotFound,
			"leads.not_found",
			"Lead no encontrado.",
			nil,
		)
		return
	}

	controller.Success(
		c,
		http.StatusOK,
		"leads.updated",
		"Lead actualizado correctamente.",
		gin.H{
			"id": id,
		},
	)
}

func (h *LeadRoutes) DeleteLead(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))

	if id == "" {
		controller.Error(
			c,
			http.StatusBadRequest,
			"leads.invalid_id",
			"ID de lead inválido.",
			nil,
		)
		return
	}

	commandTag, err := h.DB.Exec(
		c.Request.Context(),
		`
			UPDATE leads
			SET
				deleted_at = NOW(),
				activo = FALSE,
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
			"leads.delete_error",
			"No se pudo eliminar el lead.",
			errorData(err),
		)
		return
	}

	if commandTag.RowsAffected() == 0 {
		controller.Error(
			c,
			http.StatusNotFound,
			"leads.not_found",
			"Lead no encontrado.",
			nil,
		)
		return
	}

	controller.Success(
		c,
		http.StatusOK,
		"leads.deleted",
		"Lead eliminado correctamente.",
		gin.H{
			"id": id,
		},
	)
}

func (h *LeadRoutes) getEstadoLeadIDByName(
	c *gin.Context,
	name string,
) (string, error) {
	var estadoID string

	err := h.DB.QueryRow(
		c.Request.Context(),
		`
			SELECT id::text
			FROM estado_leads
			WHERE LOWER(TRIM(nombre)) = LOWER(TRIM($1))
			LIMIT 1
		`,
		name,
	).Scan(&estadoID)

	return estadoID, err
}

const leadSelectQuery = `
	SELECT
		l.id::text,

		COALESCE(l.asesor_id::text, ''),
		COALESCE(l.estado_lead_id::text, ''),
		COALESCE(l.proyecto_id::text, ''),
		COALESCE(l.campania_id::text, ''),

		COALESCE(l.nombres_completos, ''),
		COALESCE(l.tipo_documento, ''),
		COALESCE(l.numero_documento, ''),
		COALESCE(l.telefono, ''),
		COALESCE(l.email, ''),
		COALESCE(l.mensaje, ''),

		COALESCE(l.proyecto_interes, ''),
		COALESCE(l.categoria_interes, ''),
		COALESCE(l.fuente_prospeccion, ''),
		COALESCE(l.origen_ruta, ''),
		COALESCE(l.origen_componente, ''),

		COALESCE(l.activo, TRUE),

		COALESCE(e.nombre, ''),
		COALESCE(a.nombres_completos, ''),
		COALESCE(p.nombre, l.proyecto_interes, ''),
		COALESCE(ca.nombre, ''),

		COALESCE(l.created_at::text, ''),
		COALESCE(l.updated_at::text, '')

	FROM leads l

	LEFT JOIN estado_leads e
		ON e.id = l.estado_lead_id

	LEFT JOIN asesores a
		ON a.id = l.asesor_id

	LEFT JOIN proyectos p
		ON p.id = l.proyecto_id

	LEFT JOIN campanias ca
		ON ca.id = l.campania_id

	WHERE l.deleted_at IS NULL
`

type leadScanner interface {
	Scan(dest ...any) error
}

func scanLeadRow(row leadScanner) (LeadResponse, error) {
	var lead LeadResponse

	err := row.Scan(
		&lead.ID,

		&lead.AsesorID,
		&lead.EstadoLeadID,
		&lead.ProyectoID,
		&lead.CampaniaID,

		&lead.NombresCompletos,
		&lead.TipoDocumento,
		&lead.NumeroDocumento,
		&lead.Telefono,
		&lead.Email,
		&lead.Mensaje,

		&lead.ProyectoInteres,
		&lead.CategoriaInteres,
		&lead.FuenteProspeccion,
		&lead.OrigenRuta,
		&lead.OrigenComponente,

		&lead.Activo,

		&lead.EstadoLead,
		&lead.Asesor,
		&lead.Proyecto,
		&lead.Campania,

		&lead.CreatedAt,
		&lead.UpdatedAt,
	)

	return lead, err
}

func scanLead(rows pgx.Rows) (LeadResponse, error) {
	return scanLeadRow(rows)
}

func optionalString(value *string) any {
	if value == nil {
		return nil
	}

	return strings.TrimSpace(*value)
}

func optionalBool(value *bool) any {
	if value == nil {
		return nil
	}

	return *value
}

func errorData(err error) any {
	if err == nil {
		return nil
	}

	return gin.H{
		"error": err.Error(),
	}
}
