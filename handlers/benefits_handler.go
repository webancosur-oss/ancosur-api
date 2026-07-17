package handlers

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type BenefitsHandler struct {
	DB *pgxpool.Pool
}

func NewBenefitsHandler(
	db *pgxpool.Pool,
) *BenefitsHandler {
	return &BenefitsHandler{
		DB: db,
	}
}

type benefitScanner interface {
	Scan(dest ...any) error
}

/*
|--------------------------------------------------------------------------
| TERRENOS
|--------------------------------------------------------------------------
*/

type CreateTerrainRequest struct {
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

	Campaign          string `json:"campaign"`
	CampaignName      string `json:"campania_nombre"`
	OrigenRuta        string `json:"origen_ruta"`
	OrigenComponente  string `json:"origen_componente"`
}

type UpdateTerrainRequest struct {
	FullName *string `json:"fullName"`
	Phone    *string `json:"phone"`
	Email    *string `json:"email"`
	Message  *string `json:"message"`

	Location       *string `json:"location"`
	District       *string `json:"district"`
	Reference      *string `json:"reference"`
	RegistryNumber *string `json:"registryNumber"`

	Currency *int16    `json:"currency"`
	Price    *float64  `json:"price"`
	AreaM2   *float64  `json:"areaM2"`
	State    *int16    `json:"state"`
	Active   *bool     `json:"active"`
}

