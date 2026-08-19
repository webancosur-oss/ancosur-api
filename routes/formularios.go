package routes

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

type FormularioWebRequest struct {
	CodigoFormulario string `json:"codigo_formulario" binding:"required"`
	NombreFormulario string `json:"nombre_formulario" binding:"required"`
	TipoFormulario   string `json:"tipo_formulario" binding:"required"`

	Nombre   string `json:"nombre" binding:"required"`
	Telefono string `json:"telefono" binding:"required"`
	Email    string `json:"email"`
	DNI      string `json:"dni"`
	Mensaje  string `json:"mensaje"`

	Proyecto      string `json:"proyecto"`
	TipoInmueble  string `json:"tipo_inmueble"`
	Interes       string `json:"interes"`
	HorarioVisita string `json:"horario_visita"`

	Campania string `json:"campania"`
	Anuncio  string `json:"anuncio"`

	FuenteID int `json:"fuente_id"`

	RutaPagina       string `json:"ruta_pagina"`
	URLPagina        string `json:"url_pagina"`
	PaginaReferencia string `json:"pagina_referencia"`

	AsesorID string `json:"asesor_id"`
	Asesor   string `json:"asesor"`

	UTMSource   string `json:"utm_source"`
	UTMMedium   string `json:"utm_medium"`
	UTMCampaign string `json:"utm_campaign"`
	UTMContent  string `json:"utm_content"`
	UTMTerm     string `json:"utm_term"`
}

type CRMLeadPayload struct {
	FuenteID int `json:"fuente_id"`

	Telefono string `json:"telefono"`
	Nombre   string `json:"nombre"`
	Email    string `json:"email"`
	DNI      string `json:"dni"`

	Campania string `json:"campaña"`
	Anuncio  string `json:"anuncio"`

	MsjClient string `json:"msj_client"`
}

type CRMResult struct {
	Success bool
	LeadID  int64
	Accion  string
	Message string
}

/* RUTAS */

func RutasFormularios(
	api *gin.RouterGroup,
	db *pgxpool.Pool,
) {
	api.POST(
		"/formularios",
		crearFormularioWeb(db),
	)

	api.GET(
		"/formularios",
		listarFormulariosWeb(db),
	)
}

/* POST */

