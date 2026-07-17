package handlers

import (
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type LeadHandler struct {
	DB *pgxpool.Pool
}

func NewLeadHandler(
	db *pgxpool.Pool,
) *LeadHandler {
	return &LeadHandler{
		DB: db,
	}
}

type leadScanner interface {
	Scan(dest ...any) error
}

var uuidPattern = regexp.MustCompile(
	`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[1-5][0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12}$`,
)

/*
|--------------------------------------------------------------------------
| REQUESTS
|--------------------------------------------------------------------------
*/

type CreateLeadRequest struct {
	FullName         string `json:"fullName"`
	NombresCompletos string `json:"nombres_completos"`

	Phone    string `json:"phone"`
	Telefono string `json:"telefono"`

	Email string `json:"email"`

	Message string `json:"message"`
	Mensaje string `json:"mensaje"`

	Project         string `json:"project"`
	ProyectoInteres string `json:"proyecto_interes"`

	Interest         string `json:"interest"`
	CategoriaInteres string `json:"categoria_interes"`

	ProjectID  string `json:"projectId"`
	ProyectoID string `json:"proyecto_id"`

	TipoDocumento   string `json:"tipo_documento"`
	NumeroDocumento string `json:"numero_documento"`

	Campaign       string `json:"campaign"`
	CampaniaSlug   string `json:"campania_slug"`
	CampaniaNombre string `json:"campania_nombre"`

	Source             string `json:"source"`
	FuenteProspeccion  string `json:"fuente_prospeccion"`
	OrigenRuta         string `json:"origen_ruta"`
	OrigenComponente   string `json:"origen_componente"`
}

type UpdateLeadRequest struct {
	FullName         *string `json:"fullName"`
	NombresCompletos *string `json:"nombres_completos"`

	Phone    *string `json:"phone"`
	Telefono *string `json:"telefono"`

	Email *string `json:"email"`

	Message *string `json:"message"`
	Mensaje *string `json:"mensaje"`

	Project         *string `json:"project"`
	ProyectoInteres *string `json:"proyecto_interes"`

	Interest         *string `json:"interest"`
	CategoriaInteres *string `json:"categoria_interes"`

	ProjectID  *string `json:"projectId"`
	ProyectoID *string `json:"proyecto_id"`

	TipoDocumento   *string `json:"tipo_documento"`
	NumeroDocumento *string `json:"numero_documento"`

	EstadoLeadID *string `json:"estado_lead_id"`
	AsesorID     *string `json:"asesor_id"`
	CampaniaID   *string `json:"campania_id"`

	Source            *string `json:"source"`
	FuenteProspeccion *string `json:"fuente_prospeccion"`
	OrigenRuta        *string `json:"origen_ruta"`
	OrigenComponente  *string `json:"origen_componente"`

	Atendido *bool `json:"atendido"`
	Activo   *bool `json:"activo"`
}

/*
|--------------------------------------------------------------------------
| RESPUESTA
|--------------------------------------------------------------------------
*/

type LeadResponse struct {
	ID string `json:"id"`

	AsesorID     string `json:"asesor_id"`
	EstadoLeadID string `json:"estado_lead_id"`
	ProyectoID   string `json:"proyecto_id"`
	CampaniaID   string `json:"campania_id"`

	NombresCompletos string `json:"nombres_completos"`
	TipoDocumento    string `json:"tipo_documento"`
	NumeroDocumento  string `json:"numero_documento"`

	ProyectoInteres string `json:"proyecto_interes"`
	Telefono        string `json:"telefono"`
	Email           string `json:"email"`
	Mensaje         string `json:"mensaje"`

	CategoriaInteres  string `json:"categoria_interes"`
	FuenteProspeccion string `json:"fuente_prospeccion"`

	OrigenRuta       string `json:"origen_ruta"`
	OrigenComponente string `json:"origen_componente"`

	Atendido bool `json:"atendido"`
	Activo   bool `json:"activo"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	Proyecto       string `json:"proyecto"`
	TipoProyecto   string `json:"tipo_proyecto"`
	Ubicacion      string `json:"ubicacion"`
	EstadoLead     string `json:"estado_lead"`
	Asesor         string `json:"asesor"`
	CampaniaNombre string `json:"campania_nombre"`
	CampaniaSlug   string `json:"campania_slug"`
}

/*
|--------------------------------------------------------------------------
| CONSULTA BASE
|--------------------------------------------------------------------------
*/

const leadSelectQuery = `
	SELECT
		l.id::text,

		COALESCE(l.asesor_id::text, ''),
		l.estado_lead_id::text,
		COALESCE(l.proyecto_id::text, ''),
		COALESCE(l.campania_id::text, ''),

		l.nombres_completos,
		COALESCE(l.tipo_documento, ''),
		COALESCE(l.numero_documento, ''),

		COALESCE(l.proyecto_interes, ''),
		l.telefono,
		COALESCE(l.email, ''),
		COALESCE(l.mensaje, ''),

		COALESCE(l.categoria_interes, ''),
		COALESCE(l.fuente_prospeccion, ''),

		COALESCE(l.origen_ruta, ''),
		COALESCE(l.origen_componente, ''),

		l.atendido,
		l.activo,

		l.created_at,
		l.updated_at,

		COALESCE(
			p.nombre,
			l.proyecto_interes,
			''
		),

		COALESCE(p.tipo, ''),
		COALESCE(p.ubicacion, ''),

		e.nombre,

		COALESCE(a.nombres_completos, ''),

		COALESCE(c.nombre, ''),
		COALESCE(c.slug, '')

	FROM leads l

	LEFT JOIN proyectos p
		ON p.id = l.proyecto_id

	INNER JOIN estado_leads e
		ON e.id = l.estado_lead_id

	LEFT JOIN asesores a
		ON a.id = l.asesor_id

	LEFT JOIN campanias c
		ON c.id = l.campania_id

	WHERE l.deleted_at IS NULL
`

func scanLead(
	scanner leadScanner,
) (LeadResponse, error) {
	var lead LeadResponse

	err := scanner.Scan(
		&lead.ID,

		&lead.AsesorID,
		&lead.EstadoLeadID,
		&lead.ProyectoID,
		&lead.CampaniaID,

		&lead.NombresCompletos,
		&lead.TipoDocumento,
		&lead.NumeroDocumento,

		&lead.ProyectoInteres,
		&lead.Telefono,
		&lead.Email,
		&lead.Mensaje,

		&lead.CategoriaInteres,
		&lead.FuenteProspeccion,

		&lead.OrigenRuta,
		&lead.OrigenComponente,

		&lead.Atendido,
		&lead.Activo,

		&lead.CreatedAt,
		&lead.UpdatedAt,

		&lead.Proyecto,
		&lead.TipoProyecto,
		&lead.Ubicacion,
		&lead.EstadoLead,
		&lead.Asesor,
		&lead.CampaniaNombre,
		&lead.CampaniaSlug,
	)

	return lead, err
}

/*
|--------------------------------------------------------------------------
| CREATE
|--------------------------------------------------------------------------
*/

func (h *LeadHandler) CreateLead(
	c *gin.Context,
) {
	var body CreateLeadRequest

	if err := c.ShouldBindJSON(&body); err != nil {
		leadError(
			c,
			http.StatusBadRequest,
			"lead.invalid_json",
			"El JSON enviado no es válido.",
			err,
		)
		return
	}

	nombresCompletos := leadFirstValue(
		body.NombresCompletos,
		body.FullName,
	)

	telefono := leadDigits(
		leadFirstValue(
			body.Telefono,
			body.Phone,
		),
	)

	email := strings.ToLower(
		strings.TrimSpace(body.Email),
	)

	mensaje := leadFirstValue(
		body.Mensaje,
		body.Message,
	)

	proyectoInteres := leadFirstValue(
		body.ProyectoInteres,
		body.Project,
	)

	categoriaInteres := leadFirstValue(
		body.CategoriaInteres,
		body.Interest,
	)

	proyectoID := leadFirstValue(
		body.ProyectoID,
		body.ProjectID,
	)

	tipoDocumento := strings.TrimSpace(
		body.TipoDocumento,
	)

	numeroDocumento := leadDigits(
		body.NumeroDocumento,
	)

	campaniaSlug := leadFirstValue(
		body.CampaniaSlug,
		body.Campaign,
		"formulario-web-general",
	)

	campaniaNombre := leadFirstValue(
		body.CampaniaNombre,
		campaniaSlug,
	)

	fuenteProspeccion := leadFirstValue(
		body.FuenteProspeccion,
		body.Source,
		"Web",
	)

	origenRuta := leadFirstValue(
		body.OrigenRuta,
		c.GetHeader("Referer"),
	)

	origenComponente := leadFirstValue(
		body.OrigenComponente,
		"Formulario web",
	)

	if nombresCompletos == "" {
		leadError(
			c,
			http.StatusBadRequest,
			"lead.validation_error",
			"El nombre completo es obligatorio.",
			nil,
		)
		return
	}

	if len(telefono) != 9 ||
		!strings.HasPrefix(telefono, "9") {
		leadError(
			c,
			http.StatusBadRequest,
			"lead.validation_error",
			"El celular debe tener 9 dígitos y empezar con 9.",
			nil,
		)
		return
	}

	if proyectoID != "" &&
		!isValidUUID(proyectoID) {
		leadError(
			c,
			http.StatusBadRequest,
			"lead.validation_error",
			"El proyecto_id no es un UUID válido.",
			nil,
		)
		return
	}

	if tipoDocumento == "" &&
		numeroDocumento != "" {
		tipoDocumento = "DNI"
	}

	if tipoDocumento == "DNI" &&
		numeroDocumento != "" &&
		len(numeroDocumento) != 8 {
		leadError(
			c,
			http.StatusBadRequest,
			"lead.validation_error",
			"El DNI debe tener 8 dígitos.",
			nil,
		)
		return
	}

	ctx := c.Request.Context()

	var leadID string

	err := h.DB.QueryRow(
		ctx,
		`
			WITH campania_actual AS (
				INSERT INTO campanias (
					nombre,
					slug,
					activo
				)
				VALUES (
					$1,
					$2,
					TRUE
				)
				ON CONFLICT (slug)
				DO UPDATE SET
					nombre = EXCLUDED.nombre,
					activo = TRUE,
					updated_at = NOW()
				RETURNING id
			),

			proyecto_actual AS (
				SELECT
					p.id,
					p.nombre
				FROM proyectos p
				WHERE
					(
						NULLIF($3, '') IS NOT NULL
						AND p.id::text = NULLIF($3, '')
					)
					OR
					(
						NULLIF($4, '') IS NOT NULL
						AND LOWER(TRIM(p.nombre)) =
							LOWER(TRIM($4))
					)
				ORDER BY
					CASE
						WHEN p.id::text =
							NULLIF($3, '')
						THEN 0
						ELSE 1
					END
				LIMIT 1
			)

			INSERT INTO leads (
				estado_lead_id,
				proyecto_id,
				campania_id,

				nombres_completos,
				tipo_documento,
				numero_documento,

				proyecto_interes,
				telefono,
				email,
				mensaje,

				categoria_interes,
				fuente_prospeccion,

				origen_ruta,
				origen_componente,

				atendido,
				activo
			)
			VALUES (
				COALESCE(
					(
						SELECT id
						FROM estado_leads
						WHERE LOWER(nombre) =
							LOWER('Contacto inicial del cliente')
						LIMIT 1
					),
					(
						SELECT id
						FROM estado_leads
						WHERE LOWER(nombre) =
							LOWER('Nuevo')
						LIMIT 1
					),
					(
						SELECT id
						FROM estado_leads
						ORDER BY nombre
						LIMIT 1
					)
				),

				(
					SELECT id
					FROM proyecto_actual
					LIMIT 1
				),

				(
					SELECT id
					FROM campania_actual
					LIMIT 1
				),

				$5,
				NULLIF($6, ''),
				NULLIF($7, ''),

				COALESCE(
					NULLIF($4, ''),
					(
						SELECT nombre
						FROM proyecto_actual
						LIMIT 1
					)
				),

				$8,
				NULLIF($9, ''),
				NULLIF($10, ''),

				NULLIF($11, ''),
				$12,

				NULLIF($13, ''),
				NULLIF($14, ''),

				FALSE,
				TRUE
			)
			RETURNING id::text
		`,
		campaniaNombre,      // $1
		campaniaSlug,        // $2
		proyectoID,          // $3
		proyectoInteres,     // $4
		nombresCompletos,    // $5
		tipoDocumento,       // $6
		numeroDocumento,     // $7
		telefono,            // $8
		email,               // $9
		mensaje,             // $10
		categoriaInteres,    // $11
		fuenteProspeccion,   // $12
		origenRuta,          // $13
		origenComponente,    // $14
	).Scan(&leadID)

	if err != nil {
		leadError(
			c,
			http.StatusInternalServerError,
			"lead.create_error",
			"No se pudo registrar el lead.",
			err,
		)
		return
	}

	lead, err := scanLead(
		h.DB.QueryRow(
			ctx,
			leadSelectQuery+
				`
					AND l.id = $1::uuid
					LIMIT 1
				`,
			leadID,
		),
	)

	if err != nil {
		leadError(
			c,
			http.StatusInternalServerError,
			"lead.read_error",
			"El lead fue creado, pero no se pudo consultar.",
			err,
		)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success":  true,
		"response": "lead.created",
		"message":  "Datos enviados correctamente. Un asesor se comunicará contigo.",
		"data": gin.H{
			"lead": lead,
		},
	})
}

/*
|--------------------------------------------------------------------------
| READ ALL
|--------------------------------------------------------------------------
*/

func (h *LeadHandler) GetAllLeads(
	c *gin.Context,
) {
	rows, err := h.DB.Query(
		c.Request.Context(),
		leadSelectQuery+
			`
				ORDER BY l.created_at DESC
			`,
	)

	if err != nil {
		leadError(
			c,
			http.StatusInternalServerError,
			"lead.list_error",
			"No se pudieron consultar los leads.",
			err,
		)
		return
	}

	defer rows.Close()

	leads := []LeadResponse{}

	for rows.Next() {
		lead, scanErr := scanLead(rows)

		if scanErr != nil {
			leadError(
				c,
				http.StatusInternalServerError,
				"lead.scan_error",
				"No se pudo leer un lead.",
				scanErr,
			)
			return
		}

		leads = append(leads, lead)
	}

	if err := rows.Err(); err != nil {
		leadError(
			c,
			http.StatusInternalServerError,
			"lead.rows_error",
			"No se pudo completar la consulta.",
			err,
		)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":  true,
		"response": "lead.list",
		"message":  "Leads obtenidos correctamente.",
		"data": gin.H{
			"total": len(leads),
			"leads": leads,
		},
	})
}

/*
|--------------------------------------------------------------------------
| READ BY ID
|--------------------------------------------------------------------------
*/

func (h *LeadHandler) GetLeadByID(
	c *gin.Context,
) {
	leadID := strings.TrimSpace(
		c.Param("id"),
	)

	if !isValidUUID(leadID) {
		leadError(
			c,
			http.StatusBadRequest,
			"lead.invalid_id",
			"El ID del lead no es válido.",
			nil,
		)
		return
	}

	lead, err := scanLead(
		h.DB.QueryRow(
			c.Request.Context(),
			leadSelectQuery+
				`
					AND l.id = $1::uuid
					LIMIT 1
				`,
			leadID,
		),
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			leadError(
				c,
				http.StatusNotFound,
				"lead.not_found",
				"El lead no existe.",
				nil,
			)
			return
		}

		leadError(
			c,
			http.StatusInternalServerError,
			"lead.read_error",
			"No se pudo consultar el lead.",
			err,
		)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":  true,
		"response": "lead.detail",
		"message":  "Lead obtenido correctamente.",
		"data": gin.H{
			"lead": lead,
		},
	})
}

/*
|--------------------------------------------------------------------------
| UPDATE
|--------------------------------------------------------------------------
*/

func (h *LeadHandler) UpdateLead(
	c *gin.Context,
) {
	leadID := strings.TrimSpace(
		c.Param("id"),
	)

	if !isValidUUID(leadID) {
		leadError(
			c,
			http.StatusBadRequest,
			"lead.invalid_id",
			"El ID del lead no es válido.",
			nil,
		)
		return
	}

	var body UpdateLeadRequest

	if err := c.ShouldBindJSON(&body); err != nil {
		leadError(
			c,
			http.StatusBadRequest,
			"lead.invalid_json",
			"Los datos enviados no son válidos.",
			err,
		)
		return
	}

	nameProvided, nombresCompletos :=
		leadOptionalString(
			body.NombresCompletos,
			body.FullName,
		)

	phoneProvided, telefono :=
		leadOptionalDigits(
			body.Telefono,
			body.Phone,
		)

	emailProvided, email :=
		leadOptionalLower(
			body.Email,
		)

	messageProvided, mensaje :=
		leadOptionalString(
			body.Mensaje,
			body.Message,
		)

	projectProvided, proyectoInteres :=
		leadOptionalString(
			body.ProyectoInteres,
			body.Project,
		)

	categoryProvided, categoriaInteres :=
		leadOptionalString(
			body.CategoriaInteres,
			body.Interest,
		)

	projectIDProvided, proyectoID :=
		leadOptionalString(
			body.ProyectoID,
			body.ProjectID,
		)

	typeDocumentProvided, tipoDocumento :=
		leadOptionalString(
			body.TipoDocumento,
		)

	documentProvided, numeroDocumento :=
		leadOptionalDigits(
			body.NumeroDocumento,
		)

	sourceProvided, fuenteProspeccion :=
		leadOptionalString(
			body.FuenteProspeccion,
			body.Source,
		)

	routeProvided, origenRuta :=
		leadOptionalString(
			body.OrigenRuta,
		)

	componentProvided, origenComponente :=
		leadOptionalString(
			body.OrigenComponente,
		)

	stateIDProvided, estadoLeadID :=
		leadOptionalString(
			body.EstadoLeadID,
		)

	advisorIDProvided, asesorID :=
		leadOptionalString(
			body.AsesorID,
		)

	campaignIDProvided, campaniaID :=
		leadOptionalString(
			body.CampaniaID,
		)

	if nameProvided &&
		nombresCompletos == "" {
		leadError(
			c,
			http.StatusBadRequest,
			"lead.validation_error",
			"El nombre no puede estar vacío.",
			nil,
		)
		return
	}

	if phoneProvided &&
		(len(telefono) != 9 ||
			!strings.HasPrefix(telefono, "9")) {
		leadError(
			c,
			http.StatusBadRequest,
			"lead.validation_error",
			"El celular debe tener 9 dígitos y empezar con 9.",
			nil,
		)
		return
	}

	if documentProvided &&
		tipoDocumento == "DNI" &&
		numeroDocumento != "" &&
		len(numeroDocumento) != 8 {
		leadError(
			c,
			http.StatusBadRequest,
			"lead.validation_error",
			"El DNI debe tener 8 dígitos.",
			nil,
		)
		return
	}

	if projectIDProvided &&
		proyectoID != "" &&
		!isValidUUID(proyectoID) {
		leadError(
			c,
			http.StatusBadRequest,
			"lead.validation_error",
			"El proyecto_id no es válido.",
			nil,
		)
		return
	}

	if campaignIDProvided &&
		campaniaID != "" &&
		!isValidUUID(campaniaID) {
		leadError(
			c,
			http.StatusBadRequest,
			"lead.validation_error",
			"El campania_id no es válido.",
			nil,
		)
		return
	}

	if advisorIDProvided &&
		asesorID != "" &&
		!isValidUUID(asesorID) {
		leadError(
			c,
			http.StatusBadRequest,
			"lead.validation_error",
			"El asesor_id no es válido.",
			nil,
		)
		return
	}

	if stateIDProvided &&
		(estadoLeadID == "" ||
			!isValidUUID(estadoLeadID)) {
		leadError(
			c,
			http.StatusBadRequest,
			"lead.validation_error",
			"El estado_lead_id no es válido.",
			nil,
		)
		return
	}

	hasChanges :=
		nameProvided ||
			phoneProvided ||
			emailProvided ||
			messageProvided ||
			projectProvided ||
			categoryProvided ||
			projectIDProvided ||
			typeDocumentProvided ||
			documentProvided ||
			sourceProvided ||
			routeProvided ||
			componentProvided ||
			stateIDProvided ||
			advisorIDProvided ||
			campaignIDProvided ||
			body.Atendido != nil ||
			body.Activo != nil

	if !hasChanges {
		leadError(
			c,
			http.StatusBadRequest,
			"lead.no_changes",
			"No se enviaron campos para actualizar.",
			nil,
		)
		return
	}

	result, err := h.DB.Exec(
		c.Request.Context(),
		`
			UPDATE leads
			SET
				nombres_completos =
					CASE
						WHEN $2::boolean
						THEN $3
						ELSE nombres_completos
					END,

				tipo_documento =
					CASE
						WHEN $4::boolean
						THEN NULLIF($5, '')
						ELSE tipo_documento
					END,

				numero_documento =
					CASE
						WHEN $6::boolean
						THEN NULLIF($7, '')
						ELSE numero_documento
					END,

				telefono =
					CASE
						WHEN $8::boolean
						THEN $9
						ELSE telefono
					END,

				email =
					CASE
						WHEN $10::boolean
						THEN NULLIF($11, '')
						ELSE email
					END,

				mensaje =
					CASE
						WHEN $12::boolean
						THEN NULLIF($13, '')
						ELSE mensaje
					END,

				proyecto_interes =
					CASE
						WHEN $14::boolean
						THEN NULLIF($15, '')
						ELSE proyecto_interes
					END,

				categoria_interes =
					CASE
						WHEN $16::boolean
						THEN NULLIF($17, '')
						ELSE categoria_interes
					END,

				fuente_prospeccion =
					CASE
						WHEN $18::boolean
						THEN $19
						ELSE fuente_prospeccion
					END,

				origen_ruta =
					CASE
						WHEN $20::boolean
						THEN NULLIF($21, '')
						ELSE origen_ruta
					END,

				origen_componente =
					CASE
						WHEN $22::boolean
						THEN NULLIF($23, '')
						ELSE origen_componente
					END,

				atendido =
					CASE
						WHEN $24::boolean
						THEN $25
						ELSE atendido
					END,

				activo =
					CASE
						WHEN $26::boolean
						THEN $27
						ELSE activo
					END,

				proyecto_id =
					CASE
						WHEN $28::boolean
							THEN NULLIF($29, '')::uuid

						WHEN $14::boolean
							THEN (
								SELECT id
								FROM proyectos
								WHERE LOWER(TRIM(nombre)) =
									LOWER(TRIM($15))
								  AND activo = TRUE
								LIMIT 1
							)

						ELSE proyecto_id
					END,

				campania_id =
					CASE
						WHEN $30::boolean
							THEN NULLIF($31, '')::uuid
						ELSE campania_id
					END,

				estado_lead_id =
					CASE
						WHEN $32::boolean
							THEN $33::uuid
						ELSE estado_lead_id
					END,

				asesor_id =
					CASE
						WHEN $34::boolean
							THEN NULLIF($35, '')::uuid
						ELSE asesor_id
					END,

				updated_at = NOW()

			WHERE id = $1::uuid
			  AND deleted_at IS NULL
		`,
		leadID, // $1

		nameProvided,     // $2
		nombresCompletos, // $3

		typeDocumentProvided, // $4
		tipoDocumento,        // $5

		documentProvided, // $6
		numeroDocumento, // $7

		phoneProvided, // $8
		telefono,      // $9

		emailProvided, // $10
		email,         // $11

		messageProvided, // $12
		mensaje,         // $13

		projectProvided,  // $14
		proyectoInteres,  // $15

		categoryProvided,  // $16
		categoriaInteres,  // $17

		sourceProvided,     // $18
		fuenteProspeccion,  // $19

		routeProvided, // $20
		origenRuta,    // $21

		componentProvided, // $22
		origenComponente,  // $23

		body.Atendido != nil, // $24
		boolValue(body.Atendido), // $25

		body.Activo != nil, // $26
		boolValue(body.Activo), // $27

		projectIDProvided, // $28
		proyectoID,        // $29

		campaignIDProvided, // $30
		campaniaID,         // $31

		stateIDProvided, // $32
		estadoLeadID,    // $33

		advisorIDProvided, // $34
		asesorID,          // $35
	)

	if err != nil {
		leadError(
			c,
			http.StatusInternalServerError,
			"lead.update_error",
			"No se pudo actualizar el lead.",
			err,
		)
		return
	}

	if result.RowsAffected() == 0 {
		leadError(
			c,
			http.StatusNotFound,
			"lead.not_found",
			"El lead no existe.",
			nil,
		)
		return
	}

	lead, err := scanLead(
		h.DB.QueryRow(
			c.Request.Context(),
			leadSelectQuery+
				`
					AND l.id = $1::uuid
					LIMIT 1
				`,
			leadID,
		),
	)

	if err != nil {
		leadError(
			c,
			http.StatusInternalServerError,
			"lead.read_error",
			"El lead se actualizó, pero no se pudo consultar.",
			err,
		)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":  true,
		"response": "lead.updated",
		"message":  "Lead actualizado correctamente.",
		"data": gin.H{
			"lead": lead,
		},
	})
}

/*
|--------------------------------------------------------------------------
| DELETE LÓGICO
|--------------------------------------------------------------------------
*/

func (h *LeadHandler) DeleteLead(
	c *gin.Context,
) {
	leadID := strings.TrimSpace(
		c.Param("id"),
	)

	if !isValidUUID(leadID) {
		leadError(
			c,
			http.StatusBadRequest,
			"lead.invalid_id",
			"El ID del lead no es válido.",
			nil,
		)
		return
	}

	var deletedID string

	err := h.DB.QueryRow(
		c.Request.Context(),
		`
			UPDATE leads
			SET
				activo = FALSE,
				deleted_at = NOW(),
				updated_at = NOW()
			WHERE id = $1::uuid
			  AND deleted_at IS NULL
			RETURNING id::text
		`,
		leadID,
	).Scan(&deletedID)

	if err != nil {
		if err == pgx.ErrNoRows {
			leadError(
				c,
				http.StatusNotFound,
				"lead.not_found",
				"El lead no existe.",
				nil,
			)
			return
		}

		leadError(
			c,
			http.StatusInternalServerError,
			"lead.delete_error",
			"No se pudo eliminar el lead.",
			err,
		)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":  true,
		"response": "lead.deleted",
		"message":  "Lead eliminado correctamente.",
		"data": gin.H{
			"lead_id": deletedID,
		},
	})
}

/*
|--------------------------------------------------------------------------
| FUNCIONES AUXILIARES
|--------------------------------------------------------------------------
*/

func leadFirstValue(
	values ...string,
) string {
	for _, value := range values {
		cleanValue := strings.TrimSpace(
			value,
		)

		if cleanValue != "" {
			return cleanValue
		}
	}

	return ""
}

func leadDigits(
	value string,
) string {
	var result strings.Builder

	for _, character := range value {
		if character >= '0' &&
			character <= '9' {
			result.WriteRune(character)
		}
	}

	return result.String()
}

func leadOptionalString(
	values ...*string,
) (bool, string) {
	for _, value := range values {
		if value != nil {
			return true, strings.TrimSpace(
				*value,
			)
		}
	}

	return false, ""
}

func leadOptionalLower(
	values ...*string,
) (bool, string) {
	provided, value :=
		leadOptionalString(values...)

	if !provided {
		return false, ""
	}

	return true, strings.ToLower(value)
}

func leadOptionalDigits(
	values ...*string,
) (bool, string) {
	for _, value := range values {
		if value != nil {
			return true, leadDigits(*value)
		}
	}

	return false, ""
}

func boolValue(
	value *bool,
) bool {
	if value == nil {
		return false
	}

	return *value
}

func isValidUUID(
	value string,
) bool {
	return uuidPattern.MatchString(
		strings.TrimSpace(value),
	)
}

func leadError(
	c *gin.Context,
	status int,
	response string,
	message string,
	err error,
) {
	var data any = nil

	if err != nil {
		data = gin.H{
			"error": err.Error(),
		}
	}

	c.JSON(status, gin.H{
		"success":  false,
		"response": response,
		"message":  message,
		"data":     data,
	})
}