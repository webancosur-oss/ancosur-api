package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/mail"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

/*
|--------------------------------------------------------------------------
| HANDLER
|--------------------------------------------------------------------------
*/

type InvestmentHandler struct {
	DB *pgxpool.Pool
}

func NewInvestmentHandler(db *pgxpool.Pool) *InvestmentHandler {
	return &InvestmentHandler{DB: db}
}

/*
|--------------------------------------------------------------------------
| REQUESTS
|--------------------------------------------------------------------------
*/

type CreateInvestmentRequest struct {
	LeadID         string `json:"lead_id"`
	RangoInversion string `json:"rango_inversion"`
	Mensaje        string `json:"mensaje"`
	Estado         *int16 `json:"estado"`
	Atendido       *bool  `json:"atendido"`
	Activo         *bool  `json:"activo"`
}

type UpdateInvestmentRequest struct {
	RangoInversion *string `json:"rango_inversion"`
	Mensaje        *string `json:"mensaje"`
	Estado         *int16  `json:"estado"`
	Atendido       *bool   `json:"atendido"`
	Activo         *bool   `json:"activo"`
}

type CreateInvestmentLeadRequest struct {
	FullName         string `json:"fullName"`
	NombresCompletos string `json:"nombres_completos"`

	Phone    string `json:"phone"`
	Telefono string `json:"telefono"`

	Email string `json:"email"`

	InvestmentAmount string `json:"investmentAmount"`
	RangoInversion   string `json:"rango_inversion"`

	Message string `json:"message"`
	Mensaje string `json:"mensaje"`

	Project         string `json:"project"`
	ProyectoInteres string `json:"proyecto_interes"`

	Interest         string `json:"interest"`
	CategoriaInteres string `json:"categoria_interes"`

	Source            string `json:"source"`
	FuenteProspeccion string `json:"fuente_prospeccion"`

	Campaign       string `json:"campaign"`
	CampaniaSlug   string `json:"campania_slug"`
	CampaniaNombre string `json:"campania_nombre"`

	OrigenRuta       string `json:"origen_ruta"`
	OrigenComponente string `json:"origen_componente"`
}

/*
|--------------------------------------------------------------------------
| RESPONSE
|--------------------------------------------------------------------------
*/