type TerrainResponse struct {
	ID     string `json:"id"`
	LeadID string `json:"lead_id"`

	FullName string `json:"nombres_completos"`
	Phone    string `json:"telefono"`
	Email    string `json:"email"`
	Message  string `json:"mensaje"`

	Location       string  `json:"ubicacion"`
	District       string  `json:"distrito"`
	Reference      string  `json:"referencia"`
	RegistryNumber string  `json:"numero_partida"`
	Currency       int16   `json:"moneda"`
	Price          float64 `json:"precio"`
	AreaM2         float64 `json:"area_m2"`
	Observations   string  `json:"observaciones"`

	State  int16 `json:"estado"`
	Active bool  `json:"activo"`

	CampaignName string `json:"campania_nombre"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

const terrainSelectQuery = `
	SELECT
		t.id::text,
		t.lead_id::text,

		l.nombres_completos,
		l.telefono,
		COALESCE(l.email, ''),
		COALESCE(l.mensaje, ''),

		t.ubicacion,
		t.distrito,
		COALESCE(t.referencia, ''),
		COALESCE(t.numero_partida, ''),
		t.moneda,
		COALESCE(t.precio, 0)::double precision,
		COALESCE(t.area_m2, 0)::double precision,
		COALESCE(t.observaciones, ''),

		t.estado,
		t.activo,

		COALESCE(c.nombre, ''),

		t.created_at,
		t.updated_at

	FROM terrenos_ofrecidos t

	INNER JOIN leads l
		ON l.id = t.lead_id

	LEFT JOIN campanias c
		ON c.id = l.campania_id

	WHERE t.deleted_at IS NULL
`

func scanTerrain(
	scanner benefitScanner,
) (TerrainResponse, error) {
	var terrain TerrainResponse

	err := scanner.Scan(
		&terrain.ID,
		&terrain.LeadID,

		&terrain.FullName,
		&terrain.Phone,
		&terrain.Email,
		&terrain.Message,

		&terrain.Location,
		&terrain.District,
		&terrain.Reference,
		&terrain.RegistryNumber,
		&terrain.Currency,
		&terrain.Price,
		&terrain.AreaM2,
		&terrain.Observations,

		&terrain.State,
		&terrain.Active,

		&terrain.CampaignName,

		&terrain.CreatedAt,
		&terrain.UpdatedAt,
	)

	return terrain, err
}

func (h *BenefitsHandler) CreateTerrain(
	c *gin.Context,
) {
	var body CreateTerrainRequest

	if err := c.ShouldBindJSON(&body); err != nil {
		benefitError(
			c,
			http.StatusBadRequest,
			"terrain.invalid_json",
			"Los datos enviados no son válidos.",
			err,
		)
		return
	}

	fullName := strings.TrimSpace(
		body.FullName,
	)

	phone := benefitDigits(
		body.Phone,
	)

	email := strings.ToLower(
		strings.TrimSpace(body.Email),
	)

	location := strings.TrimSpace(
		body.Location,
	)

	district := strings.TrimSpace(
		body.District,
	)

	reference := strings.TrimSpace(
		body.Reference,
	)

	registryNumber := strings.TrimSpace(
		body.RegistryNumber,
	)

	message := strings.TrimSpace(
		body.Message,
	)

	campaignSlug := benefitFirstValue(
		body.Campaign,
		"compramos-tu-terreno-web",
	)

	campaignName := benefitFirstValue(
		body.CampaignName,
		"Compramos tu terreno",
	)

	origenRuta := benefitFirstValue(
		body.OrigenRuta,
		c.GetHeader("Referer"),
		"/beneficios",
	)

	origenComponente := benefitFirstValue(
		body.OrigenComponente,
		"Formulario Compramos tu terreno",
	)

	if fullName == "" {
		benefitError(
			c,
			http.StatusBadRequest,
			"terrain.validation_error",
			"El nombre completo es obligatorio.",
			nil,
		)
		return
	}

	if len(phone) != 9 ||
		!strings.HasPrefix(phone, "9") {
		benefitError(
			c,
			http.StatusBadRequest,
			"terrain.validation_error",
			"El celular debe tener 9 dígitos y empezar con 9.",
			nil,
		)
		return
	}

	if location == "" {
		benefitError(
			c,
			http.StatusBadRequest,
			"terrain.validation_error",
			"La ubicación del terreno es obligatoria.",
			nil,
		)
		return
	}

	if district == "" {
		benefitError(
			c,
			http.StatusBadRequest,
			"terrain.validation_error",
			"El distrito es obligatorio.",
			nil,
		)
		return
	}

	if body.Currency != 1 &&
		body.Currency != 2 {
		benefitError(
			c,
			http.StatusBadRequest,
			"terrain.validation_error",
			"La moneda seleccionada no es válida.",
			nil,
		)
		return
	}

	if body.Price != nil &&
		*body.Price < 0 {
		benefitError(
			c,
			http.StatusBadRequest,
			"terrain.validation_error",
			"El precio no puede ser negativo.",
			nil,
		)
		return
	}

	if body.AreaM2 != nil &&
		*body.AreaM2 <= 0 {
		benefitError(
			c,
			http.StatusBadRequest,
			"terrain.validation_error",
			"El área debe ser mayor que cero.",
			nil,
		)
		return
	}

	if !body.Consent {
		benefitError(
			c,
			http.StatusBadRequest,
			"terrain.validation_error",
			"Debes aceptar ser contactado.",
			nil,
		)
		return
	}

	ctx := c.Request.Context()

	tx, err := h.DB.Begin(ctx)

	if err != nil {
		benefitError(
			c,
			http.StatusInternalServerError,
			"terrain.transaction_error",
			"No se pudo iniciar el registro.",
			err,
		)
		return
	}

	defer tx.Rollback(ctx)

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
		benefitError(
			c,
			http.StatusInternalServerError,
			"terrain.campaign_error",
			"No se pudo registrar la campaña.",
			err,
		)
		return
	}

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
						ORDER BY nombre
						LIMIT 1
					)
				),

				$1::uuid,
				$2,
				'Compramos tu terreno',
				$3,
				NULLIF($4, ''),
				NULLIF($5, ''),
				'Evaluación y compra de terreno',
				NULLIF($6, ''),
				NULLIF($7, ''),
				FALSE,
				TRUE
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
		benefitError(
			c,
			http.StatusInternalServerError,
			"terrain.lead_error",
			"No se pudo registrar el contacto.",
			err,
		)
		return
	}

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
				estado,
				activo
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
				0,
				TRUE
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
		benefitError(
			c,
			http.StatusInternalServerError,
			"terrain.create_error",
			"No se pudo registrar el terreno.",
			err,
		)
		return
	}

	if err := tx.Commit(ctx); err != nil {
		benefitError(
			c,
			http.StatusInternalServerError,
			"terrain.commit_error",
			"No se pudo completar el registro.",
			err,
		)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success":  true,
		"response": "terrain.created",
		"message":  "Terreno registrado correctamente.",
		"data": gin.H{
			"terrain_id": terrainID,
			"lead_id":    leadID,
		},
	})
}

func (h *BenefitsHandler) GetTerrains(
	c *gin.Context,
) {
	rows, err := h.DB.Query(
		c.Request.Context(),
		terrainSelectQuery+
			` ORDER BY t.created_at DESC`,
	)

	if err != nil {
		benefitError(
			c,
			http.StatusInternalServerError,
			"terrain.list_error",
			"No se pudieron consultar los terrenos.",
			err,
		)
		return
	}

	defer rows.Close()

	terrains := []TerrainResponse{}

	for rows.Next() {
		terrain, scanErr :=
			scanTerrain(rows)

		if scanErr != nil {
			benefitError(
				c,
				http.StatusInternalServerError,
				"terrain.scan_error",
				"No se pudo leer un terreno.",
				scanErr,
			)
			return
		}

		terrains = append(
			terrains,
			terrain,
		)
	}

	if err := rows.Err(); err != nil {
		benefitError(
			c,
			http.StatusInternalServerError,
			"terrain.rows_error",
			"No se pudo completar la consulta.",
			err,
		)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":  true,
		"response": "terrain.list",
		"message":  "Terrenos obtenidos correctamente.",
		"data": gin.H{
			"total":    len(terrains),
			"terrains": terrains,
		},
	})
}

func (h *BenefitsHandler) GetTerrainByID(
	c *gin.Context,
) {
	terrainID := strings.TrimSpace(
		c.Param("id"),
	)

	terrain, err := scanTerrain(
		h.DB.QueryRow(
			c.Request.Context(),
			terrainSelectQuery+
				`
					AND t.id = $1::uuid
					LIMIT 1
				`,
			terrainID,
		),
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			benefitError(
				c,
				http.StatusNotFound,
				"terrain.not_found",
				"El terreno no existe.",
				nil,
			)
			return
		}

		benefitError(
			c,
			http.StatusInternalServerError,
			"terrain.read_error",
			"No se pudo consultar el terreno.",
			err,
		)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":  true,
		"response": "terrain.detail",
		"message":  "Terreno obtenido correctamente.",
		"data": gin.H{
			"terrain": terrain,
		},
	})
}

func (h *BenefitsHandler) UpdateTerrain(
	c *gin.Context,
) {
	terrainID := strings.TrimSpace(
		c.Param("id"),
	)

	var body UpdateTerrainRequest

	if err := c.ShouldBindJSON(&body); err != nil {
		benefitError(
			c,
			http.StatusBadRequest,
			"terrain.invalid_json",
			"Los datos enviados no son válidos.",
			err,
		)
		return
	}

	fullName := trimOptionalString(
		body.FullName,
	)

	phone := digitsOptionalString(
		body.Phone,
	)

	email := lowerOptionalString(
		body.Email,
	)

	message := trimOptionalString(
		body.Message,
	)

	location := trimOptionalString(
		body.Location,
	)

	district := trimOptionalString(
		body.District,
	)

	reference := trimOptionalString(
		body.Reference,
	)

	registryNumber := trimOptionalString(
		body.RegistryNumber,
	)

	if phone != nil &&
		(len(*phone) != 9 ||
			!strings.HasPrefix(*phone, "9")) {
		benefitError(
			c,
			http.StatusBadRequest,
			"terrain.validation_error",
			"El celular no es válido.",
			nil,
		)
		return
	}

	if location != nil &&
		*location == "" {
		benefitError(
			c,
			http.StatusBadRequest,
			"terrain.validation_error",
			"La ubicación no puede estar vacía.",
			nil,
		)
		return
	}

	if district != nil &&
		*district == "" {
		benefitError(
			c,
			http.StatusBadRequest,
			"terrain.validation_error",
			"El distrito no puede estar vacío.",
			nil,
		)
		return
	}

	if body.Currency != nil &&
		*body.Currency != 1 &&
		*body.Currency != 2 {
		benefitError(
			c,
			http.StatusBadRequest,
			"terrain.validation_error",
			"La moneda no es válida.",
			nil,
		)
		return
	}

	if body.Price != nil &&
		*body.Price < 0 {
		benefitError(
			c,
			http.StatusBadRequest,
			"terrain.validation_error",
			"El precio no puede ser negativo.",
			nil,
		)
		return
	}

	if body.AreaM2 != nil &&
		*body.AreaM2 <= 0 {
		benefitError(
			c,
			http.StatusBadRequest,
			"terrain.validation_error",
			"El área debe ser mayor que cero.",
			nil,
		)
		return
	}

	if body.State != nil &&
		(*body.State < 0 ||
			*body.State > 6) {
		benefitError(
			c,
			http.StatusBadRequest,
			"terrain.validation_error",
			"El estado no es válido.",
			nil,
		)
		return
	}

	ctx := c.Request.Context()

	tx, err := h.DB.Begin(ctx)

	if err != nil {
		benefitError(
			c,
			http.StatusInternalServerError,
			"terrain.transaction_error",
			"No se pudo iniciar la actualización.",
			err,
		)
		return
	}

	defer tx.Rollback(ctx)

	var leadID string

	err = tx.QueryRow(
		ctx,
		`
			SELECT lead_id::text
			FROM terrenos_ofrecidos
			WHERE id = $1::uuid
			  AND deleted_at IS NULL
			LIMIT 1
		`,
		terrainID,
	).Scan(&leadID)

	if err != nil {
		if err == pgx.ErrNoRows {
			benefitError(
				c,
				http.StatusNotFound,
				"terrain.not_found",
				"El terreno no existe.",
				nil,
			)
			return
		}

		benefitError(
			c,
			http.StatusInternalServerError,
			"terrain.search_error",
			"No se pudo localizar el terreno.",
			err,
		)
		return
	}

	_, err = tx.Exec(
		ctx,
		`
			UPDATE leads
			SET
				nombres_completos =
					COALESCE(
						$2::text,
						nombres_completos
					),

				telefono =
					COALESCE(
						$3::text,
						telefono
					),

				email =
					CASE
						WHEN $4::text IS NULL
							THEN email
						ELSE NULLIF($4, '')
					END,

				mensaje =
					CASE
						WHEN $5::text IS NULL
							THEN mensaje
						ELSE NULLIF($5, '')
					END,

				updated_at = NOW()

			WHERE id = $1::uuid
		`,
		leadID,
		fullName,
		phone,
		email,
		message,
	)

	if err != nil {
		benefitError(
			c,
			http.StatusInternalServerError,
			"terrain.lead_update_error",
			"No se pudo actualizar el contacto.",
			err,
		)
		return
	}

	_, err = tx.Exec(
		ctx,
		`
			UPDATE terrenos_ofrecidos
			SET
				ubicacion =
					COALESCE(
						$2::text,
						ubicacion
					),

				distrito =
					COALESCE(
						$3::text,
						distrito
					),

				referencia =
					CASE
						WHEN $4::text IS NULL
							THEN referencia
						ELSE NULLIF($4, '')
					END,

				numero_partida =
					CASE
						WHEN $5::text IS NULL
							THEN numero_partida
						ELSE NULLIF($5, '')
					END,

				moneda =
					COALESCE(
						$6::smallint,
						moneda
					),

				precio =
					CASE
						WHEN $7::double precision IS NULL
							THEN precio
						ELSE $7::numeric
					END,

				area_m2 =
					CASE
						WHEN $8::double precision IS NULL
							THEN area_m2
						ELSE $8::numeric
					END,

				observaciones =
					CASE
						WHEN $9::text IS NULL
							THEN observaciones
						ELSE NULLIF($9, '')
					END,

				estado =
					COALESCE(
						$10::smallint,
						estado
					),

				activo =
					COALESCE(
						$11::boolean,
						activo
					),

				updated_at = NOW()

			WHERE id = $1::uuid
			  AND deleted_at IS NULL
		`,
		terrainID,
		location,
		district,
		reference,
		registryNumber,
		body.Currency,
		body.Price,
		body.AreaM2,
		message,
		body.State,
		body.Active,
	)

	if err != nil {
		benefitError(
			c,
			http.StatusInternalServerError,
			"terrain.update_error",
			"No se pudo actualizar el terreno.",
			err,
		)
		return
	}

	if err := tx.Commit(ctx); err != nil {
		benefitError(
			c,
			http.StatusInternalServerError,
			"terrain.commit_error",
			"No se pudo completar la actualización.",
			err,
		)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":  true,
		"response": "terrain.updated",
		"message":  "Terreno actualizado correctamente.",
		"data": gin.H{
			"terrain_id": terrainID,
			"lead_id":    leadID,
		},
	})
}

func (h *BenefitsHandler) DeleteTerrain(
	c *gin.Context,
) {
	terrainID := strings.TrimSpace(
		c.Param("id"),
	)

	var deletedID string

	err := h.DB.QueryRow(
		c.Request.Context(),
		`
			UPDATE terrenos_ofrecidos
			SET
				activo = FALSE,
				deleted_at = NOW(),
				updated_at = NOW()
			WHERE id = $1::uuid
			  AND deleted_at IS NULL
			RETURNING id::text
		`,
		terrainID,
	).Scan(&deletedID)

	if err != nil {
		if err == pgx.ErrNoRows {
			benefitError(
				c,
				http.StatusNotFound,
				"terrain.not_found",
				"El terreno no existe.",
				nil,
			)
			return
		}

		benefitError(
			c,
			http.StatusInternalServerError,
			"terrain.delete_error",
			"No se pudo eliminar el terreno.",
			err,
		)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":  true,
		"response": "terrain.deleted",
		"message":  "Terreno eliminado correctamente.",
		"data": gin.H{
			"terrain_id": deletedID,
		},
	})
}

/*
|--------------------------------------------------------------------------
| REFERIDOS
|--------------------------------------------------------------------------
*/

type CreateReferralRequest struct {
	ReferrerFullName string `json:"referrerFullName"`
	ReferrerDNI      string `json:"referrerDni"`
	ReferrerPhone    string `json:"referrerPhone"`
	ReferrerEmail    string `json:"referrerEmail"`

	ReferredFullName string `json:"referredFullName"`
	ReferredDNI      string `json:"referredDni"`
	ReferredPhone    string `json:"referredPhone"`
	ReferredEmail    string `json:"referredEmail"`

	Project string `json:"project"`
	Consent bool   `json:"consent"`

	Campaign          string `json:"campaign"`
	CampaignName      string `json:"campania_nombre"`
	OrigenRuta        string `json:"origen_ruta"`
	OrigenComponente  string `json:"origen_componente"`
}

type UpdateReferralRequest struct {
	ReferredFullName *string `json:"referredFullName"`
	ReferredDNI      *string `json:"referredDni"`
	ReferredPhone    *string `json:"referredPhone"`
	ReferredEmail    *string `json:"referredEmail"`

	Project *string `json:"project"`

	State         *int16   `json:"state"`
	BenefitAmount *float64 `json:"benefitAmount"`
	Observations  *string  `json:"observations"`
	Active        *bool    `json:"active"`
}

type ReferralResponse struct {
	ID string `json:"id"`

	ClientReferrerID string `json:"cliente_referente_id"`
	ReferredLeadID   string `json:"lead_referido_id"`

	State         int16   `json:"estado"`
	BenefitAmount float64 `json:"monto_beneficio"`
	Observations  string  `json:"observaciones"`
	Active        bool    `json:"activo"`

	ReferrerName  string `json:"referente_nombre"`
	ReferrerDNI   string `json:"referente_dni"`
	ReferrerPhone string `json:"referente_telefono"`
	ReferrerEmail string `json:"referente_email"`

	ReferredName  string `json:"referido_nombre"`
	ReferredDNI   string `json:"referido_dni"`
	ReferredPhone string `json:"referido_telefono"`
	ReferredEmail string `json:"referido_email"`

	ProjectName  string `json:"proyecto"`
	CampaignName string `json:"campania"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

const referralSelectQuery = `
	SELECT
		r.id::text,

		r.cliente_referente_id::text,
		r.lead_referido_id::text,

		r.estado,
		COALESCE(
			r.monto_beneficio,
			0
		)::double precision,
		COALESCE(r.observaciones, ''),
		r.activo,

		c.nombres_completos,
		COALESCE(c.numero_documento, ''),
		COALESCE(c.telefono, ''),
		COALESCE(c.email, ''),

		l.nombres_completos,
		COALESCE(l.numero_documento, ''),
		l.telefono,
		COALESCE(l.email, ''),

		COALESCE(
			p.nombre,
			l.proyecto_interes,
			''
		),

		COALESCE(ca.nombre, ''),

		r.created_at,
		r.updated_at

	FROM referidos r

	INNER JOIN clientes c
		ON c.id = r.cliente_referente_id

	INNER JOIN leads l
		ON l.id = r.lead_referido_id

	LEFT JOIN proyectos p
		ON p.id = l.proyecto_id

	LEFT JOIN campanias ca
		ON ca.id = l.campania_id

	WHERE r.deleted_at IS NULL
`

func scanReferral(
	scanner benefitScanner,
) (ReferralResponse, error) {
	var referral ReferralResponse

	err := scanner.Scan(
		&referral.ID,

		&referral.ClientReferrerID,
		&referral.ReferredLeadID,

		&referral.State,
		&referral.BenefitAmount,
		&referral.Observations,
		&referral.Active,

		&referral.ReferrerName,
		&referral.ReferrerDNI,
		&referral.ReferrerPhone,
		&referral.ReferrerEmail,

		&referral.ReferredName,
		&referral.ReferredDNI,
		&referral.ReferredPhone,
		&referral.ReferredEmail,

		&referral.ProjectName,
		&referral.CampaignName,

		&referral.CreatedAt,
		&referral.UpdatedAt,
	)

	return referral, err
}

func (h *BenefitsHandler) CreateReferral(
	c *gin.Context,
) {
	var body CreateReferralRequest

	if err := c.ShouldBindJSON(&body); err != nil {
		benefitError(
			c,
			http.StatusBadRequest,
			"referral.invalid_json",
			"Los datos enviados no son válidos.",
			err,
		)
		return
	}

	referrerDNI := benefitDigits(
		body.ReferrerDNI,
	)

	referrerPhone := benefitDigits(
		body.ReferrerPhone,
	)

	referredFullName := strings.TrimSpace(
		body.ReferredFullName,
	)

	referredDNI := benefitDigits(
		body.ReferredDNI,
	)

	referredPhone := benefitDigits(
		body.ReferredPhone,
	)

	referredEmail := strings.ToLower(
		strings.TrimSpace(
			body.ReferredEmail,
		),
	)

	project := strings.TrimSpace(
		body.Project,
	)

	campaignSlug := benefitFirstValue(
		body.Campaign,
		"socio-referido-web",
	)

	campaignName := benefitFirstValue(
		body.CampaignName,
		"Socio Referido",
	)

	origenRuta := benefitFirstValue(
		body.OrigenRuta,
		"/beneficios",
	)

	origenComponente := benefitFirstValue(
		body.OrigenComponente,
		"Formulario Socio Referido",
	)

	if len(referrerDNI) != 8 {
		benefitError(
			c,
			http.StatusBadRequest,
			"referral.validation_error",
			"El DNI del referente debe tener 8 dígitos.",
			nil,
		)
		return
	}

	if len(referrerPhone) != 9 ||
		!strings.HasPrefix(
			referrerPhone,
			"9",
		) {
		benefitError(
			c,
			http.StatusBadRequest,
			"referral.validation_error",
			"El celular del referente no es válido.",
			nil,
		)
		return
	}

	if referredFullName == "" {
		benefitError(
			c,
			http.StatusBadRequest,
			"referral.validation_error",
			"El nombre del referido es obligatorio.",
			nil,
		)
		return
	}

	if len(referredDNI) != 8 {
		benefitError(
			c,
			http.StatusBadRequest,
			"referral.validation_error",
			"El DNI del referido debe tener 8 dígitos.",
			nil,
		)
		return
	}

	if len(referredPhone) != 9 ||
		!strings.HasPrefix(
			referredPhone,
			"9",
		) {
		benefitError(
			c,
			http.StatusBadRequest,
			"referral.validation_error",
			"El celular del referido no es válido.",
			nil,
		)
		return
	}

	if referrerDNI == referredDNI {
		benefitError(
			c,
			http.StatusBadRequest,
			"referral.validation_error",
			"El referente y el referido no pueden tener el mismo DNI.",
			nil,
		)
		return
	}

	if project == "" {
		benefitError(
			c,
			http.StatusBadRequest,
			"referral.validation_error",
			"Selecciona un proyecto.",
			nil,
		)
		return
	}

	if !body.Consent {
		benefitError(
			c,
			http.StatusBadRequest,
			"referral.validation_error",
			"Debes aceptar los términos y condiciones.",
			nil,
		)
		return
	}

	ctx := c.Request.Context()

	tx, err := h.DB.Begin(ctx)

	if err != nil {
		benefitError(
			c,
			http.StatusInternalServerError,
			"referral.transaction_error",
			"No se pudo iniciar el registro.",
			err,
		)
		return
	}

	defer tx.Rollback(ctx)

	var clientReferrerID string

	err = tx.QueryRow(
		ctx,
		`
			SELECT id::text
			FROM clientes
			WHERE numero_documento = $1
			  AND activo = TRUE
			  AND deleted_at IS NULL
			LIMIT 1
		`,
		referrerDNI,
	).Scan(&clientReferrerID)

	if err != nil {
		if err == pgx.ErrNoRows {
			benefitError(
				c,
				http.StatusNotFound,
				"referral.referrer_not_found",
				"El referente no está registrado como cliente ANCOSUR.",
				nil,
			)
			return
		}

		benefitError(
			c,
			http.StatusInternalServerError,
			"referral.referrer_search_error",
			"No se pudo verificar al referente.",
			err,
		)
		return
	}

	var existingReferralID string

	err = tx.QueryRow(
		ctx,
		`
			SELECT r.id::text
			FROM referidos r

			INNER JOIN leads l
				ON l.id = r.lead_referido_id

			WHERE r.cliente_referente_id =
					$1::uuid
			  AND l.numero_documento = $2
			  AND r.deleted_at IS NULL
			  AND l.deleted_at IS NULL

			LIMIT 1
		`,
		clientReferrerID,
		referredDNI,
	).Scan(&existingReferralID)

	if err == nil {
		benefitError(
			c,
			http.StatusConflict,
			"referral.already_exists",
			"Esta persona ya fue registrada por el cliente.",
			nil,
		)
		return
	}

	if err != pgx.ErrNoRows {
		benefitError(
			c,
			http.StatusInternalServerError,
			"referral.validation_error",
			"No se pudo validar el referido.",
			err,
		)
		return
	}

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
		benefitError(
			c,
			http.StatusInternalServerError,
			"referral.campaign_error",
			"No se pudo registrar la campaña.",
			err,
		)
		return
	}

	var referredLeadID string

	err = tx.QueryRow(
		ctx,
		`
			INSERT INTO leads (
				estado_lead_id,
				campania_id,
				proyecto_id,
				nombres_completos,
				tipo_documento,
				numero_documento,
				telefono,
				email,
				proyecto_interes,
				categoria_interes,
				mensaje,
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
						ORDER BY nombre
						LIMIT 1
					)
				),

				$1::uuid,

				(
					SELECT id
					FROM proyectos
					WHERE LOWER(TRIM(nombre)) =
						LOWER(TRIM($2))
					  AND activo = TRUE
					LIMIT 1
				),

				$3,
				'DNI',
				$4,
				$5,
				NULLIF($6, ''),
				$2,
				'Programa Socio Referido',
				'Persona registrada mediante el programa Socio Referido ANCOSUR.',
				NULLIF($7, ''),
				NULLIF($8, ''),
				FALSE,
				TRUE
			)
			RETURNING id::text
		`,
		campaignID,
		project,
		referredFullName,
		referredDNI,
		referredPhone,
		referredEmail,
		origenRuta,
		origenComponente,
	).Scan(&referredLeadID)

	if err != nil {
		benefitError(
			c,
			http.StatusInternalServerError,
			"referral.lead_error",
			"No se pudo registrar al referido.",
			err,
		)
		return
	}

	var referralID string

	err = tx.QueryRow(
		ctx,
		`
			INSERT INTO referidos (
				cliente_referente_id,
				lead_referido_id,
				estado,
				monto_beneficio,
				observaciones,
				activo
			)
			VALUES (
				$1::uuid,
				$2::uuid,
				0,
				500,
				'Registro enviado desde el formulario web.',
				TRUE
			)
			RETURNING id::text
		`,
		clientReferrerID,
		referredLeadID,
	).Scan(&referralID)

	if err != nil {
		benefitError(
			c,
			http.StatusInternalServerError,
			"referral.create_error",
			"No se pudo registrar la relación del referido.",
			err,
		)
		return
	}

	if err := tx.Commit(ctx); err != nil {
		benefitError(
			c,
			http.StatusInternalServerError,
			"referral.commit_error",
			"No se pudo completar el registro.",
			err,
		)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success":  true,
		"response": "referral.created",
		"message":  "Referido registrado correctamente.",
		"data": gin.H{
			"referral_id":           referralID,
			"cliente_referente_id": clientReferrerID,
			"lead_referido_id":     referredLeadID,
		},
	})
}

