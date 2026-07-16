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
	FullName          string `json:"fullName"`
	NombresCompletos  string `json:"nombres_completos"`
	Phone             string `json:"phone"`
	Telefono          string `json:"telefono"`
	Email             string `json:"email"`
	Message           string `json:"message"`
	Mensaje           string `json:"mensaje"`
	Project           string `json:"project"`
	CategoriaInteres  string `json:"categoria_interes"`
	ProyectoID        string `json:"proyecto_id"`
	Campaign          string `json:"campaign"`
	CampaniaSlug      string `json:"campania_slug"`
	CampaniaNombre    string `json:"campania_nombre"`
	OrigenRuta        string `json:"origen_ruta"`
	OrigenComponente  string `json:"origen_componente"`
}

type LeadResponse struct {
	ID                string    `json:"id"`
	NombresCompletos string    `json:"nombres_completos"`
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
	Asesor          string `json:"asesor"`
	CampaniaNombre  string `json:"campania_nombre"`
	CampaniaSlug    string `json:"campania_slug"`
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
	phone = strings.ReplaceAll(phone, " ", "")
	phone = strings.ReplaceAll(phone, "-", "")
	return phone
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

	nombresCompletos := firstValue(body.NombresCompletos, body.FullName)
	telefono := cleanPhone(firstValue(body.Telefono, body.Phone))
	email := strings.ToLower(firstValue(body.Email))
	mensaje := firstValue(body.Mensaje, body.Message)

	categoriaInteres := firstValue(body.CategoriaInteres, body.Project)
	campaniaSlug := firstValue(body.CampaniaSlug, body.Campaign, "formulario-web-general")
	campaniaNombre := firstValue(body.CampaniaNombre, campaniaSlug)

	origenRuta := firstValue(body.OrigenRuta, c.GetHeader("Referer"))
	origenComponente := firstValue(body.OrigenComponente, "Formulario web")

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
			set updated_at = now()
			returning id, nombre, slug
		),
		lead_creado as (
			insert into leads (
				estado_lead_id,
				proyecto_id,
				campania_id,
				nombres_completos,
				telefono,
				email,
				mensaje,
				categoria_interes,
				origen_ruta,
				origen_componente
			)
			values (
				(select id from estado_leads where nombre = 'Nuevo' limit 1),
				nullif($3, '')::uuid,
				(select id from campania_actual limit 1),
				$4,
				$5,
				nullif($6, ''),
				nullif($7, ''),
				nullif($8, ''),
				nullif($9, ''),
				nullif($10, '')
			)
			returning
				id,
				nombres_completos,
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

			coalesce(p.nombre, ''),
			coalesce(p.tipo, ''),
			coalesce(p.ubicacion, ''),
			e.nombre,
			coalesce(a.nombres_completos, ''),
			coalesce(ca.nombre, ''),
			coalesce(ca.slug, '')

		from lead_creado l
		left join proyectos p on p.id = l.proyecto_id
		inner join estado_leads e on e.id = l.estado_lead_id
		left join asesores a on a.id = l.asesor_id
		left join campanias ca on ca.id = l.campania_id;
	`

	err := h.DB.QueryRow(
		c.Request.Context(),
		query,
		campaniaSlug,
		campaniaNombre,
		firstValue(body.ProyectoID),
		nombresCompletos,
		telefono,
		email,
		mensaje,
		categoriaInteres,
		origenRuta,
		origenComponente,
	).Scan(
		&lead.ID,
		&lead.NombresCompletos,
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
		"message":  "Lead registrado correctamente.",
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

			coalesce(p.nombre, ''),
			coalesce(p.tipo, ''),
			coalesce(p.ubicacion, ''),
			e.nombre,
			coalesce(a.nombres_completos, ''),
			coalesce(ca.nombre, ''),
			coalesce(ca.slug, '')

		from leads l
		left join proyectos p on p.id = l.proyecto_id
		inner join estado_leads e on e.id = l.estado_lead_id
		left join asesores a on a.id = l.asesor_id
		left join campanias ca on ca.id = l.campania_id
		where l.deleted_at is null
		order by l.created_at desc;
	`

	rows, err := h.DB.Query(c.Request.Context(), query)
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
				"message":  "Error al leer leads.",
				"data": gin.H{
					"error": err.Error(),
				},
			})
			return
		}

		leads = append(leads, lead)
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