type InvestmentResponse struct {
	ID             string     `json:"id"`
	LeadID         string     `json:"lead_id"`
	RangoInversion string     `json:"rango_inversion"`
	Mensaje        string     `json:"mensaje"`
	Estado         int16      `json:"estado"`
	Atendido       bool       `json:"atendido"`
	Activo         bool       `json:"activo"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	DeletedAt      *time.Time `json:"deleted_at"`

	NombresCompletos  string `json:"nombres_completos"`
	Telefono          string `json:"telefono"`
	Email             string `json:"email"`
	ProyectoInteres   string `json:"proyecto_interes"`
	CategoriaInteres  string `json:"categoria_interes"`
	FuenteProspeccion string `json:"fuente_prospeccion"`
	OrigenRuta        string `json:"origen_ruta"`
	OrigenComponente  string `json:"origen_componente"`
	CampaniaNombre    string `json:"campania_nombre"`
	CampaniaSlug      string `json:"campania_slug"`
}

type investmentScanner interface {
	Scan(dest ...any) error
}

/*
|--------------------------------------------------------------------------
| BASE QUERY
|--------------------------------------------------------------------------
*/

const investmentSelectQuery = `
	SELECT
		s.id::text,
		s.lead_id::text,
		COALESCE(s.rango_inversion, ''),
		COALESCE(s.mensaje, ''),
		COALESCE(s.estado, 1),
		COALESCE(s.atendido, FALSE),
		COALESCE(s.activo, TRUE),
		COALESCE(s.created_at, NOW()),
		COALESCE(s.updated_at, s.created_at, NOW()),
		s.deleted_at,

		COALESCE(l.nombres_completos, ''),
		COALESCE(l.telefono, ''),
		COALESCE(l.email, ''),
		COALESCE(l.proyecto_interes, ''),
		COALESCE(l.categoria_interes, ''),
		COALESCE(l.fuente_prospeccion, ''),
		COALESCE(l.origen_ruta, ''),
		COALESCE(l.origen_componente, ''),
		COALESCE(c.nombre, ''),
		COALESCE(c.slug, '')

	FROM solicitudes_inversion s

	INNER JOIN leads l
		ON l.id = s.lead_id

	LEFT JOIN campanias c
		ON c.id = l.campania_id
`

func scanInvestment(scanner investmentScanner) (InvestmentResponse, error) {
	var investment InvestmentResponse

	err := scanner.Scan(
		&investment.ID,
		&investment.LeadID,
		&investment.RangoInversion,
		&investment.Mensaje,
		&investment.Estado,
		&investment.Atendido,
		&investment.Activo,
		&investment.CreatedAt,
		&investment.UpdatedAt,
		&investment.DeletedAt,

		&investment.NombresCompletos,
		&investment.Telefono,
		&investment.Email,
		&investment.ProyectoInteres,
		&investment.CategoriaInteres,
		&investment.FuenteProspeccion,
		&investment.OrigenRuta,
		&investment.OrigenComponente,
		&investment.CampaniaNombre,
		&investment.CampaniaSlug,
	)

	return investment, err
}

/*
|--------------------------------------------------------------------------
| HELPERS
|--------------------------------------------------------------------------
*/

var investmentUUIDPattern = regexp.MustCompile(
	`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[1-5][0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12}$`,
)

func investmentSuccess(
	c *gin.Context,
	status int,
	response string,
	message string,
	data any,
) {
	c.JSON(status, gin.H{
		"success":  true,
		"response": response,
		"message":  message,
		"data":     data,
	})
}

func investmentError(
	c *gin.Context,
	status int,
	response string,
	message string,
	err error,
) {
	var data any

	if err != nil {
		data = gin.H{"error": err.Error()}
	}

	c.JSON(status, gin.H{
		"success":  false,
		"response": response,
		"message":  message,
		"data":     data,
	})
}

func investmentFirstValue(values ...string) string {
	for _, value := range values {
		cleanValue := strings.TrimSpace(value)

		if cleanValue != "" {
			return cleanValue
		}
	}

	return ""
}

func investmentDigits(value string) string {
	var builder strings.Builder

	for _, character := range value {
		if character >= '0' && character <= '9' {
			builder.WriteRune(character)
		}
	}

	return builder.String()
}

func investmentValidEmail(value string) bool {
	cleanValue := strings.TrimSpace(value)

	if cleanValue == "" {
		return true
	}

	address, err := mail.ParseAddress(cleanValue)
	if err != nil {
		return false
	}

	return strings.EqualFold(address.Address, cleanValue)
}

func investmentValidUUID(value string) bool {
	return investmentUUIDPattern.MatchString(strings.TrimSpace(value))
}

func investmentHasSQLState(err error, code string) bool {
	var pgError *pgconn.PgError

	return errors.As(err, &pgError) && pgError.Code == code
}

func investmentParseLimit(value string, defaultValue int, maxValue int) int {
	parsedValue, err := strconv.Atoi(value)
	if err != nil || parsedValue < 1 {
		return defaultValue
	}

	if parsedValue > maxValue {
		return maxValue
	}

	return parsedValue
}

func investmentParseOffset(value string) int {
	parsedValue, err := strconv.Atoi(value)
	if err != nil || parsedValue < 0 {
		return 0
	}

	return parsedValue
}

func investmentGetDefaultLeadStateID(
	ctx context.Context,
	tx pgx.Tx,
) (string, error) {
	var stateID string

	err := tx.QueryRow(
		ctx,
		`
			SELECT id::text
			FROM estado_leads
			ORDER BY
				CASE
					WHEN LOWER(TRIM(nombre)) =
						LOWER('Contacto inicial del cliente')
					THEN 0

					WHEN LOWER(TRIM(nombre)) =
						LOWER('Nuevo')
					THEN 1

					ELSE 2
				END,
				nombre ASC
			LIMIT 1
		`,
	).Scan(&stateID)

	return stateID, err
}

func investmentGetOrCreateCampaignID(
	ctx context.Context,
	tx pgx.Tx,
	name string,
	slug string,
) (string, error) {
	var campaignID string

	err := tx.QueryRow(
		ctx,
		`
			SELECT id::text
			FROM campanias
			WHERE slug = $1
			LIMIT 1
		`,
		slug,
	).Scan(&campaignID)

	if err == nil {
		_, updateErr := tx.Exec(
			ctx,
			`
				UPDATE campanias
				SET
					nombre = $2,
					activo = TRUE,
					updated_at = NOW()
				WHERE id = $1::uuid
			`,
			campaignID,
			name,
		)

		if updateErr != nil {
			return "", updateErr
		}

		return campaignID, nil
	}

	if !errors.Is(err, pgx.ErrNoRows) {
		return "", err
	}

	err = tx.QueryRow(
		ctx,
		`
			INSERT INTO campanias (
				nombre,
				slug,
				activo,
				created_at,
				updated_at
			)
			VALUES (
				$1,
				$2,
				TRUE,
				NOW(),
				NOW()
			)
			RETURNING id::text
		`,
		name,
		slug,
	).Scan(&campaignID)

	if err != nil &&
		investmentHasSQLState(err, "23505") {
		err = tx.QueryRow(
			ctx,
			`
				SELECT id::text
				FROM campanias
				WHERE slug = $1
				LIMIT 1
			`,
			slug,
		).Scan(&campaignID)
	}

	return campaignID, err
}

func (h *InvestmentHandler) getInvestmentByID(
	ctx context.Context,
	investmentID string,
) (InvestmentResponse, error) {
	return scanInvestment(
		h.DB.QueryRow(
			ctx,
			investmentSelectQuery+`
				WHERE s.id = $1::uuid
				  AND s.deleted_at IS NULL
				  AND l.deleted_at IS NULL
				LIMIT 1
			`,
			investmentID,
		),
	)
}

/*
|--------------------------------------------------------------------------
| CREATE DIRECT INVESTMENT
| POST /api/investments
|--------------------------------------------------------------------------
*/

func (h *InvestmentHandler) CreateInvestment(c *gin.Context) {
	var body CreateInvestmentRequest

	if err := c.ShouldBindJSON(&body); err != nil {
		investmentError(
			c,
			http.StatusBadRequest,
			"investment.invalid_json",
			"El JSON enviado no es válido.",
			err,
		)
		return
	}

	leadID := strings.TrimSpace(body.LeadID)
	rangoInversion := strings.TrimSpace(body.RangoInversion)
	mensaje := strings.TrimSpace(body.Mensaje)

	if !investmentValidUUID(leadID) {
		investmentError(
			c,
			http.StatusBadRequest,
			"investment.invalid_lead_id",
			"El lead_id no es válido.",
			nil,
		)
		return
	}

	if rangoInversion == "" {
		investmentError(
			c,
			http.StatusBadRequest,
			"investment.range_required",
			"El rango de inversión es obligatorio.",
			nil,
		)
		return
	}

	estado := int16(1)
	if body.Estado != nil {
		estado = *body.Estado
	}

	if estado < 0 {
		investmentError(
			c,
			http.StatusBadRequest,
			"investment.invalid_state",
			"El estado no puede ser negativo.",
			nil,
		)
		return
	}

	atendido := false
	if body.Atendido != nil {
		atendido = *body.Atendido
	}

	activo := true
	if body.Activo != nil {
		activo = *body.Activo
	}

	ctx := c.Request.Context()
	var investmentID string

	err := h.DB.QueryRow(
		ctx,
		`
			INSERT INTO solicitudes_inversion (
				lead_id,
				rango_inversion,
				mensaje,
				estado,
				atendido,
				activo,
				created_at,
				updated_at
			)
			VALUES (
				$1::uuid,
				$2,
				NULLIF($3, ''),
				$4,
				$5,
				$6,
				NOW(),
				NOW()
			)
			RETURNING id::text
		`,
		leadID,
		rangoInversion,
		mensaje,
		estado,
		atendido,
		activo,
	).Scan(&investmentID)

	if err != nil {
		switch {
		case investmentHasSQLState(err, "23505"):
			investmentError(
				c,
				http.StatusConflict,
				"investment.already_exists",
				"Este lead ya tiene una solicitud de inversión registrada.",
				nil,
			)

		case investmentHasSQLState(err, "23503"):
			investmentError(
				c,
				http.StatusBadRequest,
				"investment.lead_not_found",
				"El lead indicado no existe.",
				nil,
			)

		default:
			investmentError(
				c,
				http.StatusInternalServerError,
				"investment.create_error",
				"No se pudo registrar la solicitud de inversión.",
				err,
			)
		}

		return
	}

	investment, err := h.getInvestmentByID(ctx, investmentID)
	if err != nil {
		investmentError(
			c,
			http.StatusInternalServerError,
			"investment.read_after_create_error",
			"La solicitud fue registrada, pero no se pudo consultar.",
			err,
		)
		return
	}

	investmentSuccess(
		c,
		http.StatusCreated,
		"investment.created",
		"Solicitud de inversión registrada correctamente.",
		gin.H{"investment": investment},
	)
}

/*
|--------------------------------------------------------------------------
| READ ALL
| GET /api/investments
|--------------------------------------------------------------------------
*/

func (h *InvestmentHandler) GetInvestments(c *gin.Context) {
	ctx := c.Request.Context()

	limit := investmentParseLimit(
		c.DefaultQuery("limit", "50"),
		50,
		200,
	)

	offset := investmentParseOffset(
		c.DefaultQuery("offset", "0"),
	)

	conditions := []string{
		"s.deleted_at IS NULL",
		"l.deleted_at IS NULL",
	}

	args := make([]any, 0)

	addArgument := func(value any) string {
		args = append(args, value)
		return fmt.Sprintf("$%d", len(args))
	}

	if estadoQuery := strings.TrimSpace(c.Query("estado")); estadoQuery != "" {
		estadoValue, err := strconv.ParseInt(estadoQuery, 10, 16)
		if err != nil || estadoValue < 0 {
			investmentError(
				c,
				http.StatusBadRequest,
				"investment.invalid_state",
				"El filtro estado no es válido.",
				nil,
			)
			return
		}

		placeholder := addArgument(int16(estadoValue))
		conditions = append(conditions, "s.estado = "+placeholder)
	}

	if attendedQuery := strings.TrimSpace(c.Query("atendido")); attendedQuery != "" {
		attendedValue, err := strconv.ParseBool(attendedQuery)
		if err != nil {
			investmentError(
				c,
				http.StatusBadRequest,
				"investment.invalid_attended",
				"El filtro atendido no es válido.",
				nil,
			)
			return
		}

		placeholder := addArgument(attendedValue)
		conditions = append(conditions, "s.atendido = "+placeholder)
	}

	if activeQuery := strings.TrimSpace(c.Query("activo")); activeQuery != "" {
		activeValue, err := strconv.ParseBool(activeQuery)
		if err != nil {
			investmentError(
				c,
				http.StatusBadRequest,
				"investment.invalid_active",
				"El filtro activo no es válido.",
				nil,
			)
			return
		}

		placeholder := addArgument(activeValue)
		conditions = append(conditions, "s.activo = "+placeholder)
	}

	if leadID := strings.TrimSpace(c.Query("lead_id")); leadID != "" {
		if !investmentValidUUID(leadID) {
			investmentError(
				c,
				http.StatusBadRequest,
				"investment.invalid_lead_id",
				"El filtro lead_id no es válido.",
				nil,
			)
			return
		}

		placeholder := addArgument(leadID)
		conditions = append(conditions, "s.lead_id = "+placeholder+"::uuid")
	}

	if search := strings.TrimSpace(c.Query("search")); search != "" {
		placeholder := addArgument("%" + search + "%")

		conditions = append(
			conditions,
			`(
				s.rango_inversion ILIKE `+placeholder+`
				OR COALESCE(s.mensaje, '') ILIKE `+placeholder+`
				OR COALESCE(l.nombres_completos, '') ILIKE `+placeholder+`
				OR COALESCE(l.telefono, '') ILIKE `+placeholder+`
				OR COALESCE(l.email, '') ILIKE `+placeholder+`
				OR COALESCE(l.proyecto_interes, '') ILIKE `+placeholder+`
			)`,
		)
	}

	whereClause := " WHERE " + strings.Join(conditions, " AND ")

	countQuery := `
		SELECT COUNT(*)
		FROM solicitudes_inversion s
		INNER JOIN leads l
			ON l.id = s.lead_id
	` + whereClause

	var total int

	if err := h.DB.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		investmentError(
			c,
			http.StatusInternalServerError,
			"investment.count_error",
			"No se pudo contar las solicitudes de inversión.",
			err,
		)
		return
	}

	listArgs := append([]any{}, args...)

	limitPlaceholder := fmt.Sprintf("$%d", len(listArgs)+1)
	listArgs = append(listArgs, limit)

	offsetPlaceholder := fmt.Sprintf("$%d", len(listArgs)+1)
	listArgs = append(listArgs, offset)

	query := investmentSelectQuery +
		whereClause +
		` ORDER BY s.created_at DESC LIMIT ` +
		limitPlaceholder +
		` OFFSET ` +
		offsetPlaceholder

	rows, err := h.DB.Query(ctx, query, listArgs...)
	if err != nil {
		investmentError(
			c,
			http.StatusInternalServerError,
			"investment.list_error",
			"No se pudieron consultar las solicitudes de inversión.",
			err,
		)
		return
	}
	defer rows.Close()

	investments := make([]InvestmentResponse, 0)

	for rows.Next() {
		investment, scanErr := scanInvestment(rows)
		if scanErr != nil {
			investmentError(
				c,
				http.StatusInternalServerError,
				"investment.scan_error",
				"No se pudo leer una solicitud de inversión.",
				scanErr,
			)
			return
		}

		investments = append(investments, investment)
	}

	if err := rows.Err(); err != nil {
		investmentError(
			c,
			http.StatusInternalServerError,
			"investment.rows_error",
			"No se pudo completar la consulta de solicitudes.",
			err,
		)
		return
	}

	investmentSuccess(
		c,
		http.StatusOK,
		"investment.list",
		"Solicitudes de inversión obtenidas correctamente.",
		gin.H{
			"items":  investments,
			"total":  total,
			"limit":  limit,
			"offset": offset,
		},
	)
}

/*
|--------------------------------------------------------------------------
| READ BY ID
| GET /api/investments/:id
|--------------------------------------------------------------------------
*/

func (h *InvestmentHandler) GetInvestmentByID(c *gin.Context) {
	investmentID := strings.TrimSpace(c.Param("id"))

	if !investmentValidUUID(investmentID) {
		investmentError(
			c,
			http.StatusBadRequest,
			"investment.invalid_id",
			"El ID de la solicitud no es válido.",
			nil,
		)
		return
	}

	investment, err := h.getInvestmentByID(
		c.Request.Context(),
		investmentID,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			investmentError(
				c,
				http.StatusNotFound,
				"investment.not_found",
				"La solicitud de inversión no existe.",
				nil,
			)
			return
		}

		investmentError(
			c,
			http.StatusInternalServerError,
			"investment.read_error",
			"No se pudo consultar la solicitud de inversión.",
			err,
		)
		return
	}

	investmentSuccess(
		c,
		http.StatusOK,
		"investment.detail",
		"Solicitud de inversión obtenida correctamente.",
		gin.H{"investment": investment},
	)
}

/*
|--------------------------------------------------------------------------
| UPDATE
| PUT/PATCH /api/investments/:id
|--------------------------------------------------------------------------
*/

func (h *InvestmentHandler) UpdateInvestment(c *gin.Context) {
	investmentID := strings.TrimSpace(c.Param("id"))

	if !investmentValidUUID(investmentID) {
		investmentError(
			c,
			http.StatusBadRequest,
			"investment.invalid_id",
			"El ID de la solicitud no es válido.",
			nil,
		)
		return
	}

	var body UpdateInvestmentRequest

	if err := c.ShouldBindJSON(&body); err != nil {
		investmentError(
			c,
			http.StatusBadRequest,
			"investment.invalid_json",
			"Los datos enviados no son válidos.",
			err,
		)
		return
	}

	if body.RangoInversion == nil &&
		body.Mensaje == nil &&
		body.Estado == nil &&
		body.Atendido == nil &&
		body.Activo == nil {
		investmentError(
			c,
			http.StatusBadRequest,
			"investment.empty_update",
			"No se enviaron campos para actualizar.",
			nil,
		)
		return
	}

	updates := make([]string, 0)
	args := make([]any, 0)

	addUpdate := func(column string, value any) {
		args = append(args, value)
		updates = append(
			updates,
			fmt.Sprintf("%s = $%d", column, len(args)),
		)
	}

	if body.RangoInversion != nil {
		value := strings.TrimSpace(*body.RangoInversion)
		if value == "" {
			investmentError(
				c,
				http.StatusBadRequest,
				"investment.range_required",
				"El rango de inversión no puede estar vacío.",
				nil,
			)
			return
		}

		addUpdate("rango_inversion", value)
	}

	if body.Mensaje != nil {
		args = append(args, strings.TrimSpace(*body.Mensaje))
		updates = append(
			updates,
			fmt.Sprintf("mensaje = NULLIF($%d, '')", len(args)),
		)
	}

	if body.Estado != nil {
		if *body.Estado < 0 {
			investmentError(
				c,
				http.StatusBadRequest,
				"investment.invalid_state",
				"El estado no puede ser negativo.",
				nil,
			)
			return
		}

		addUpdate("estado", *body.Estado)
	}

	if body.Atendido != nil {
		addUpdate("atendido", *body.Atendido)
	}

	if body.Activo != nil {
		addUpdate("activo", *body.Activo)
	}

	updates = append(updates, "updated_at = NOW()")

	args = append(args, investmentID)
	idPlaceholder := fmt.Sprintf("$%d", len(args))

	query := `
		UPDATE solicitudes_inversion
		SET ` + strings.Join(updates, ", ") + `
		WHERE id = ` + idPlaceholder + `::uuid
		  AND deleted_at IS NULL
		RETURNING id::text
	`

	var updatedID string

	err := h.DB.QueryRow(
		c.Request.Context(),
		query,
		args...,
	).Scan(&updatedID)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			investmentError(
				c,
				http.StatusNotFound,
				"investment.not_found",
				"La solicitud de inversión no existe.",
				nil,
			)
			return
		}

		investmentError(
			c,
			http.StatusInternalServerError,
			"investment.update_error",
			"No se pudo actualizar la solicitud de inversión.",
			err,
		)
		return
	}

	investment, err := h.getInvestmentByID(
		c.Request.Context(),
		updatedID,
	)
	if err != nil {
		investmentError(
			c,
			http.StatusInternalServerError,
			"investment.read_after_update_error",
			"La solicitud fue actualizada, pero no se pudo consultar.",
			err,
		)
		return
	}

	investmentSuccess(
		c,
		http.StatusOK,
		"investment.updated",
		"Solicitud de inversión actualizada correctamente.",
		gin.H{"investment": investment},
	)
}

/*
|--------------------------------------------------------------------------
| DELETE - SOFT DELETE
| DELETE /api/investments/:id
|--------------------------------------------------------------------------
*/

func (h *InvestmentHandler) DeleteInvestment(c *gin.Context) {
	investmentID := strings.TrimSpace(c.Param("id"))

	if !investmentValidUUID(investmentID) {
		investmentError(
			c,
			http.StatusBadRequest,
			"investment.invalid_id",
			"El ID de la solicitud no es válido.",
			nil,
		)
		return
	}

	var deletedID string

	err := h.DB.QueryRow(
		c.Request.Context(),
		`
			UPDATE solicitudes_inversion
			SET
				activo = FALSE,
				deleted_at = NOW(),
				updated_at = NOW()
			WHERE id = $1::uuid
			  AND deleted_at IS NULL
			RETURNING id::text
		`,
		investmentID,
	).Scan(&deletedID)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			investmentError(
				c,
				http.StatusNotFound,
				"investment.not_found",
				"La solicitud de inversión no existe.",
				nil,
			)
			return
		}

		investmentError(
			c,
			http.StatusInternalServerError,
			"investment.delete_error",
			"No se pudo eliminar la solicitud de inversión.",
			err,
		)
		return
	}

	investmentSuccess(
		c,
		http.StatusOK,
		"investment.deleted",
		"Solicitud de inversión eliminada correctamente.",
		gin.H{"id": deletedID},
	)
}

/*
|--------------------------------------------------------------------------
| CREATE LEAD + INVESTMENT IN ONE TRANSACTION
| POST /api/investment-leads
|--------------------------------------------------------------------------
*/

func (h *InvestmentHandler) CreateInvestmentLead(c *gin.Context) {
	var body CreateInvestmentLeadRequest

	if err := c.ShouldBindJSON(&body); err != nil {
		investmentError(
			c,
			http.StatusBadRequest,
			"investment_lead.invalid_json",
			"El JSON enviado no es válido.",
			err,
		)
		return
	}

	fullName := investmentFirstValue(
		body.NombresCompletos,
		body.FullName,
	)

	phone := investmentDigits(
		investmentFirstValue(
			body.Telefono,
			body.Phone,
		),
	)

	email := strings.ToLower(strings.TrimSpace(body.Email))

	rangoInversion := investmentFirstValue(
		body.RangoInversion,
		body.InvestmentAmount,
	)

	additionalMessage := investmentFirstValue(
		body.Mensaje,
		body.Message,
	)

	project := investmentFirstValue(
		body.ProyectoInteres,
		body.Project,
		"Inversiones ANCOSUR",
	)

	category := investmentFirstValue(
		body.CategoriaInteres,
		body.Interest,
		"Inversionista",
	)

	source := investmentFirstValue(
		body.FuenteProspeccion,
		body.Source,
		"Web",
	)

	campaignSlug := investmentFirstValue(
		body.CampaniaSlug,
		body.Campaign,
		"inversionistas-web",
	)

	campaignName := investmentFirstValue(
		body.CampaniaNombre,
		"Inversionistas ANCOSUR",
	)

	originRoute := investmentFirstValue(
		body.OrigenRuta,
		c.GetHeader("Referer"),
		"/inversionistas",
	)

	originComponent := investmentFirstValue(
		body.OrigenComponente,
		"Formulario Inversionistas ANCOSUR",
	)

	if len(fullName) < 3 {
		investmentError(
			c,
			http.StatusBadRequest,
			"investment_lead.invalid_name",
			"Ingresa nombres y apellidos válidos.",
			nil,
		)
		return
	}

	if len(phone) != 9 || !strings.HasPrefix(phone, "9") {
		investmentError(
			c,
			http.StatusBadRequest,
			"investment_lead.invalid_phone",
			"El celular debe tener 9 dígitos y empezar con 9.",
			nil,
		)
		return
	}

	if !investmentValidEmail(email) {
		investmentError(
			c,
			http.StatusBadRequest,
			"investment_lead.invalid_email",
			"El correo electrónico no es válido.",
			nil,
		)
		return
	}

	if rangoInversion == "" {
		investmentError(
			c,
			http.StatusBadRequest,
			"investment_lead.range_required",
			"Selecciona un monto de inversión.",
			nil,
		)
		return
	}

	leadMessage :=
		"Solicitud de información para invertir en ANCOSUR. " +
			"Monto de inversión seleccionado: " +
			rangoInversion +
			"."

	if additionalMessage != "" {
		leadMessage += " Mensaje: " + additionalMessage
	} else {
		leadMessage += " Solicita asesoría sobre las opciones de inversión disponibles."
	}

	ctx := c.Request.Context()

	tx, err := h.DB.BeginTx(
		ctx,
		pgx.TxOptions{IsoLevel: pgx.ReadCommitted},
	)
	if err != nil {
		investmentError(
			c,
			http.StatusInternalServerError,
			"investment_lead.transaction_error",
			"No se pudo iniciar el registro.",
			err,
		)
		return
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	stateID, err := investmentGetDefaultLeadStateID(
		ctx,
		tx,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			investmentError(
				c,
				http.StatusInternalServerError,
				"investment_lead.state_not_found",
				"No existe ningún estado registrado en estado_leads.",
				nil,
			)
			return
		}

		investmentError(
			c,
			http.StatusInternalServerError,
			"investment_lead.state_error",
			"No se pudo obtener el estado inicial del lead.",
			err,
		)
		return
	}

	campaignID, err := investmentGetOrCreateCampaignID(
		ctx,
		tx,
		campaignName,
		campaignSlug,
	)
	if err != nil {
		investmentError(
			c,
			http.StatusInternalServerError,
			"investment_lead.campaign_error",
			"No se pudo obtener o crear la campaña.",
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
				fuente_prospeccion,
				lead,
				nombres_completos,
				proyecto_interes,
				telefono,
				email,
				mensaje,
				origen_ruta,
				origen_componente,
				atendido,
				activo,
				created_at,
				updated_at,
				categoria_interes
			)
			VALUES (
				$1::uuid,
				$2::uuid,
				$3,
				$4,
				$5,
				$6,
				$7,
				NULLIF($8, ''),
				$9,
				NULLIF($10, ''),
				NULLIF($11, ''),
				FALSE,
				TRUE,
				NOW(),
				NOW(),
				$12
			)
			RETURNING id::text
		`,
		stateID,
		campaignID,
		source,
		fullName,
		fullName,
		project,
		phone,
		email,
		leadMessage,
		originRoute,
		originComponent,
		category,
	).Scan(&leadID)

	if err != nil {
		investmentError(
			c,
			http.StatusInternalServerError,
			"investment_lead.lead_create_error",
			"No se pudo registrar el lead inversionista.",
			err,
		)
		return
	}

	var investmentID string

	err = tx.QueryRow(
		ctx,
		`
			INSERT INTO solicitudes_inversion (
				lead_id,
				rango_inversion,
				mensaje,
				estado,
				atendido,
				activo,
				created_at,
				updated_at
			)
			VALUES (
				$1::uuid,
				$2,
				NULLIF($3, ''),
				1,
				FALSE,
				TRUE,
				NOW(),
				NOW()
			)
			RETURNING id::text
		`,
		leadID,
		rangoInversion,
		additionalMessage,
	).Scan(&investmentID)

	if err != nil {
		investmentError(
			c,
			http.StatusInternalServerError,
			"investment_lead.investment_create_error",
			"No se pudo registrar la solicitud de inversión.",
			err,
		)
		return
	}

	investment, err := scanInvestment(
		tx.QueryRow(
			ctx,
			investmentSelectQuery+`
				WHERE s.id = $1::uuid
				  AND s.deleted_at IS NULL
				  AND l.deleted_at IS NULL
				LIMIT 1
			`,
			investmentID,
		),
	)
	if err != nil {
		investmentError(
			c,
			http.StatusInternalServerError,
			"investment_lead.read_error",
			"La solicitud fue registrada, pero no se pudo consultar.",
			err,
		)
		return
	}

	if err := tx.Commit(ctx); err != nil {
		investmentError(
			c,
			http.StatusInternalServerError,
			"investment_lead.commit_error",
			"No se pudo confirmar el registro.",
			err,
		)
		return
	}

	investmentSuccess(
		c,
		http.StatusCreated,
		"investment_lead.created",
		"Solicitud de inversión registrada correctamente.",
		gin.H{
			"lead_id":       leadID,
			"investment_id": investmentID,
			"investment":    investment,
		},
	)
}