func (h *BenefitsHandler) GetReferrals(
	c *gin.Context,
) {
	rows, err := h.DB.Query(
		c.Request.Context(),
		referralSelectQuery+
			` ORDER BY r.created_at DESC`,
	)

	if err != nil {
		benefitError(
			c,
			http.StatusInternalServerError,
			"referral.list_error",
			"No se pudieron consultar los referidos.",
			err,
		)
		return
	}

	defer rows.Close()

	referrals := []ReferralResponse{}

	for rows.Next() {
		referral, scanErr :=
			scanReferral(rows)

		if scanErr != nil {
			benefitError(
				c,
				http.StatusInternalServerError,
				"referral.scan_error",
				"No se pudo leer un referido.",
				scanErr,
			)
			return
		}

		referrals = append(
			referrals,
			referral,
		)
	}

	if err := rows.Err(); err != nil {
		benefitError(
			c,
			http.StatusInternalServerError,
			"referral.rows_error",
			"No se pudo completar la consulta.",
			err,
		)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":  true,
		"response": "referral.list",
		"message":  "Referidos obtenidos correctamente.",
		"data": gin.H{
			"total":     len(referrals),
			"referrals": referrals,
		},
	})
}

func (h *BenefitsHandler) GetReferralByID(
	c *gin.Context,
) {
	referralID := strings.TrimSpace(
		c.Param("id"),
	)

	referral, err := scanReferral(
		h.DB.QueryRow(
			c.Request.Context(),
			referralSelectQuery+
				`
					AND r.id = $1::uuid
					LIMIT 1
				`,
			referralID,
		),
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			benefitError(
				c,
				http.StatusNotFound,
				"referral.not_found",
				"El referido no existe.",
				nil,
			)
			return
		}

		benefitError(
			c,
			http.StatusInternalServerError,
			"referral.read_error",
			"No se pudo consultar el referido.",
			err,
		)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":  true,
		"response": "referral.detail",
		"message":  "Referido obtenido correctamente.",
		"data": gin.H{
			"referral": referral,
		},
	})
}

