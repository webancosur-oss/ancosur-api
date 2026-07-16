package handlers

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

type LeadHandler struct {
	DB *pgxpool.Pool
}

func NewLeadHandler(db *pgxpool.Pool) *LeadHandler {
	return &LeadHandler{
		DB: db,
	}
}

type CreateLeadRequest struct {
	FullName         string `json:"fullName"`
	NombresCompletos string `json:"nombres_completos"`

	Phone    string `json:"phone"`
	Telefono string `json:"telefono"`

	Email string `json:"email"`

	Message string `json:"message"`
	Mensaje string `json:"mensaje"`

	// Nombre visible del proyecto.
	Project         string `json:"project"`
	ProyectoInteres string `json:"proyecto_interes"`

	// Categoría, tipología o motivo de interés.
	Interest         string `json:"interest"`
	CategoriaInteres string `json:"categoria_interes"`

	// UUID opcional del proyecto.
	ProjectID  string `json:"projectId"`
	ProyectoID string `json:"proyecto_id"`

	Campaign       string `json:"campaign"`
	CampaniaSlug   string `json:"campania_slug"`
	CampaniaNombre string `json:"campania_nombre"`

	OrigenRuta       string `json:"origen_ruta"`
	OrigenComponente string `json:"origen_componente"`
}

type LeadResponse struct {
	ID                string    `json:"id"`
	NombresCompletos  string    `json:"nombres_completos"`
	ProyectoInteres   string    `json:"proyecto_interes"`
	Telefono          string    `json:"telefono"`
	Email             string    `json:"email"`
	Mensaje           string    `json:"mensaje"`
	CategoriaInteres  string    `json:"categoria_interes"`
	FuenteProspeccion string    `json:"fuente_prospeccion"`
	OrigenRuta        string    `json:"origen_ruta"`
	OrigenComponente  string    `json:"origen_componente"`
	Atendido          bool      `json:"atendido"`
	Activo            bool      `json:"activo"`
	CreatedAt         time.Time `json:"created_at"`

	Proyecto       string `json:"proyecto"`
	TipoProyecto   string `json:"tipo_proyecto"`
	Ubicacion      string `json:"ubicacion"`
	EstadoLead     string `json:"estado_lead"`
	Asesor         string `json:"asesor"`
	CampaniaNombre string `json:"campania_nombre"`
	CampaniaSlug   string `json:"campania_slug"`
}

type CreateTerrainLeadRequest struct {
	FullName string `json:"fullName"`
	Phone    string `json:"phone"`
	Email    string `json:"email"`

	Location       string `json:"location"`
	District       string `json:"district"`
	Reference      string `json:"reference"`
	RegistryNumber string `json:"registryNumber"`

	Currency int16    `json:"currency"`
	Price    *float64 `json:"price"`
	AreaM2   *float64 `json:"areaM2"`

	Message string `json:"message"`
	Consent bool   `json:"consent"`

	Campaign         string `json:"campaign"`
	CampaniaNombre   string `json:"campania_nombre"`
	OrigenRuta       string `json:"origen_ruta"`
	OrigenComponente string `json:"origen_componente"`
}