func crearFormularioWeb(
	db *pgxpool.Pool,
) gin.HandlerFunc {
	return func(c *gin.Context) {
		var request FormularioWebRequest

		if err := c.ShouldBindJSON(&request); err != nil {
			c.JSON(
				http.StatusBadRequest,
				gin.H{
					"success": false,
					"message": "Los datos enviados no son válidos.",
					"error":   err.Error(),
				},
			)

			return
		}

		normalizarFormulario(
			&request,
		)

		request.FuenteID = 4

		if request.CodigoFormulario == "" ||
			request.NombreFormulario == "" ||
			request.TipoFormulario == "" {

			c.JSON(
				http.StatusBadRequest,
				gin.H{
					"success": false,
					"message": "Faltan datos del formulario.",
				},
			)

			return
		}

		if len([]rune(request.Nombre)) < 3 {
			c.JSON(
				http.StatusBadRequest,
				gin.H{
					"success": false,
					"message": "El nombre no es válido.",
				},
			)

			return
		}

		if !telefonoFormularioValido(
			request.Telefono,
		) {
			c.JSON(
				http.StatusBadRequest,
				gin.H{
					"success": false,
					"message": "El celular debe tener 9 dígitos y comenzar con 9.",
				},
			)

			return
		}

		if request.DNI != "" &&
			!dniFormularioValido(
				request.DNI,
			) {

			c.JSON(
				http.StatusBadRequest,
				gin.H{
					"success": false,
					"message": "El DNI debe tener 8 dígitos.",
				},
			)

			return
		}

		datosOriginales, err :=
			json.Marshal(
				request,
			)

		if err != nil {
			c.JSON(
				http.StatusInternalServerError,
				gin.H{
					"success": false,
					"message": "No se pudo preparar el formulario.",
				},
			)

			return
		}

		/* 1. GUARDAR EN POSTGRESQL */

		dbCtx, dbCancel :=
			context.WithTimeout(
				context.Background(),
				8*time.Second,
			)

		defer dbCancel()

		var id string
		var creadoEn time.Time

		err =
			db.QueryRow(
				dbCtx,
				`
				INSERT INTO leads_web (
					codigo_formulario,
					nombre_formulario,
					tipo_formulario,

					nombre_completo,
					celular,
					correo,
					documento,
					mensaje,

					proyecto,
					tipo_inmueble,
					interes,
					horario_visita,

					campaña,
					anuncio,

					fuente_id,
					fuente_descripcion,

					ruta_pagina,
					url_pagina,
					pagina_referencia,

					utm_source,
					utm_medium,
					utm_campaign,
					utm_content,
					utm_term,

					estado_crm,

					codigo_http_crm,
					respuesta_crm,
					error_crm,
					enviado_crm_en,

					asesor_id,
					datos_originales,

					crm_lead_id
				)
				VALUES (
					$1,
					$2,
					$3,

					$4,
					$5,
					NULLIF($6, ''),
					NULLIF($7, ''),
					NULLIF($8, ''),

					NULLIF($9, ''),
					NULLIF($10, ''),
					NULLIF($11, ''),
					NULLIF($12, ''),

					NULLIF($13, ''),
					NULLIF($14, ''),

					4,
					'PAGINA WEB',

					NULLIF($15, ''),
					NULLIF($16, ''),
					NULLIF($17, ''),

					NULLIF($18, ''),
					NULLIF($19, ''),
					NULLIF($20, ''),
					NULLIF($21, ''),
					NULLIF($22, ''),

					'pendiente',

					NULL,
					NULL,
					NULL,
					NULL,

					NULLIF($23, '')::uuid,
					$24::jsonb,

					NULL
				)
				RETURNING
					id::text,
					creado_en
				`,
				request.CodigoFormulario,
				request.NombreFormulario,
				request.TipoFormulario,

				request.Nombre,
				request.Telefono,
				request.Email,
				request.DNI,
				request.Mensaje,

				request.Proyecto,
				request.TipoInmueble,
				request.Interes,
				request.HorarioVisita,

				request.Campania,
				request.Anuncio,

				request.RutaPagina,
				request.URLPagina,
				request.PaginaReferencia,

				request.UTMSource,
				request.UTMMedium,
				request.UTMCampaign,
				request.UTMContent,
				request.UTMTerm,

				request.AsesorID,

				string(
					datosOriginales,
				),
			).
				Scan(
					&id,
					&creadoEn,
				)

		if err != nil {
			c.JSON(
				http.StatusInternalServerError,
				gin.H{
					"success": false,
					"message": "No se pudo guardar el formulario en la base de datos.",
					"error":   err.Error(),
				},
			)

			return
		}

		/* 2. PREPARAR CRM */

		crmURL :=
			strings.TrimSpace(
				os.Getenv(
					"LEADS_API_URL",
				),
			)

		if crmURL == "" {
			guardarCRMError(
				db,
				id,
				0,
				nil,
				"CRM_NO_CONFIGURADO",
				"LEADS_API_URL no está configurada.",
			)

			responderCRMError(
				c,
				id,
				creadoEn,
				"CRM_NO_CONFIGURADO",
				"El CRM no está configurado.",
				0,
			)

			return
		}

		clientMetadata :=
			map[string]string{
				"origenRuta": request.RutaPagina,

				"origenComponente": request.NombreFormulario,

				"tipoLead": "WEB Ancosur",
			}

		if request.Proyecto != "" {
			clientMetadata["proyecto"] =
				request.Proyecto
		}

		if request.TipoInmueble != "" {
			clientMetadata["tipoInmueble"] =
				request.TipoInmueble
		}

		if request.Interes != "" {
			clientMetadata["interes"] =
				request.Interes
		}

		if request.HorarioVisita != "" {
			clientMetadata["horarioVisita"] =
				request.HorarioVisita
		}

		if request.Mensaje != "" {
			clientMetadata["mensaje"] =
				request.Mensaje
		}

		msjClient, err :=
			json.Marshal(
				clientMetadata,
			)

		if err != nil {
			guardarCRMError(
				db,
				id,
				0,
				nil,
				"CRM_PAYLOAD_ERROR",
				err.Error(),
			)

			responderCRMError(
				c,
				id,
				creadoEn,
				"CRM_PAYLOAD_ERROR",
				err.Error(),
				0,
			)

			return
		}

		crmPayload :=
			CRMLeadPayload{
				FuenteID: 4,

				Telefono: request.Telefono,

				Nombre: request.Nombre,

				Email: request.Email,

				DNI: request.DNI,

				Campania: request.Campania,

				Anuncio: request.Anuncio,

				MsjClient: string(
					msjClient,
				),
			}

		crmBody, err :=
			json.Marshal(
				crmPayload,
			)

		if err != nil {
			guardarCRMError(
				db,
				id,
				0,
				nil,
				"CRM_PAYLOAD_ERROR",
				err.Error(),
			)

			responderCRMError(
				c,
				id,
				creadoEn,
				"CRM_PAYLOAD_ERROR",
				err.Error(),
				0,
			)

			return
		}

		/* 3. ENVIAR AL CRM */

		crmCtx, crmCancel :=
			context.WithTimeout(
				context.Background(),
				20*time.Second,
			)

		defer crmCancel()

		crmRequest, err :=
			http.NewRequestWithContext(
				crmCtx,
				http.MethodPost,
				crmURL,
				bytes.NewReader(
					crmBody,
				),
			)

		if err != nil {
			guardarCRMError(
				db,
				id,
				0,
				nil,
				"CRM_REQUEST_ERROR",
				err.Error(),
			)

			responderCRMError(
				c,
				id,
				creadoEn,
				"CRM_REQUEST_ERROR",
				err.Error(),
				0,
			)

			return
		}

		crmRequest.Header.Set(
			"Content-Type",
			"application/json",
		)

		crmRequest.Header.Set(
			"Accept",
			"application/json",
		)

		crmClient :=
			&http.Client{
				Timeout: 20 * time.Second,
			}

		crmHTTPResponse, err :=
			crmClient.Do(
				crmRequest,
			)

		if err != nil {
			guardarCRMError(
				db,
				id,
				0,
				nil,
				"CRM_NO_DISPONIBLE",
				err.Error(),
			)

			responderCRMError(
				c,
				id,
				creadoEn,
				"CRM_NO_DISPONIBLE",
				"No fue posible conectar con el CRM.",
				0,
			)

			return
		}

		defer crmHTTPResponse.Body.Close()

		responseBody, err :=
			io.ReadAll(
				crmHTTPResponse.Body,
			)

		if err != nil {
			guardarCRMError(
				db,
				id,
				crmHTTPResponse.StatusCode,
				nil,
				"CRM_RESPONSE_READ_ERROR",
				err.Error(),
			)

			responderCRMError(
				c,
				id,
				creadoEn,
				"CRM_RESPONSE_READ_ERROR",
				err.Error(),
				crmHTTPResponse.StatusCode,
			)

			return
		}

		fmt.Println(
			"CRM HTTP:",
			crmHTTPResponse.StatusCode,
		)

		fmt.Println(
			"CRM RESPONSE:",
			string(responseBody),
		)

		/* 4. INTERPRETAR RESPUESTA */

		crmResult :=
			analizarRespuestaCRM(
				crmHTTPResponse.StatusCode,
				responseBody,
			)

		if !crmResult.Success {
			guardarCRMError(
				db,
				id,
				crmHTTPResponse.StatusCode,
				responseBody,
				"CRM_REJECTED",
				crmResult.Message,
			)

			responderCRMError(
				c,
				id,
				creadoEn,
				"CRM_REJECTED",
				crmResult.Message,
				crmHTTPResponse.StatusCode,
			)

			return
		}

		/* 5. CRM CONFIRMADO */

		updateCtx, updateCancel :=
			context.WithTimeout(
				context.Background(),
				5*time.Second,
			)

		defer updateCancel()

		_, updateErr :=
			db.Exec(
				updateCtx,
				`
				UPDATE leads_web
				SET
					estado_crm = 'enviado',

					crm_lead_id =
						NULLIF($2, 0),

					codigo_http_crm =
						$3,

					respuesta_crm =
						$4::jsonb,

					error_crm =
						NULL,

					enviado_crm_en =
						NOW(),

					actualizado_en =
						NOW()

				WHERE id =
					$1
				`,
				id,
				crmResult.LeadID,
				crmHTTPResponse.StatusCode,
				string(
					responseBody,
				),
			)

		if updateErr != nil {
			c.JSON(
				http.StatusCreated,
				gin.H{
					"success": true,

					"message": "El lead llegó al CRM, pero no se pudo actualizar el estado local.",

					"data": gin.H{
						"id": id,

						"creado_en": creadoEn,

						"guardado_local": true,

						"estado_crm": "enviado",

						"crm": gin.H{
							"success": true,

							"estado": "enviado",

							"lead_id": crmResult.LeadID,

							"accion": crmResult.Accion,

							"http_status": crmHTTPResponse.StatusCode,

							"message": crmResult.Message,
						},

						"error_actualizacion_local": updateErr.Error(),
					},
				},
			)

			return
		}

		c.JSON(
			http.StatusCreated,
			gin.H{
				"success": true,

				"message": "Formulario guardado y enviado al CRM correctamente.",

				"data": gin.H{
					"id": id,

					"creado_en": creadoEn,

					"guardado_local": true,

					"estado_crm": "enviado",

					"crm": gin.H{
						"success": true,

						"estado": "enviado",

						"lead_id": crmResult.LeadID,

						"accion": crmResult.Accion,

						"http_status": crmHTTPResponse.StatusCode,

						"message": crmResult.Message,
					},
				},
			},
		)
	}
}