func (h *BenefitsHandler) UpdateReferral(
	c *gin.Context,
) {
	referralID := strings.TrimSpace(
		c.Param("id"),
	)

	var body UpdateReferralRequest

	if err := c.ShouldBindJSON(&body); err != nil {
		benefitError(
			c,
			http.StatusBadRequest,
			"referral.invalid_json",
			"Los datos enviados no son válidos.",
			err,
		)
		return
	}

	referredFullName := trimOptionalString(
		body.ReferredFullName,
	)

	referredDNI := digitsOptionalString(
		body.ReferredDNI,
	)

	referredPhone := digitsOptionalString(
		body.ReferredPhone,
	)

	referredEmail := lowerOptionalString(
		body.ReferredEmail,
	)

	project := trimOptionalString(
		body.Project,
	)

	observations := trimOptionalString(
		body.Observations,
	)

	if referredDNI != nil &&
		len(*referredDNI) != 8 {
		benefitError(
			c,
			http.StatusBadRequest,
			"referral.validation_error",
			"El DNI debe tener 8 dígitos.",
			nil,
		)
		return
	}

	if referredPhone != nil &&
		(len(*referredPhone) != 9 ||
			!strings.HasPrefix(
				*referredPhone,
				"9",
			)) {
		benefitError(
			c,
			http.StatusBadRequest,
			"referral.validation_error",
			"El celular no es válido.",
			nil,
		)
		return
	}

	if body.State != nil &&
		(*body.State < 0 ||
			*body.State > 5) {
		benefitError(
			c,
			http.StatusBadRequest,
			"referral.validation_error",
			"El estado no es válido.",
			nil,
		)
		return
	}

	if body.BenefitAmount != nil &&
		*body.BenefitAmount < 0 {
		benefitError(
			c,
			http.StatusBadRequest,
			"referral.validation_error",
			"El monto no puede ser negativo.",
			nil,
		)
		return
	}

	ctx := c.Request.Context()

	tx, err := h.DB.Begin(ctx)

	if err != nil {
		benefitError(
			c,
			http.StatusInternalServerError,
			"referral.transaction_error",
			"No se pudo iniciar la actualización.",
			err,
		)
		return
	}

	defer tx.Rollback(ctx)

	var referredLeadID string

	err = tx.QueryRow(
		ctx,
		`
			SELECT lead_referido_id::text
			FROM referidos
			WHERE id = $1::uuid
			  AND deleted_at IS NULL
			LIMIT 1
		`,
		referralID,
	).Scan(&referredLeadID)

	if err != nil {
		if err == pgx.ErrNoRows {
			benefitError(
				c,
				http.StatusNotFound,
				"referral.not_found",
				"El referido no existe.",
				nil,
			)
			return
		}

		benefitError(
			c,
			http.StatusInternalServerError,
			"referral.search_error",
			"No se pudo localizar el referido.",
			err,
		)
		return
	}

	_, err = tx.Exec(
		ctx,
		`
			UPDATE leads
			SET
				nombres_completos =
					COALESCE(
						$2::text,
						nombres_completos
					),

				tipo_documento =
					CASE
						WHEN $3::text IS NULL
							THEN tipo_documento
						ELSE 'DNI'
					END,

				numero_documento =
					COALESCE(
						$3::text,
						numero_documento
					),

				telefono =
					COALESCE(
						$4::text,
						telefono
					),

				email =
					CASE
						WHEN $5::text IS NULL
							THEN email
						ELSE NULLIF($5, '')
					END,

				proyecto_interes =
					COALESCE(
						$6::text,
						proyecto_interes
					),

				proyecto_id =
					CASE
						WHEN $6::text IS NULL
							THEN proyecto_id
						ELSE (
							SELECT id
							FROM proyectos
							WHERE LOWER(TRIM(nombre)) =
								LOWER(TRIM($6))
							  AND activo = TRUE
							LIMIT 1
						)
					END,

				updated_at = NOW()

			WHERE id = $1::uuid
		`,
		referredLeadID,
		referredFullName,
		referredDNI,
		referredPhone,
		referredEmail,
		project,
	)

	if err != nil {
		benefitError(
			c,
			http.StatusInternalServerError,
			"referral.lead_update_error",
			"No se pudo actualizar al referido.",
			err,
		)
		return
	}

	_, err = tx.Exec(
		ctx,
		`
			UPDATE referidos
			SET
				estado =
					COALESCE(
						$2::smallint,
						estado
					),

				monto_beneficio =
					CASE
						WHEN $3::double precision IS NULL
							THEN monto_beneficio
						ELSE $3::numeric
					END,

				observaciones =
					CASE
						WHEN $4::text IS NULL
							THEN observaciones
						ELSE NULLIF($4, '')
					END,

				activo =
					COALESCE(
						$5::boolean,
						activo
					),

				updated_at = NOW()

			WHERE id = $1::uuid
			  AND deleted_at IS NULL
		`,
		referralID,
		body.State,
		body.BenefitAmount,
		observations,
		body.Active,
	)

	if err != nil {
		benefitError(
			c,
			http.StatusInternalServerError,
			"referral.update_error",
			"No se pudo actualizar el referido.",
			err,
		)
		return
	}

	if err := tx.Commit(ctx); err != nil {
		benefitError(
			c,
			http.StatusInternalServerError,
			"referral.commit_error",
			"No se pudo completar la actualización.",
			err,
		)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":  true,
		"response": "referral.updated",
		"message":  "Referido actualizado correctamente.",
		"data": gin.H{
			"referral_id":      referralID,
			"lead_referido_id": referredLeadID,
		},
	})
}

