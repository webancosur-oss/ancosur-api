package routes

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

/* =========================================================
   CONFIGURACIÓN
========================================================= */

const maxCVSize int64 = 5 * 1024 * 1024 // 5 MB

/* =========================================================
   MODELO DE RESPUESTA
========================================================= */

type PostulacionTrabajo struct {
	ID string `json:"id"`

	VacanteID     *string `json:"vacante_id"`
	VacanteTitulo *string `json:"vacante_titulo"`
	Area          *string `json:"area"`

	NombreCompleto string `json:"nombre_completo"`
	Celular        string `json:"celular"`
	Correo         string `json:"correo"`

	Mensaje *string `json:"mensaje"`

	CVNombre string `json:"cv_nombre"`
	CVTipo   string `json:"cv_tipo"`
	CVTamano int64  `json:"cv_tamano"`

	Estado string `json:"estado"`

	RutaPagina       *string `json:"ruta_pagina"`
	URLPagina        *string `json:"url_pagina"`
	PaginaReferencia *string `json:"pagina_referencia"`

	UTMSource   *string `json:"utm_source"`
	UTMMedium   *string `json:"utm_medium"`
	UTMCampaign *string `json:"utm_campaign"`
	UTMContent  *string `json:"utm_content"`
	UTMTerm     *string `json:"utm_term"`

	CreadoEn      time.Time `json:"creado_en"`
	ActualizadoEn time.Time `json:"actualizado_en"`
}

/* =========================================================
   RUTAS
========================================================= */

func RutasPostulaciones(
	api *gin.RouterGroup,
	db *pgxpool.Pool,
) {
	postulaciones := api.Group(
		"/postulaciones",
	)

	// Crear postulación.
	postulaciones.POST(
		"",
		crearPostulacion(db),
	)

	// Listar postulaciones.
	postulaciones.GET(
		"",
		listarPostulaciones(db),
	)

	// Obtener una postulación.
	postulaciones.GET(
		"/:id",
		obtenerPostulacion(db),
	)

	// Visualizar CV PDF.
	postulaciones.GET(
		"/:id/cv",
		verCVPostulacion(db),
	)

	// Cambiar fase / estado de la postulación.
	postulaciones.PATCH(
		"/:id/estado",
		actualizarEstadoPostulacion(db),
	)
}

/* =========================================================
   POST /api/postulaciones
========================================================= */