/*
	Permite las distintas estructuras
	que puede devolver Sentinel.
*/

func analizarRespuestaCRM(
	httpStatus int,
	responseBody []byte,
) CRMResult {
	result :=
		CRMResult{
			Success: httpStatus >= 200 &&
				httpStatus < 300,

			Message: http.StatusText(
				httpStatus,
			),
		}

	if len(responseBody) == 0 {
		return result
	}

	var root map[string]any

	if err :=
		json.Unmarshal(
			responseBody,
			&root,
		); err != nil {

		if !result.Success {
			result.Message =
				string(
					responseBody,
				)
		}

		return result
	}

	/*
		Si success existe en el nivel
		superior y es false, es error.
	*/
	if value,
		exists :=
		root["success"]; exists {

		if success,
			ok :=
			value.(bool); ok {

			if !success {
				result.Success =
					false
			}
		}
	}

	if message :=
		obtenerString(
			root,
			"message",
		); message != "" {

		result.Message =
			message
	}

	/*
		Buscar ID / acción / success
		dentro de data.
	*/
	if data,
		ok :=
		root["data"].(map[string]any); ok {

		if id :=
			obtenerInt64(
				data,
				"id",
			); id > 0 {

			result.LeadID =
				id
		}

		if accion :=
			obtenerString(
				data,
				"accion",
			); accion != "" {

			result.Accion =
				accion
		}

		if message :=
			obtenerString(
				data,
				"message",
			); message != "" {

			result.Message =
				message
		}

		/*
			Solo consideramos error
			si data.success viene
			explícitamente en false
			Y el success superior también
			es false o el HTTP falló.

			No castigamos un data.success
			ausente.
		*/
		if nestedData,
			ok :=
			data["data"].(map[string]any); ok {

			if result.LeadID == 0 {
				result.LeadID =
					obtenerInt64(
						nestedData,
						"id",
					)
			}
		}
	}

	/*
		Si HTTP es 2xx y success superior
		no fue false, se considera enviado.
	*/
	if httpStatus >= 200 &&
		httpStatus < 300 {

		if topSuccess,
			exists :=
			root["success"]; exists {

			if value,
				ok :=
				topSuccess.(bool); ok &&
				!value {

				result.Success =
					false

				return result
			}
		}

		result.Success =
			true
	}

	if result.Message == "" {
		if result.Success {
			result.Message =
				"Lead enviado correctamente al CRM."
		} else {
			result.Message =
				"El CRM rechazó el lead."
		}
	}

	return result
}