func (h *LeadHandler) CreateTerrainLead(c *gin.Context) {
	var body CreateTerrainLeadRequest

	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success":  false,
			"response": "terrain.invalid_json",
			"message":  "El JSON enviado no es válido.",
			"data": gin.H{
				"error": err.Error(),
			},
		})
		return
	}

	fullName := strings.TrimSpace(body.FullName)

	phone := cleanPhone(body.Phone)

	email := strings.ToLower(
		strings.TrimSpace(body.Email),
	)

	location := strings.TrimSpace(body.Location)

	district := strings.TrimSpace(body.District)

	reference := strings.TrimSpace(
		body.Reference,
	)

	registryNumber := strings.TrimSpace(
		body.RegistryNumber,
	)

	message := strings.TrimSpace(
		body.Message,
	)

	campaignSlug := firstValue(
		body.Campaign,
		"compramos-tu-terreno-web",
	)

	campaignName := firstValue(
		body.CampaniaNombre,
		"Compramos tu terreno",
	)

	origenRuta := firstValue(
		body.OrigenRuta,
		c.GetHeader("Referer"),
	)

	origenComponente := firstValue(
		body.OrigenComponente,
		"Formulario Compramos tu terreno",
	)

	if fullName == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success":  false,
			"response": "terrain.validation_error",
			"message":  "El nombre completo es obligatorio.",
			"data":     nil,
		})
		return
	}

	if len(phone) != 9 ||
		!strings.HasPrefix(phone, "9") {

		c.JSON(http.StatusBadRequest, gin.H{
			"success":  false,
			"response": "terrain.validation_error",
			"message":  "El celular debe tener 9 dígitos y empezar con 9.",
			"data":     nil,
		})
		return
	}

	if location == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success":  false,
			"response": "terrain.validation_error",
			"message":  "La ubicación del terreno es obligatoria.",
			"data":     nil,
		})
		return
	}

	if district == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success":  false,
			"response": "terrain.validation_error",
			"message":  "El distrito es obligatorio.",
			"data":     nil,
		})
		return
	}

	if body.Currency < 0 || body.Currency > 2 {
		c.JSON(http.StatusBadRequest, gin.H{
			"success":  false,
			"response": "terrain.validation_error",
			"message":  "La moneda seleccionada no es válida.",
			"data":     nil,
		})
		return
	}

	if body.Price != nil && *body.Price < 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"success":  false,
			"response": "terrain.validation_error",
			"message":  "El precio no puede ser negativo.",
			"data":     nil,
		})
		return
	}

	if body.AreaM2 != nil && *body.AreaM2 <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"success":  false,
			"response": "terrain.validation_error",
			"message":  "El área del terreno debe ser mayor que cero.",
			"data":     nil,
		})
		return
	}

	if !body.Consent {
		c.JSON(http.StatusBadRequest, gin.H{
			"success":  false,
			"response": "terrain.validation_error",
			"message":  "Debes aceptar ser contactado.",
			"data":     nil,
		})
		return
	}

	ctx := c.Request.Context()

	tx, err := h.DB.Begin(ctx)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success":  false,
			"response": "terrain.transaction_error",
			"message":  "No se pudo iniciar el registro.",
			"data": gin.H{
				"error": err.Error(),
			},
		})
		return
	}

	defer tx.Rollback(ctx)

	/*
	 * 1. Crear o actualizar la campaña.
	 */
	var campaignID string

	err = tx.QueryRow(
		ctx,
		`
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
			RETURNING id::text
		`,
		campaignName,
		campaignSlug,
	).Scan(&campaignID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success":  false,
			"response": "terrain.campaign_error",
			"message":  "No se pudo registrar la campaña.",
			"data": gin.H{
				"error": err.Error(),
			},
		})
		return
	}

	/*
	 * 2. Registrar el lead.
	 */
	var leadID string

	err = tx.QueryRow(
		ctx,
		`
			INSERT INTO leads (
				estado_lead_id,
				campania_id,
				nombres_completos,
				proyecto_interes,
				telefono,
				email,
				mensaje,
				categoria_interes,
				origen_ruta,
				origen_componente
			)
			VALUES (
				(
					SELECT id
					FROM estado_leads
					WHERE nombre = 'Nuevo'
					LIMIT 1
				),

				$1::uuid,

				$2,

				'Compramos tu terreno',

				$3,

				NULLIF($4, ''),

				NULLIF($5, ''),

				'Evaluación y compra de terreno',

				NULLIF($6, ''),

				NULLIF($7, '')
			)
			RETURNING id::text
		`,
		campaignID,
		fullName,
		phone,
		email,
		message,
		origenRuta,
		origenComponente,
	).Scan(&leadID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success":  false,
			"response": "terrain.lead_error",
			"message":  "No se pudo registrar el contacto.",
			"data": gin.H{
				"error": err.Error(),
			},
		})
		return
	}

	/*
	 * 3. Registrar los datos específicos del terreno.
	 */
	var terrainID string

	err = tx.QueryRow(
		ctx,
		`
			INSERT INTO terrenos_ofrecidos (
				lead_id,
				ubicacion,
				distrito,
				referencia,
				numero_partida,
				moneda,
				precio,
				area_m2,
				observaciones,
				estado
			)
			VALUES (
				$1::uuid,
				$2,
				$3,
				NULLIF($4, ''),
				NULLIF($5, ''),
				$6,
				$7,
				$8,
				NULLIF($9, ''),
				0
			)
			RETURNING id::text
		`,
		leadID,
		location,
		district,
		reference,
		registryNumber,
		body.Currency,
		body.Price,
		body.AreaM2,
		message,
	).Scan(&terrainID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success":  false,
			"response": "terrain.create_error",
			"message":  "No se pudo registrar el terreno.",
			"data": gin.H{
				"error": err.Error(),
			},
		})
		return
	}

	/*
	 * 4. Confirmar los dos registros.
	 */
	if err := tx.Commit(ctx); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success":  false,
			"response": "terrain.commit_error",
			"message":  "No se pudo completar el registro.",
			"data": gin.H{
				"error": err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success":  true,
		"response": "terrain.created",
		"message":  "Datos enviados correctamente. Nuestro equipo evaluará tu terreno.",
		"data": gin.H{
			"lead_id":    leadID,
			"terrain_id": terrainID,
		},
	})
}