func crearPostulacion(
	db *pgxpool.Pool,
) gin.HandlerFunc {
	return func(c *gin.Context) {
		if db == nil {
			c.JSON(
				http.StatusInternalServerError,
				gin.H{
					"success": false,
					"message": "La base de datos no está disponible.",
				},
			)

			return
		}

		/*
			Permitimos 5 MB para el CV
			más espacio adicional para los
			campos multipart.
		*/
		c.Request.Body =
			http.MaxBytesReader(
				c.Writer,
				c.Request.Body,
				maxCVSize+(1024*1024),
			)

		if err :=
			c.Request.ParseMultipartForm(
				maxCVSize,
			); err != nil {

			c.JSON(
				http.StatusBadRequest,
				gin.H{
					"success": false,
					"message": "El formulario o el CV supera el tamaño permitido.",
				},
			)

			return
		}

		/* =========================================
		   DATOS
		========================================= */

		vacanteID :=
			limpiarTextoPostulacion(
				c.PostForm(
					"vacante_id",
				),
			)

		vacanteTitulo :=
			limpiarTextoPostulacion(
				c.PostForm(
					"vacante_titulo",
				),
			)

		area :=
			limpiarTextoPostulacion(
				c.PostForm(
					"area",
				),
			)

		nombre :=
			limpiarTextoPostulacion(
				c.PostForm(
					"nombre",
				),
			)

		telefono :=
			soloNumerosPostulacion(
				c.PostForm(
					"telefono",
				),
			)

		email :=
			strings.ToLower(
				limpiarTextoPostulacion(
					c.PostForm(
						"email",
					),
				),
			)

		mensaje :=
			strings.TrimSpace(
				c.PostForm(
					"mensaje",
				),
			)

		rutaPagina :=
			strings.TrimSpace(
				c.PostForm(
					"ruta_pagina",
				),
			)

		urlPagina :=
			strings.TrimSpace(
				c.PostForm(
					"url_pagina",
				),
			)

		paginaReferencia :=
			strings.TrimSpace(
				c.PostForm(
					"pagina_referencia",
				),
			)

		utmSource :=
			strings.TrimSpace(
				c.PostForm(
					"utm_source",
				),
			)

		utmMedium :=
			strings.TrimSpace(
				c.PostForm(
					"utm_medium",
				),
			)

		utmCampaign :=
			strings.TrimSpace(
				c.PostForm(
					"utm_campaign",
				),
			)

		utmContent :=
			strings.TrimSpace(
				c.PostForm(
					"utm_content",
				),
			)

		utmTerm :=
			strings.TrimSpace(
				c.PostForm(
					"utm_term",
				),
			)

		/* =========================================
		   VALIDACIONES
		========================================= */

		if len([]rune(nombre)) < 3 {
			c.JSON(
				http.StatusBadRequest,
				gin.H{
					"success": false,
					"message": "Ingresa un nombre completo válido.",
				},
			)

			return
		}

		if len([]rune(nombre)) > 120 {
			c.JSON(
				http.StatusBadRequest,
				gin.H{
					"success": false,
					"message": "El nombre completo es demasiado largo.",
				},
			)

			return
		}

		if !telefonoPostulacionValido(
			telefono,
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

		if !correoPostulacionValido(
			email,
		) {
			c.JSON(
				http.StatusBadRequest,
				gin.H{
					"success": false,
					"message": "Ingresa un correo electrónico válido.",
				},
			)

			return
		}

		if len([]rune(mensaje)) > 500 {
			c.JSON(
				http.StatusBadRequest,
				gin.H{
					"success": false,
					"message": "El mensaje no debe superar los 500 caracteres.",
				},
			)

			return
		}

		/* =========================================
		   CV
		========================================= */

		fileHeader, err :=
			c.FormFile(
				"cv",
			)

		if err != nil {
			c.JSON(
				http.StatusBadRequest,
				gin.H{
					"success": false,
					"message": "Debes adjuntar tu CV en formato PDF.",
				},
			)

			return
		}

		if fileHeader.Size <= 0 {
			c.JSON(
				http.StatusBadRequest,
				gin.H{
					"success": false,
					"message": "El archivo del CV está vacío.",
				},
			)

			return
		}

		if fileHeader.Size >
			maxCVSize {

			c.JSON(
				http.StatusBadRequest,
				gin.H{
					"success": false,
					"message": "El CV no debe superar los 5 MB.",
				},
			)

			return
		}

		file, err :=
			fileHeader.Open()

		if err != nil {
			c.JSON(
				http.StatusBadRequest,
				gin.H{
					"success": false,
					"message": "No se pudo abrir el CV.",
				},
			)

			return
		}

		defer file.Close()

		/*
			LimitReader evita que un archivo
			más grande se cargue completamente
			en memoria.
		*/
		cvData, err :=
			io.ReadAll(
				io.LimitReader(
					file,
					maxCVSize+1,
				),
			)

		if err != nil {
			c.JSON(
				http.StatusBadRequest,
				gin.H{
					"success": false,
					"message": "No se pudo leer el CV.",
				},
			)

			return
		}

		if len(cvData) == 0 {
			c.JSON(
				http.StatusBadRequest,
				gin.H{
					"success": false,
					"message": "El CV está vacío.",
				},
			)

			return
		}

		if int64(len(cvData)) >
			maxCVSize {

			c.JSON(
				http.StatusBadRequest,
				gin.H{
					"success": false,
					"message": "El CV no debe superar los 5 MB.",
				},
			)

			return
		}

		/* =========================================
		   VALIDAR PDF REAL
		========================================= */

		contentType :=
			http.DetectContentType(
				cvData,
			)

		if contentType !=
			"application/pdf" {

			c.JSON(
				http.StatusBadRequest,
				gin.H{
					"success": false,
					"message": "El CV debe estar en formato PDF.",
				},
			)

			return
		}

		cvNombre :=
			limpiarNombreArchivo(
				fileHeader.Filename,
			)

		if cvNombre == "" {
			cvNombre =
				"curriculum.pdf"
		}

		if !strings.HasSuffix(
			strings.ToLower(
				cvNombre,
			),
			".pdf",
		) {
			cvNombre += ".pdf"
		}

		/* =========================================
		   INSERTAR
		========================================= */

		ctx, cancel :=
			context.WithTimeout(
				c.Request.Context(),
				10*time.Second,
			)

		defer cancel()

		var id string
		var creadoEn time.Time

		err =
			db.QueryRow(
				ctx,
				`
				INSERT INTO postulaciones_trabajo (
					vacante_id,
					vacante_titulo,
					area,

					nombre_completo,
					celular,
					correo,
					mensaje,

					cv_nombre,
					cv_tipo,
					cv_tamano,
					cv_data,

					estado,

					ruta_pagina,
					url_pagina,
					pagina_referencia,

					utm_source,
					utm_medium,
					utm_campaign,
					utm_content,
					utm_term
				)
				VALUES (
					NULLIF($1, ''),
					NULLIF($2, ''),
					NULLIF($3, ''),

					$4,
					$5,
					$6,
					NULLIF($7, ''),

					$8,
					$9,
					$10,
					$11,

					'recibido',

					NULLIF($12, ''),
					NULLIF($13, ''),
					NULLIF($14, ''),

					NULLIF($15, ''),
					NULLIF($16, ''),
					NULLIF($17, ''),
					NULLIF($18, ''),
					NULLIF($19, '')
				)
				RETURNING
					id,
					creado_en
				`,
				vacanteID,
				vacanteTitulo,
				area,

				nombre,
				telefono,
				email,
				mensaje,

				cvNombre,
				contentType,
				len(cvData),
				cvData,

				rutaPagina,
				urlPagina,
				paginaReferencia,

				utmSource,
				utmMedium,
				utmCampaign,
				utmContent,
				utmTerm,
			).
				Scan(
					&id,
					&creadoEn,
				)

		if err != nil {
			fmt.Println(
				"Error guardando postulación:",
				err,
			)

			c.JSON(
				http.StatusInternalServerError,
				gin.H{
					"success": false,
					"message": "No se pudo registrar la postulación.",
					"error":   err.Error(),
				},
			)

			return
		}

		/* =========================================
		   RESPUESTA
		========================================= */

		c.JSON(
			http.StatusCreated,
			gin.H{
				"success": true,

				"message": "Tu postulación fue registrada correctamente.",

				"data": gin.H{
					"id": id,

					"estado": "recibido",

					"vacante": vacanteTitulo,

					"cv_nombre": cvNombre,

					"cv_tipo": contentType,

					"cv_tamano": len(cvData),

					"cv_guardado": true,

					"creado_en": creadoEn,
				},
			},
		)
	}
}

/* =========================================================
   GET /api/postulaciones
========================================================= */

func listarPostulaciones(
	db *pgxpool.Pool,
) gin.HandlerFunc {
	return func(c *gin.Context) {
		if db == nil {
			c.JSON(
				http.StatusInternalServerError,
				gin.H{
					"success": false,
					"message": "La base de datos no está disponible.",
				},
			)

			return
		}

		ctx, cancel :=
			context.WithTimeout(
				c.Request.Context(),
				10*time.Second,
			)

		defer cancel()

		rows, err :=
			db.Query(
				ctx,
				`
				SELECT
					id,

					vacante_id,
					vacante_titulo,
					area,

					nombre_completo,
					celular,
					correo,
					mensaje,

					cv_nombre,
					cv_tipo,
					cv_tamano,

					estado,

					ruta_pagina,
					url_pagina,
					pagina_referencia,

					utm_source,
					utm_medium,
					utm_campaign,
					utm_content,
					utm_term,

					creado_en,
					actualizado_en

				FROM
					postulaciones_trabajo

				ORDER BY
					creado_en DESC
				`,
			)

		if err != nil {
			c.JSON(
				http.StatusInternalServerError,
				gin.H{
					"success": false,
					"message": "No se pudieron cargar las postulaciones.",
					"error":   err.Error(),
				},
			)

			return
		}

		defer rows.Close()

		postulaciones :=
			make(
				[]PostulacionTrabajo,
				0,
			)

		for rows.Next() {
			var item PostulacionTrabajo

			err :=
				rows.Scan(
					&item.ID,

					&item.VacanteID,
					&item.VacanteTitulo,
					&item.Area,

					&item.NombreCompleto,
					&item.Celular,
					&item.Correo,
					&item.Mensaje,

					&item.CVNombre,
					&item.CVTipo,
					&item.CVTamano,

					&item.Estado,

					&item.RutaPagina,
					&item.URLPagina,
					&item.PaginaReferencia,

					&item.UTMSource,
					&item.UTMMedium,
					&item.UTMCampaign,
					&item.UTMContent,
					&item.UTMTerm,

					&item.CreadoEn,
					&item.ActualizadoEn,
				)

			if err != nil {
				c.JSON(
					http.StatusInternalServerError,
					gin.H{
						"success": false,
						"message": "No se pudieron procesar las postulaciones.",
						"error":   err.Error(),
					},
				)

				return
			}

			postulaciones =
				append(
					postulaciones,
					item,
				)
		}

		if err :=
			rows.Err(); err != nil {

			c.JSON(
				http.StatusInternalServerError,
				gin.H{
					"success": false,
					"message": "Ocurrió un error leyendo las postulaciones.",
					"error":   err.Error(),
				},
			)

			return
		}

		c.JSON(
			http.StatusOK,
			gin.H{
				"success": true,

				"total": len(
					postulaciones,
				),

				"data": postulaciones,
			},
		)
	}
}

/* =========================================================
   GET /api/postulaciones/:id
========================================================= */

func obtenerPostulacion(
	db *pgxpool.Pool,
) gin.HandlerFunc {
	return func(c *gin.Context) {
		if db == nil {
			c.JSON(
				http.StatusInternalServerError,
				gin.H{
					"success": false,
					"message": "La base de datos no está disponible.",
				},
			)

			return
		}

		id :=
			strings.TrimSpace(
				c.Param(
					"id",
				),
			)

		if id == "" {
			c.JSON(
				http.StatusBadRequest,
				gin.H{
					"success": false,
					"message": "ID de postulación inválido.",
				},
			)

			return
		}

		ctx, cancel :=
			context.WithTimeout(
				c.Request.Context(),
				10*time.Second,
			)

		defer cancel()

		var item PostulacionTrabajo

		err :=
			db.QueryRow(
				ctx,
				`
				SELECT
					id,

					vacante_id,
					vacante_titulo,
					area,

					nombre_completo,
					celular,
					correo,
					mensaje,

					cv_nombre,
					cv_tipo,
					cv_tamano,

					estado,

					ruta_pagina,
					url_pagina,
					pagina_referencia,

					utm_source,
					utm_medium,
					utm_campaign,
					utm_content,
					utm_term,

					creado_en,
					actualizado_en

				FROM
					postulaciones_trabajo

				WHERE
					id = $1

				LIMIT 1
				`,
				id,
			).
				Scan(
					&item.ID,

					&item.VacanteID,
					&item.VacanteTitulo,
					&item.Area,

					&item.NombreCompleto,
					&item.Celular,
					&item.Correo,
					&item.Mensaje,

					&item.CVNombre,
					&item.CVTipo,
					&item.CVTamano,

					&item.Estado,

					&item.RutaPagina,
					&item.URLPagina,
					&item.PaginaReferencia,

					&item.UTMSource,
					&item.UTMMedium,
					&item.UTMCampaign,
					&item.UTMContent,
					&item.UTMTerm,

					&item.CreadoEn,
					&item.ActualizadoEn,
				)

		if errors.Is(
			err,
			pgx.ErrNoRows,
		) {
			c.JSON(
				http.StatusNotFound,
				gin.H{
					"success": false,
					"message": "La postulación no existe.",
				},
			)

			return
		}

		if err != nil {
			c.JSON(
				http.StatusInternalServerError,
				gin.H{
					"success": false,
					"message": "No se pudo cargar la postulación.",
					"error":   err.Error(),
				},
			)

			return
		}

		c.JSON(
			http.StatusOK,
			gin.H{
				"success": true,
				"data":    item,
			},
		)
	}
}

/* =========================================================
   GET /api/postulaciones/:id/cv
========================================================= */

func verCVPostulacion(
	db *pgxpool.Pool,
) gin.HandlerFunc {
	return func(c *gin.Context) {
		if db == nil {
			c.JSON(
				http.StatusInternalServerError,
				gin.H{
					"success": false,
					"message": "La base de datos no está disponible.",
				},
			)

			return
		}

		id :=
			strings.TrimSpace(
				c.Param(
					"id",
				),
			)

		if id == "" {
			c.JSON(
				http.StatusBadRequest,
				gin.H{
					"success": false,
					"message": "ID de postulación inválido.",
				},
			)

			return
		}

		ctx, cancel :=
			context.WithTimeout(
				c.Request.Context(),
				10*time.Second,
			)

		defer cancel()

		var cvNombre string
		var cvTipo string
		var cvData []byte

		err :=
			db.QueryRow(
				ctx,
				`
				SELECT
					cv_nombre,
					cv_tipo,
					cv_data

				FROM
					postulaciones_trabajo

				WHERE
					id = $1

				LIMIT 1
				`,
				id,
			).
				Scan(
					&cvNombre,
					&cvTipo,
					&cvData,
				)

		if errors.Is(
			err,
			pgx.ErrNoRows,
		) {
			c.JSON(
				http.StatusNotFound,
				gin.H{
					"success": false,
					"message": "No se encontró el CV.",
				},
			)

			return
		}

		if err != nil {
			c.JSON(
				http.StatusInternalServerError,
				gin.H{
					"success": false,
					"message": "No se pudo cargar el CV.",
					"error":   err.Error(),
				},
			)

			return
		}

		if len(cvData) == 0 {
			c.JSON(
				http.StatusNotFound,
				gin.H{
					"success": false,
					"message": "La postulación no contiene un CV.",
				},
			)

			return
		}

		if cvTipo == "" {
			cvTipo =
				"application/pdf"
		}

		cvNombre =
			limpiarNombreArchivo(
				cvNombre,
			)

		if cvNombre == "" {
			cvNombre =
				"curriculum.pdf"
		}

		/*
			inline hace que Chrome/Edge
			intenten mostrar el PDF.

			Si luego quieres forzar descarga,
			cambia inline por attachment.
		*/
		c.Header(
			"Content-Disposition",
			fmt.Sprintf(
				`inline; filename="%s"`,
				cvNombre,
			),
		)

		c.Header(
			"Content-Type",
			cvTipo,
		)

		c.Header(
			"Content-Length",
			fmt.Sprintf(
				"%d",
				len(cvData),
			),
		)

		c.Header(
			"Cache-Control",
			"private, no-store, max-age=0",
		)

		c.Header(
			"X-Content-Type-Options",
			"nosniff",
		)

		c.Data(
			http.StatusOK,
			cvTipo,
			cvData,
		)
	}
}

/* =========================================================
   HELPERS
========================================================= */

func limpiarTextoPostulacion(
	value string,
) string {
	return strings.Join(
		strings.Fields(
			strings.TrimSpace(
				value,
			),
		),
		" ",
	)
}

func soloNumerosPostulacion(
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

func telefonoPostulacionValido(
	telefono string,
) bool {
	if len(telefono) != 9 {
		return false
	}

	return telefono[0] == '9'
}

func correoPostulacionValido(
	email string,
) bool {
	email =
		strings.TrimSpace(
			email,
		)

	if email == "" {
		return false
	}

	if len(email) > 150 {
		return false
	}

	if strings.Contains(
		email,
		" ",
	) {
		return false
	}

	at :=
		strings.LastIndex(
			email,
			"@",
		)

	if at <= 0 ||
		at >= len(email)-1 {
		return false
	}

	domain :=
		email[at+1:]

	if !strings.Contains(
		domain,
		".",
	) {
		return false
	}

	return true
}

func limpiarNombreArchivo(
	filename string,
) string {
	filename =
		strings.TrimSpace(
			filename,
		)

	filename =
		strings.ReplaceAll(
			filename,
			"\\",
			"_",
		)

	filename =
		strings.ReplaceAll(
			filename,
			"/",
			"_",
		)

	filename =
		strings.ReplaceAll(
			filename,
			`"`,
			"",
		)

	filename =
		strings.ReplaceAll(
			filename,
			"\r",
			"",
		)

	filename =
		strings.ReplaceAll(
			filename,
			"\n",
			"",
		)

	if len(filename) > 200 {
		filename =
			filename[len(filename)-200:]
	}

	return filename
}

/* =========================================================
   ESTADOS PERMITIDOS
========================================================= */

var estadosPostulacionPermitidos = map[string]bool{
	"recibido":              true,
	"en_revision":           true,
	"entrevista_programada": true,
	"entrevista_atendida":   true,
	"seleccionado":          true,
	"descartado":            true,
}

/* =========================================================
   REQUEST
========================================================= */

type actualizarEstadoPostulacionRequest struct {
	Estado string `json:"estado" binding:"required"`
}

/* =========================================================
   RESPONSE
========================================================= */

type estadoPostulacionResponse struct {
	ID            string    `json:"id"`
	Estado        string    `json:"estado"`
	ActualizadoEn time.Time `json:"actualizado_en"`
}

/* =========================================================
   PATCH /api/postulaciones/:id/estado

   BODY:

   {
     "estado": "entrevista_programada"
   }
========================================================= */

func actualizarEstadoPostulacion(
	db *pgxpool.Pool,
) gin.HandlerFunc {
	return func(c *gin.Context) {
		id :=
			strings.TrimSpace(
				c.Param("id"),
			)

		if id == "" {
			c.JSON(
				http.StatusBadRequest,
				gin.H{
					"success": false,
					"message": "El ID de la postulación es obligatorio.",
				},
			)

			return
		}

		var request actualizarEstadoPostulacionRequest

		if err :=
			c.ShouldBindJSON(
				&request,
			); err != nil {
			c.JSON(
				http.StatusBadRequest,
				gin.H{
					"success": false,
					"message": "Debes enviar un estado válido.",
					"error":   err.Error(),
				},
			)

			return
		}

		estado :=
			strings.ToLower(
				strings.TrimSpace(
					request.Estado,
				),
			)

		/* =========================================
		   ALIASES OPCIONALES
		========================================= */

		switch estado {
		case "revisado":
			estado =
				"en_revision"

		case "revision":
			estado =
				"en_revision"

		case "entrevista":
			estado =
				"entrevista_programada"

		case "programado":
			estado =
				"entrevista_programada"

		case "atendido":
			estado =
				"entrevista_atendida"
		}

		if !estadosPostulacionPermitidos[estado] {
			c.JSON(
				http.StatusBadRequest,
				gin.H{
					"success": false,

					"message": "Estado no permitido.",

					"estados_permitidos": []string{
						"recibido",
						"en_revision",
						"entrevista_programada",
						"entrevista_atendida",
						"seleccionado",
						"descartado",
					},
				},
			)

			return
		}

		var result estadoPostulacionResponse

		ctx, cancel :=
			context.WithTimeout(
				c.Request.Context(),
				10*time.Second,
			)

		defer cancel()

		err :=
			db.QueryRow(
				ctx,
				`
				UPDATE postulaciones_trabajo
				SET
					estado = $1,
					actualizado_en = NOW()
				WHERE id = $2
				RETURNING
					id,
					estado,
					actualizado_en
				`,
				estado,
				id,
			).Scan(
				&result.ID,
				&result.Estado,
				&result.ActualizadoEn,
			)

		if errors.Is(
			err,
			pgx.ErrNoRows,
		) {
			c.JSON(
				http.StatusNotFound,
				gin.H{
					"success": false,
					"message": "La postulación no existe.",
				},
			)

			return
		}

		if err != nil {
			c.JSON(
				http.StatusInternalServerError,
				gin.H{
					"success": false,
					"message": "No se pudo actualizar el estado de la postulación.",
					"error":   err.Error(),
				},
			)

			return
		}

		c.JSON(
			http.StatusOK,
			gin.H{
				"success": true,

				"message": "Estado actualizado correctamente.",

				"data": result,
			},
		)
	}
}