func (h *BenefitsHandler) DeleteReferral(
	c *gin.Context,
) {
	referralID := strings.TrimSpace(
		c.Param("id"),
	)

	var deletedID string

	err := h.DB.QueryRow(
		c.Request.Context(),
		`
			UPDATE referidos
			SET
				activo = FALSE,
				deleted_at = NOW(),
				updated_at = NOW()
			WHERE id = $1::uuid
			  AND deleted_at IS NULL
			RETURNING id::text
		`,
		referralID,
	).Scan(&deletedID)

	if err != nil {
		if err == pgx.ErrNoRows {
			benefitError(
				c,
				http.StatusNotFound,
				"referral.not_found",
				"El referido no existe.",
				nil,
			)
			return
		}

		benefitError(
			c,
			http.StatusInternalServerError,
			"referral.delete_error",
			"No se pudo eliminar el referido.",
			err,
		)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":  true,
		"response": "referral.deleted",
		"message":  "Referido eliminado correctamente.",
		"data": gin.H{
			"referral_id": deletedID,
		},
	})
}

/*
|--------------------------------------------------------------------------
| FUNCIONES AUXILIARES
|--------------------------------------------------------------------------
*/

func benefitDigits(
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

func benefitFirstValue(
	values ...string,
) string {
	for _, value := range values {
		cleanValue :=
			strings.TrimSpace(value)

		if cleanValue != "" {
			return cleanValue
		}
	}

	return ""
}

func trimOptionalString(
	value *string,
) *string {
	if value == nil {
		return nil
	}

	result := strings.TrimSpace(
		*value,
	)

	return &result
}

func lowerOptionalString(
	value *string,
) *string {
	if value == nil {
		return nil
	}

	result := strings.ToLower(
		strings.TrimSpace(*value),
	)

	return &result
}

func digitsOptionalString(
	value *string,
) *string {
	if value == nil {
		return nil
	}

	result := benefitDigits(
		*value,
	)

	return &result
}

func benefitError(
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