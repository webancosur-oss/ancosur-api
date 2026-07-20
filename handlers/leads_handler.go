package handlers

import (
	"fmt"
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

	Source            string `json:"source"`
	FuenteProspeccion string `json:"fuente_prospeccion"`
	OrigenRuta        string `json:"origen_ruta"`
	OrigenComponente  string `json:"origen_componente"`

	EtapaEmbudo string   `json:"etapa_embudo"`
	Lead        string   `json:"lead"`
	FechaVenta  string   `json:"fecha_venta"`
	MontoVenta  *float64 `json:"monto_venta"`
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

	EtapaEmbudo *string  `json:"etapa_embudo"`
	Lead        *string  `json:"lead"`
	FechaVenta  *string  `json:"fecha_venta"`
	MontoVenta  *float64 `json:"monto_venta"`

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

	EtapaEmbudo string `json:"etapa_embudo"`
	Lead        string `json:"lead"`

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

	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at"`

	FechaVenta *time.Time `json:"fecha_venta"`
	MontoVenta *float64   `json:"monto_venta"`

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
		COALESCE(l.estado_lead_id::text, ''),
		COALESCE(l.proyecto_id::text, ''),
		COALESCE(l.campania_id::text, ''),

		COALESCE(l.etapa_embudo, ''),
		COALESCE(l.lead, ''),

		COALESCE(l.nombres_completos, ''),
		COALESCE(l.tipo_documento, ''),
		COALESCE(l.numero_documento, ''),

		COALESCE(l.proyecto_interes, ''),
		COALESCE(l.telefono, ''),
		COALESCE(l.email, ''),
		COALESCE(l.mensaje, ''),

		COALESCE(l.categoria_interes, ''),
		COALESCE(l.fuente_prospeccion, ''),

		COALESCE(l.origen_ruta, ''),
		COALESCE(l.origen_componente, ''),

		COALESCE(l.atendido, FALSE),
		COALESCE(l.activo, TRUE),

		COALESCE(l.created_at, NOW()),
		COALESCE(l.updated_at, l.created_at, NOW()),
		l.deleted_at,

		l.fecha_venta,
		l.monto_venta,

		COALESCE(
			p.nombre,
			l.proyecto_interes,
			''
		),

		COALESCE(p.tipo, ''),
		COALESCE(p.ubicacion, ''),

		COALESCE(e.nombre, ''),

		COALESCE(
			a.nombres_completos,
			''
		),

		COALESCE(c.nombre, ''),
		COALESCE(c.slug, '')

	FROM leads l

	LEFT JOIN proyectos p
		ON p.id = l.proyecto_id

	LEFT JOIN estado_leads e
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

		&lead.EtapaEmbudo,
		&lead.Lead,

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
		&lead.DeletedAt,

		&lead.FechaVenta,
		&lead.MontoVenta,

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

	etapaEmbudo := strings.TrimSpace(
		body.EtapaEmbudo,
	)

	leadTipo := strings.TrimSpace(
		body.Lead,
	)

	fechaVenta := strings.TrimSpace(
		body.FechaVenta,
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

	if fechaVenta != "" &&
		!isValidDate(fechaVenta) {
		leadError(
			c,
			http.StatusBadRequest,
			"lead.validation_error",
			"La fecha_venta debe tener el formato YYYY-MM-DD.",
			nil,
		)
		return
	}

	if body.MontoVenta != nil &&
		*body.MontoVenta < 0 {
		leadError(
			c,
			http.StatusBadRequest,
			"lead.validation_error",
			"El monto_venta no puede ser negativo.",
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

				etapa_embudo,
				lead,

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

				fecha_venta,
				monto_venta,

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

				NULLIF($15, ''),
				NULLIF($16, ''),

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

				NULLIF($17, '')::date,
				$18,

				FALSE,
				TRUE
			)
			RETURNING id::text
		`,
		campaniaNombre,    // $1
		campaniaSlug,      // $2
		proyectoID,        // $3
		proyectoInteres,   // $4
		nombresCompletos,  // $5
		tipoDocumento,     // $6
		numeroDocumento,   // $7
		telefono,          // $8
		email,             // $9
		mensaje,           // $10
		categoriaInteres,  // $11
		fuenteProspeccion, // $12
		origenRuta,        // $13
		origenComponente,  // $14
		etapaEmbudo,       // $15
		leadTipo,          // $16
		fechaVenta,        // $17
		body.MontoVenta,   // $18
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

	stageProvided, etapaEmbudo :=
		leadOptionalString(
			body.EtapaEmbudo,
		)

	leadTypeProvided, leadTipo :=
		leadOptionalString(
			body.Lead,
		)

	saleDateProvided, fechaVenta :=
		leadOptionalString(
			body.FechaVenta,
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

	if saleDateProvided &&
		fechaVenta != "" &&
		!isValidDate(fechaVenta) {
		leadError(
			c,
			http.StatusBadRequest,
			"lead.validation_error",
			"La fecha_venta debe tener el formato YYYY-MM-DD.",
			nil,
		)
		return
	}

	if body.MontoVenta != nil &&
		*body.MontoVenta < 0 {
		leadError(
			c,
			http.StatusBadRequest,
			"lead.validation_error",
			"El monto_venta no puede ser negativo.",
			nil,
		)
		return
	}

	setClauses := []string{}
	args := []any{
		leadID,
	}

	addArg := func(value any) string {
		args = append(args, value)

		return fmt.Sprintf(
			"$%d",
			len(args),
		)
	}

	addStringSet := func(
		column string,
		provided bool,
		value string,
		nullable bool,
	) {
		if !provided {
			return
		}

		param := addArg(value)

		if nullable {
			setClauses = append(
				setClauses,
				fmt.Sprintf(
					"%s = NULLIF(%s, '')",
					column,
					param,
				),
			)

			return
		}

		setClauses = append(
			setClauses,
			fmt.Sprintf(
				"%s = %s",
				column,
				param,
			),
		)
	}

	addUUIDSet := func(
		column string,
		provided bool,
		value string,
		nullable bool,
	) {
		if !provided {
			return
		}

		param := addArg(value)

		if nullable {
			setClauses = append(
				setClauses,
				fmt.Sprintf(
					"%s = NULLIF(%s, '')::uuid",
					column,
					param,
				),
			)

			return
		}

		setClauses = append(
			setClauses,
			fmt.Sprintf(
				"%s = %s::uuid",
				column,
				param,
			),
		)
	}

	addStringSet(
		"nombres_completos",
		nameProvided,
		nombresCompletos,
		false,
	)

	addStringSet(
		"telefono",
		phoneProvided,
		telefono,
		false,
	)

	addStringSet(
		"email",
		emailProvided,
		email,
		true,
	)

	addStringSet(
		"mensaje",
		messageProvided,
		mensaje,
		true,
	)

	addStringSet(
		"proyecto_interes",
		projectProvided,
		proyectoInteres,
		true,
	)

	addStringSet(
		"categoria_interes",
		categoryProvided,
		categoriaInteres,
		true,
	)

	addStringSet(
		"tipo_documento",
		typeDocumentProvided,
		tipoDocumento,
		true,
	)

	addStringSet(
		"numero_documento",
		documentProvided,
		numeroDocumento,
		true,
	)

	addStringSet(
		"fuente_prospeccion",
		sourceProvided,
		fuenteProspeccion,
		true,
	)

	addStringSet(
		"origen_ruta",
		routeProvided,
		origenRuta,
		true,
	)

	addStringSet(
		"origen_componente",
		componentProvided,
		origenComponente,
		true,
	)

	addStringSet(
		"etapa_embudo",
		stageProvided,
		etapaEmbudo,
		true,
	)

	addStringSet(
		"lead",
		leadTypeProvided,
		leadTipo,
		true,
	)

	if saleDateProvided {
		param := addArg(fechaVenta)

		setClauses = append(
			setClauses,
			fmt.Sprintf(
				"fecha_venta = NULLIF(%s, '')::date",
				param,
			),
		)
	}

	if body.MontoVenta != nil {
		param := addArg(
			*body.MontoVenta,
		)

		setClauses = append(
			setClauses,
			fmt.Sprintf(
				"monto_venta = %s",
				param,
			),
		)
	}

	if body.Atendido != nil {
		param := addArg(
			*body.Atendido,
		)

		setClauses = append(
			setClauses,
			fmt.Sprintf(
				"atendido = %s",
				param,
			),
		)
	}

	if body.Activo != nil {
		param := addArg(
			*body.Activo,
		)

		setClauses = append(
			setClauses,
			fmt.Sprintf(
				"activo = %s",
				param,
			),
		)
	}

	addUUIDSet(
		"proyecto_id",
		projectIDProvided,
		proyectoID,
		true,
	)

	addUUIDSet(
		"campania_id",
		campaignIDProvided,
		campaniaID,
		true,
	)

	addUUIDSet(
		"estado_lead_id",
		stateIDProvided,
		estadoLeadID,
		false,
	)

	addUUIDSet(
		"asesor_id",
		advisorIDProvided,
		asesorID,
		true,
	)

	if projectProvided &&
		!projectIDProvided {
		param := addArg(
			proyectoInteres,
		)

		setClauses = append(
			setClauses,
			fmt.Sprintf(
				`proyecto_id = COALESCE(
					(
						SELECT id
						FROM proyectos
						WHERE LOWER(TRIM(nombre)) =
							LOWER(TRIM(%s))
						  AND activo = TRUE
						LIMIT 1
					),
					proyecto_id
				)`,
				param,
			),
		)
	}

	if len(setClauses) == 0 {
		leadError(
			c,
			http.StatusBadRequest,
			"lead.no_changes",
			"No se enviaron campos para actualizar.",
			nil,
		)
		return
	}

	setClauses = append(
		setClauses,
		"updated_at = NOW()",
	)

	query := fmt.Sprintf(
		`
			UPDATE leads
			SET
				%s
			WHERE id = $1::uuid
			  AND deleted_at IS NULL
		`,
		strings.Join(
			setClauses,
			",\n				",
		),
	)

	result, err := h.DB.Exec(
		c.Request.Context(),
		query,
		args...,
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

func isValidUUID(
	value string,
) bool {
	return uuidPattern.MatchString(
		strings.TrimSpace(value),
	)
}

func isValidDate(
	value string,
) bool {
	_, err := time.Parse(
		"2006-01-02",
		value,
	)

	return err == nil
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