func obtenerString(
	data map[string]any,
	key string,
) string {
	value,
		exists :=
		data[key]

	if !exists ||
		value == nil {
		return ""
	}

	text,
		ok :=
		value.(string)

	if !ok {
		return ""
	}

	return strings.TrimSpace(
		text,
	)
}

func obtenerInt64(
	data map[string]any,
	key string,
) int64 {
	value,
		exists :=
		data[key]

	if !exists ||
		value == nil {
		return 0
	}

	switch number :=
		value.(type) {

	case float64:
		return int64(
			number,
		)

	case int:
		return int64(
			number,
		)

	case int64:
		return number

	case json.Number:
		value, _ :=
			number.Int64()

		return value
	}

	return 0
}

func guardarCRMError(
	db *pgxpool.Pool,
	id string,
	httpStatus int,
	responseBody []byte,
	errorCode string,
	errorMessage string,
) {
	respuesta :=
		map[string]any{
			"success": false,

			"estado": "error",

			"codigo": errorCode,

			"message": errorMessage,

			"fecha": time.Now(),
		}

	if httpStatus > 0 {
		respuesta["http_status"] =
			httpStatus
	}

	if len(responseBody) > 0 {
		var external any

		if json.Unmarshal(
			responseBody,
			&external,
		) == nil {

			respuesta["respuesta_externa"] =
				external
		} else {
			respuesta["respuesta_externa"] =
				string(
					responseBody,
				)
		}
	}

	respuestaJSON, _ :=
		json.Marshal(
			respuesta,
		)

	ctx, cancel :=
		context.WithTimeout(
			context.Background(),
			5*time.Second,
		)

	defer cancel()

	_, _ =
		db.Exec(
			ctx,
			`
			UPDATE leads_web
			SET
				estado_crm =
					'error',

				crm_lead_id =
					NULL,

				codigo_http_crm =
					NULLIF($2, 0),

				respuesta_crm =
					$3::jsonb,

				error_crm =
					$4,

				enviado_crm_en =
					NULL,

				actualizado_en =
					NOW()

			WHERE id =
				$1
			`,
			id,
			httpStatus,
			string(
				respuestaJSON,
			),
			errorMessage,
		)
}