func firstValue(values ...string) string {
	for _, value := range values {
		cleanValue := strings.TrimSpace(value)

		if cleanValue != "" {
			return cleanValue
		}
	}

	return ""
}

func cleanPhone(phone string) string {
	var builder strings.Builder

	for _, character := range phone {
		if character >= '0' && character <= '9' {
			builder.WriteRune(character)
		}
	}

	return builder.String()
}

func (h *LeadHandler) CreateLead(c *gin.Context) {
	var body CreateLeadRequest

	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success":  false,
			"response": "lead.invalid_json",
			"message":  "El JSON enviado no es válido.",
			"data": gin.H{
				"error": err.Error(),
			},
		})
		return
	}

	nombresCompletos := firstValue(
		body.NombresCompletos,
		body.FullName,
	)

	telefono := cleanPhone(
		firstValue(
			body.Telefono,
			body.Phone,
		),
	)

	email := strings.ToLower(
		firstValue(body.Email),
	)

	mensaje := firstValue(
		body.Mensaje,
		body.Message,
	)

	proyectoInteres := firstValue(
		body.ProyectoInteres,
		body.Project,
	)

	categoriaInteres := firstValue(
		body.CategoriaInteres,
		body.Interest,
	)

	proyectoID := firstValue(
		body.ProyectoID,
		body.ProjectID,
	)

	campaniaSlug := firstValue(
		body.CampaniaSlug,
		body.Campaign,
		"formulario-web-general",
	)

	campaniaNombre := firstValue(
		body.CampaniaNombre,
		campaniaSlug,
	)

	origenRuta := firstValue(
		body.OrigenRuta,
		c.GetHeader("Referer"),
	)

	origenComponente := firstValue(
		body.OrigenComponente,
		"Formulario web",
	)

	if nombresCompletos == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success":  false,
			"response": "lead.validation_error",
			"message":  "El nombre completo es obligatorio.",
			"data":     nil,
		})
		return
	}

	if telefono == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success":  false,
			"response": "lead.validation_error",
			"message":  "El celular es obligatorio.",
			"data":     nil,
		})
		return
	}

	if len(telefono) != 9 || !strings.HasPrefix(telefono, "9") {
		c.JSON(http.StatusBadRequest, gin.H{
			"success":  false,
			"response": "lead.validation_error",
			"message":  "El celular debe tener 9 dígitos y empezar con 9.",
			"data":     nil,
		})
		return
	}

	var lead LeadResponse

	query := `
		with campania_actual as (
			insert into campanias (
				nombre,
				slug,
				activo
			)
			values (
				$2,
				$1,
				true
			)
			on conflict (slug) do update
			set
				nombre = excluded.nombre,
				activo = true,
				updated_at = now()
			returning
				id,
				nombre,
				slug
		),

		proyecto_actual as (
			select
				p.id,
				p.nombre
			from proyectos p
			where
				(
					nullif($3, '') is not null
					and p.id::text = nullif($3, '')
				)
				or
				(
					nullif($5, '') is not null
					and lower(trim(p.nombre)) = lower(trim($5))
				)
			order by
				case
					when p.id::text = nullif($3, '') then 0
					else 1
				end
			limit 1
		),

		lead_creado as (
			insert into leads (
				estado_lead_id,
				proyecto_id,
				campania_id,
				nombres_completos,
				proyecto_interes,
				telefono,
				email,
				mensaje,
				categoria_interes,
				origen_ruta,
				origen_componente
			)
			values (
				(
					select id
					from estado_leads
					where nombre = 'Nuevo'
					limit 1
				),

				(
					select id
					from proyecto_actual
					limit 1
				),

				(
					select id
					from campania_actual
					limit 1
				),

				$4,

				coalesce(
					nullif($5, ''),
					(
						select nombre
						from proyecto_actual
						limit 1
					)
				),

				$6,
				nullif($7, ''),
				nullif($8, ''),
				nullif($9, ''),
				nullif($10, ''),
				nullif($11, '')
			)
			returning
				id,
				nombres_completos,
				proyecto_interes,
				telefono,
				email,
				mensaje,
				categoria_interes,
				fuente_prospeccion,
				origen_ruta,
				origen_componente,
				atendido,
				activo,
				created_at,
				proyecto_id,
				estado_lead_id,
				asesor_id,
				campania_id
		)

		select
			l.id::text,
			l.nombres_completos,
			coalesce(l.proyecto_interes, ''),
			l.telefono,
			coalesce(l.email, ''),
			coalesce(l.mensaje, ''),
			coalesce(l.categoria_interes, ''),
			l.fuente_prospeccion,
			coalesce(l.origen_ruta, ''),
			coalesce(l.origen_componente, ''),
			l.atendido,
			l.activo,
			l.created_at,

			coalesce(
				p.nombre,
				l.proyecto_interes,
				''
			),

			coalesce(p.tipo, ''),
			coalesce(p.ubicacion, ''),
			e.nombre,
			coalesce(a.nombres_completos, ''),
			coalesce(ca.nombre, ''),
			coalesce(ca.slug, '')

		from lead_creado l

		left join proyectos p
			on p.id = l.proyecto_id

		inner join estado_leads e
			on e.id = l.estado_lead_id

		left join asesores a
			on a.id = l.asesor_id

		left join campanias ca
			on ca.id = l.campania_id;
	`

	err := h.DB.QueryRow(
		c.Request.Context(),
		query,

		// $1
		campaniaSlug,

		// $2
		campaniaNombre,

		// $3
		proyectoID,

		// $4
		nombresCompletos,

		// $5
		proyectoInteres,

		// $6
		telefono,

		// $7
		email,

		// $8
		mensaje,

		// $9
		categoriaInteres,

		// $10
		origenRuta,

		// $11
		origenComponente,
	).Scan(
		&lead.ID,
		&lead.NombresCompletos,
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
		&lead.Proyecto,
		&lead.TipoProyecto,
		&lead.Ubicacion,
		&lead.EstadoLead,
		&lead.Asesor,
		&lead.CampaniaNombre,
		&lead.CampaniaSlug,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success":  false,
			"response": "lead.create_error",
			"message":  "No se pudo registrar el lead.",
			"data": gin.H{
				"error": err.Error(),
			},
		})
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