func responderCRMError(
	c *gin.Context,
	id string,
	creadoEn time.Time,
	codigo string,
	mensaje string,
	httpStatus int,
) {
	crm :=
		gin.H{
			"success": false,

			"estado": "error",

			"codigo": codigo,

			"message": mensaje,
		}

	if httpStatus > 0 {
		crm["http_status"] =
			httpStatus
	}

	c.JSON(
		http.StatusCreated,
		gin.H{
			"success": true,

			"message": "Formulario guardado correctamente.",

			"data": gin.H{
				"id": id,

				"creado_en": creadoEn,

				"guardado_local": true,

				"estado_crm": "error",

				"crm": crm,
			},
		},
	)
}

func normalizarFormulario(
	request *FormularioWebRequest,
) {
	request.CodigoFormulario =
		strings.TrimSpace(
			request.CodigoFormulario,
		)

	request.NombreFormulario =
		strings.TrimSpace(
			request.NombreFormulario,
		)

	request.TipoFormulario =
		strings.ToLower(
			strings.TrimSpace(
				request.TipoFormulario,
			),
		)

	request.Nombre =
		strings.TrimSpace(
			request.Nombre,
		)

	request.Telefono =
		soloNumerosFormulario(
			request.Telefono,
		)

	request.Email =
		strings.ToLower(
			strings.TrimSpace(
				request.Email,
			),
		)

	request.DNI =
		soloNumerosFormulario(
			request.DNI,
		)

	request.Mensaje =
		strings.TrimSpace(
			request.Mensaje,
		)

	request.Proyecto =
		strings.TrimSpace(
			request.Proyecto,
		)

	request.TipoInmueble =
		strings.TrimSpace(
			request.TipoInmueble,
		)

	request.Interes =
		strings.TrimSpace(
			request.Interes,
		)

	request.HorarioVisita =
		strings.TrimSpace(
			request.HorarioVisita,
		)

	request.Campania =
		strings.TrimSpace(
			request.Campania,
		)

	request.Anuncio =
		strings.TrimSpace(
			request.Anuncio,
		)

	request.RutaPagina =
		strings.TrimSpace(
			request.RutaPagina,
		)

	request.URLPagina =
		strings.TrimSpace(
			request.URLPagina,
		)

	request.PaginaReferencia =
		strings.TrimSpace(
			request.PaginaReferencia,
		)

	request.AsesorID =
		strings.TrimSpace(
			request.AsesorID,
		)

	request.Asesor =
		strings.TrimSpace(
			request.Asesor,
		)

	request.UTMSource =
		strings.TrimSpace(
			request.UTMSource,
		)

	request.UTMMedium =
		strings.TrimSpace(
			request.UTMMedium,
		)

	request.UTMCampaign =
		strings.TrimSpace(
			request.UTMCampaign,
		)

	request.UTMContent =
		strings.TrimSpace(
			request.UTMContent,
		)

	request.UTMTerm =
		strings.TrimSpace(
			request.UTMTerm,
		)
}

func soloNumerosFormulario(
	value string,
) string {
	var result strings.Builder

	for _, char := range value {

		if char >= '0' &&
			char <= '9' {

			result.WriteRune(
				char,
			)
		}
	}

	return result.String()
}

func telefonoFormularioValido(
	telefono string,
) bool {
	if len(telefono) != 9 {
		return false
	}

	if telefono[0] != '9' {
		return false
	}

	for _, char := range telefono {

		if char < '0' ||
			char > '9' {

			return false
		}
	}

	return true
}

func dniFormularioValido(
	dni string,
) bool {
	if len(dni) != 8 {
		return false
	}

	for _, char := range dni {

		if char < '0' ||
			char > '9' {

			return false
		}
	}

	return true
}

/* GET */

func listarFormulariosWeb(
	db *pgxpool.Pool,
) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel :=
			context.WithTimeout(
				c.Request.Context(),
				8*time.Second,
			)

		defer cancel()

		rows, err :=
			db.Query(
				ctx,
				`
		SELECT
			l.id::text,

			l.codigo_formulario,
			l.nombre_formulario,
			l.tipo_formulario,

			l.nombre_completo,
			l.celular,
			COALESCE(l.correo, ''),
			COALESCE(l.documento, ''),
			COALESCE(l.mensaje, ''),

			COALESCE(l.proyecto, ''),
			COALESCE(l.tipo_inmueble, ''),
			COALESCE(l.interes, ''),
			COALESCE(l.horario_visita, ''),

			COALESCE(l.campaña, ''),
			COALESCE(l.anuncio, ''),

			l.fuente_id,

			COALESCE(
				l.fuente_descripcion,
				''
			),

			COALESCE(l.ruta_pagina, ''),
			COALESCE(l.url_pagina, ''),

			COALESCE(l.utm_source, ''),
			COALESCE(l.utm_medium, ''),
			COALESCE(l.utm_campaign, ''),

			l.estado_crm,

			COALESCE(
				l.codigo_http_crm,
				0
			),

			COALESCE(
				l.crm_lead_id,
				0
			),

			COALESCE(
				l.error_crm,
				''
			),

			l.creado_en,

			l.enviado_crm_en,

			COALESCE(
				l.asesor_id::text,
				''
			) AS asesor_id,

			COALESCE(
				a.nombres_completos,
				''
			) AS asesor

		FROM leads_web AS l

		LEFT JOIN asesores AS a
			ON a.id = l.asesor_id

		ORDER BY
			l.creado_en DESC
		`,
			)

		if err != nil {
			c.JSON(
				http.StatusInternalServerError,
				gin.H{
					"success": false,
					"message": "No se pudieron cargar los leads.",
					"error":   err.Error(),
				},
			)

			return
		}

		defer rows.Close()

		type LeadWeb struct {
			ID string `json:"id"`

			CodigoFormulario string `json:"codigo_formulario"`
			NombreFormulario string `json:"nombre_formulario"`
			TipoFormulario   string `json:"tipo_formulario"`

			Nombre   string `json:"nombre"`
			Telefono string `json:"telefono"`
			Email    string `json:"email"`
			DNI      string `json:"dni"`
			Mensaje  string `json:"mensaje"`

			Proyecto      string `json:"proyecto"`
			TipoInmueble  string `json:"tipo_inmueble"`
			Interes       string `json:"interes"`
			HorarioVisita string `json:"horario_visita"`

			Campania string `json:"campania"`
			Anuncio  string `json:"anuncio"`

			FuenteID int `json:"fuente_id"`

			FuenteDescripcion string `json:"fuente_descripcion"`

			RutaPagina string `json:"ruta_pagina"`
			URLPagina  string `json:"url_pagina"`

			UTMSource   string `json:"utm_source"`
			UTMMedium   string `json:"utm_medium"`
			UTMCampaign string `json:"utm_campaign"`

			EstadoCRM string `json:"estado_crm"`

			CodigoHTTPCRM int `json:"codigo_http_crm"`

			CRMLeadID int64 `json:"crm_lead_id"`

			ErrorCRM string `json:"error_crm"`

			CreadoEn time.Time `json:"creado_en"`

			EnviadoCRMEn *time.Time `json:"enviado_crm_en"`

			AsesorID string `json:"asesor_id"`
			Asesor   string `json:"asesor"`
		}

		leads :=
			make(
				[]LeadWeb,
				0,
			)

		for rows.Next() {
			var lead LeadWeb

			err :=
				rows.Scan(
					&lead.ID,

					&lead.CodigoFormulario,
					&lead.NombreFormulario,
					&lead.TipoFormulario,

					&lead.Nombre,
					&lead.Telefono,
					&lead.Email,
					&lead.DNI,
					&lead.Mensaje,

					&lead.Proyecto,
					&lead.TipoInmueble,
					&lead.Interes,
					&lead.HorarioVisita,

					&lead.Campania,
					&lead.Anuncio,

					&lead.FuenteID,
					&lead.FuenteDescripcion,

					&lead.RutaPagina,
					&lead.URLPagina,

					&lead.UTMSource,
					&lead.UTMMedium,
					&lead.UTMCampaign,

					&lead.EstadoCRM,

					&lead.CodigoHTTPCRM,

					&lead.CRMLeadID,

					&lead.ErrorCRM,

					&lead.CreadoEn,

					&lead.EnviadoCRMEn,

					&lead.AsesorID,
					&lead.Asesor,
				)

			if err != nil {
				c.JSON(
					http.StatusInternalServerError,
					gin.H{
						"success": false,
						"message": "No se pudo leer un lead.",
						"error":   err.Error(),
					},
				)

				return
			}

			leads =
				append(
					leads,
					lead,
				)
		}

		if err := rows.Err(); err != nil {
			c.JSON(
				http.StatusInternalServerError,
				gin.H{
					"success": false,
					"message": "No se pudieron recorrer los leads.",
					"error":   err.Error(),
				},
			)

			return
		}

		c.JSON(
			http.StatusOK,
			gin.H{
				"success": true,
				"total":   len(leads),
				"data":    leads,
			},
		)
	}
}