func (h *LeadHandler) GetAllLeads(c *gin.Context) {
	query := `
		select
			l.id::text,
			l.nombres_completos,
			coalesce(l.proyecto_interes, ''),
			l.telefono,
			coalesce(l.email, ''),
			coalesce(l.mensaje, ''),
			coalesce(l.categoria_interes, ''),
			l.fuente_prospeccion,
			coalesce(l.origen_ruta, ''),
			coalesce(l.origen_componente, ''),
			l.atendido,
			l.activo,
			l.created_at,

			coalesce(
				p.nombre,
				l.proyecto_interes,
				''
			),

			coalesce(p.tipo, ''),
			coalesce(p.ubicacion, ''),
			e.nombre,
			coalesce(a.nombres_completos, ''),
			coalesce(ca.nombre, ''),
			coalesce(ca.slug, '')

		from leads l

		left join proyectos p
			on p.id = l.proyecto_id

		inner join estado_leads e
			on e.id = l.estado_lead_id

		left join asesores a
			on a.id = l.asesor_id

		left join campanias ca
			on ca.id = l.campania_id

		where l.deleted_at is null

		order by l.created_at desc;
	`

	rows, err := h.DB.Query(
		c.Request.Context(),
		query,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success":  false,
			"response": "lead.list_error",
			"message":  "Error al consultar leads.",
			"data": gin.H{
				"error": err.Error(),
			},
		})
		return
	}

	defer rows.Close()

	leads := []LeadResponse{}

	for rows.Next() {
		var lead LeadResponse

		err := rows.Scan(
			&lead.ID,
			&lead.NombresCompletos,
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
			&lead.Proyecto,
			&lead.TipoProyecto,
			&lead.Ubicacion,
			&lead.EstadoLead,
			&lead.Asesor,
			&lead.CampaniaNombre,
			&lead.CampaniaSlug,
		)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success":  false,
				"response": "lead.scan_error",
				"message":  "Error al leer los leads.",
				"data": gin.H{
					"error": err.Error(),
				},
			})
			return
		}

		leads = append(leads, lead)
	}

	if err := rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success":  false,
			"response": "lead.rows_error",
			"message":  "Error al recorrer los leads.",
			"data": gin.H{
				"error": err.Error(),
			},
		})
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
