package routes

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

/* =========================================================
   CONFIGURACIÓN
========================================================= */

const maxConvocatoriaImageSize int64 = 5 * 1024 * 1024 // 5 MB

/* =========================================================
   MODELOS
========================================================= */

type Convocatoria struct {
	ID string `json:"id"`

	Titulo      string `json:"titulo"`
	Descripcion string `json:"descripcion"`

	ImagenNombre string `json:"imagen_nombre"`
	ImagenTipo   string `json:"imagen_tipo"`
	ImagenTamano int64  `json:"imagen_tamano"`

	ImagenURL string `json:"imagen_url"`

	Activo bool `json:"activo"`

	CreadoEn      time.Time `json:"creado_en"`
	ActualizadoEn time.Time `json:"actualizado_en"`
}

type CambiarEstadoConvocatoriaRequest struct {
	Activo *bool `json:"activo" binding:"required"`
}

/* =========================================================
   RUTAS
========================================================= */

func RutasConvocatorias(
	api *gin.RouterGroup,
	db *pgxpool.Pool,
) {
	/*
		DASHBOARD

		Devuelve activas e inactivas.
	*/
	api.GET(
		"/convocatorias",
		listarConvocatorias(db),
	)

	/*
		WEB PÚBLICA

		Solo devuelve:
		activo = TRUE
	*/
	api.GET(
		"/convocatorias/publicas",
		listarConvocatoriasPublicas(db),
	)

	/*
		IMAGEN
	*/
	api.GET(
		"/convocatorias/:id/imagen",
		obtenerImagenConvocatoria(db),
	)

	/*
		CREAR
	*/
	api.POST(
		"/convocatorias",
		crearConvocatoria(db),
	)

	/*
		EDITAR

		Permite cambiar:
		- título
		- descripción
		- imagen opcional
	*/
	api.PUT(
		"/convocatorias/:id",
		actualizarConvocatoria(db),
	)

	/*
		CAMBIAR ESTADO

		true  = publicada
		false = oculta
	*/
	api.PATCH(
		"/convocatorias/:id/estado",
		actualizarEstadoConvocatoria(db),
	)
}

/* =========================================================
   CREAR CONVOCATORIA
========================================================= */

func crearConvocatoria(
	db *pgxpool.Pool,
) gin.HandlerFunc {
	return func(c *gin.Context) {
		/* =========================================
		   MULTIPART
		========================================= */

		if err := c.Request.ParseMultipartForm(
			maxConvocatoriaImageSize,
		); err != nil {
			c.JSON(
				http.StatusBadRequest,
				gin.H{
					"success": false,
					"message": "No se pudo procesar el formulario.",
					"error":   err.Error(),
				},
			)

			return
		}

		/* =========================================
		   CAMPOS
		========================================= */

		titulo :=
			strings.TrimSpace(
				c.PostForm("titulo"),
			)

		descripcion :=
			strings.TrimSpace(
				c.PostForm("descripcion"),
			)

		if titulo == "" {
			c.JSON(
				http.StatusBadRequest,
				gin.H{
					"success": false,
					"message": "El título es obligatorio.",
				},
			)

			return
		}

		if descripcion == "" {
			c.JSON(
				http.StatusBadRequest,
				gin.H{
					"success": false,
					"message": "La descripción es obligatoria.",
				},
			)

			return
		}

		if len(titulo) > 200 {
			c.JSON(
				http.StatusBadRequest,
				gin.H{
					"success": false,
					"message": "El título es demasiado largo.",
				},
			)

			return
		}

		/* =========================================
		   IMAGEN
		========================================= */

		file,
			header,
			err :=
			c.Request.FormFile(
				"imagen",
			)

		if err != nil {
			c.JSON(
				http.StatusBadRequest,
				gin.H{
					"success": false,
					"message": "Debes seleccionar una imagen.",
				},
			)

			return
		}

		defer file.Close()

		/* =========================================
		   VALIDAR TAMAÑO
		========================================= */

		if header.Size <= 0 {
			c.JSON(
				http.StatusBadRequest,
				gin.H{
					"success": false,
					"message": "La imagen está vacía.",
				},
			)

			return
		}

		if header.Size > maxConvocatoriaImageSize {
			c.JSON(
				http.StatusRequestEntityTooLarge,
				gin.H{
					"success": false,
					"message": "La imagen no debe superar los 5 MB.",
				},
			)

			return
		}

		/* =========================================
		   LEER IMAGEN
		========================================= */

		imageData,
			err :=
			io.ReadAll(
				io.LimitReader(
					file,
					maxConvocatoriaImageSize+1,
				),
			)

		if err != nil {
			c.JSON(
				http.StatusInternalServerError,
				gin.H{
					"success": false,
					"message": "No se pudo leer la imagen.",
				},
			)

			return
		}

		if int64(len(imageData)) >
			maxConvocatoriaImageSize {
			c.JSON(
				http.StatusRequestEntityTooLarge,
				gin.H{
					"success": false,
					"message": "La imagen no debe superar los 5 MB.",
				},
			)

			return
		}

		/* =========================================
		   DETECTAR MIME REAL
		========================================= */

		contentType :=
			http.DetectContentType(
				imageData,
			)

		if !imagenPermitida(
			contentType,
		) {
			c.JSON(
				http.StatusBadRequest,
				gin.H{
					"success": false,
					"message": "La imagen debe ser JPG, PNG o WEBP.",
				},
			)

			return
		}

		/* =========================================
		   INSERT
		========================================= */

		var convocatoria Convocatoria

		err =
			db.QueryRow(
				context.Background(),
				`
				INSERT INTO convocatorias_trabajo (
					titulo,
					descripcion,

					imagen_nombre,
					imagen_tipo,
					imagen_tamano,
					imagen_data,

					activo,

					creado_en,
					actualizado_en
				)
				VALUES (
					$1,
					$2,

					$3,
					$4,
					$5,
					$6,

					TRUE,

					NOW(),
					NOW()
				)
				RETURNING
					id,
					titulo,
					descripcion,

					imagen_nombre,
					imagen_tipo,
					imagen_tamano,

					activo,

					creado_en,
					actualizado_en
				`,
				titulo,
				descripcion,

				header.Filename,
				contentType,
				len(imageData),
				imageData,
			).Scan(
				&convocatoria.ID,
				&convocatoria.Titulo,
				&convocatoria.Descripcion,

				&convocatoria.ImagenNombre,
				&convocatoria.ImagenTipo,
				&convocatoria.ImagenTamano,

				&convocatoria.Activo,

				&convocatoria.CreadoEn,
				&convocatoria.ActualizadoEn,
			)

		if err != nil {
			fmt.Println(
				"Error creando convocatoria:",
				err,
			)

			c.JSON(
				http.StatusInternalServerError,
				gin.H{
					"success": false,
					"message": "No se pudo registrar la convocatoria.",
					"error":   err.Error(),
				},
			)

			return
		}

		convocatoria.ImagenURL =
			"/api/convocatorias/" +
				convocatoria.ID +
				"/imagen"

		c.JSON(
			http.StatusCreated,
			gin.H{
				"success": true,

				"message": "Convocatoria publicada correctamente.",

				"data": convocatoria,
			},
		)
	}
}

/* =========================================================
   LISTAR TODAS

   PARA DASHBOARD.

   AQUÍ NO PONEMOS:
   WHERE activo = TRUE

   porque el administrador necesita ver
   activas e inactivas.
========================================================= */

func listarConvocatorias(
	db *pgxpool.Pool,
) gin.HandlerFunc {
	return func(c *gin.Context) {
		rows,
			err :=
			db.Query(
				context.Background(),
				`
				SELECT
					id,
					titulo,
					descripcion,

					imagen_nombre,
					imagen_tipo,
					imagen_tamano,

					activo,

					creado_en,
					actualizado_en
				FROM convocatorias_trabajo
				ORDER BY creado_en DESC
				`,
			)

		if err != nil {
			c.JSON(
				http.StatusInternalServerError,
				gin.H{
					"success": false,
					"message": "No se pudieron cargar las convocatorias.",
					"error":   err.Error(),
				},
			)

			return
		}

		defer rows.Close()

		convocatorias :=
			make(
				[]Convocatoria,
				0,
			)

		for rows.Next() {
			var item Convocatoria

			err :=
				rows.Scan(
					&item.ID,
					&item.Titulo,
					&item.Descripcion,

					&item.ImagenNombre,
					&item.ImagenTipo,
					&item.ImagenTamano,

					&item.Activo,

					&item.CreadoEn,
					&item.ActualizadoEn,
				)

			if err != nil {
				c.JSON(
					http.StatusInternalServerError,
					gin.H{
						"success": false,
						"message": "No se pudo leer una convocatoria.",
						"error":   err.Error(),
					},
				)

				return
			}

			item.ImagenURL =
				"/api/convocatorias/" +
					item.ID +
					"/imagen"

			convocatorias =
				append(
					convocatorias,
					item,
				)
		}

		if err := rows.Err(); err != nil {
			c.JSON(
				http.StatusInternalServerError,
				gin.H{
					"success": false,
					"message": "Error recorriendo las convocatorias.",
					"error":   err.Error(),
				},
			)

			return
		}

		c.JSON(
			http.StatusOK,
			gin.H{
				"success": true,
				"total":   len(convocatorias),
				"data":    convocatorias,
			},
		)
	}
}

/* =========================================================
   LISTAR CONVOCATORIAS PÚBLICAS

   ESTA ES LA QUE USA:
   /trabaja-con-nosotros

   IMPORTANTE:
   únicamente activo = TRUE
========================================================= */

func listarConvocatoriasPublicas(
	db *pgxpool.Pool,
) gin.HandlerFunc {
	return func(c *gin.Context) {
		rows,
			err :=
			db.Query(
				context.Background(),
				`
				SELECT
					id,
					titulo,
					descripcion,

					imagen_nombre,
					imagen_tipo,
					imagen_tamano,

					activo,

					creado_en,
					actualizado_en
				FROM convocatorias_trabajo
				WHERE activo = TRUE
				ORDER BY creado_en DESC
				`,
			)

		if err != nil {
			c.JSON(
				http.StatusInternalServerError,
				gin.H{
					"success": false,
					"message": "No se pudieron cargar las convocatorias públicas.",
					"error":   err.Error(),
				},
			)

			return
		}

		defer rows.Close()

		convocatorias :=
			make(
				[]Convocatoria,
				0,
			)

		for rows.Next() {
			var item Convocatoria

			err :=
				rows.Scan(
					&item.ID,
					&item.Titulo,
					&item.Descripcion,

					&item.ImagenNombre,
					&item.ImagenTipo,
					&item.ImagenTamano,

					&item.Activo,

					&item.CreadoEn,
					&item.ActualizadoEn,
				)

			if err != nil {
				c.JSON(
					http.StatusInternalServerError,
					gin.H{
						"success": false,
						"message": "No se pudo leer una convocatoria.",
						"error":   err.Error(),
					},
				)

				return
			}

			item.ImagenURL =
				"/api/convocatorias/" +
					item.ID +
					"/imagen"

			convocatorias =
				append(
					convocatorias,
					item,
				)
		}

		if err := rows.Err(); err != nil {
			c.JSON(
				http.StatusInternalServerError,
				gin.H{
					"success": false,
					"message": "Error recorriendo las convocatorias.",
					"error":   err.Error(),
				},
			)

			return
		}

		c.JSON(
			http.StatusOK,
			gin.H{
				"success": true,
				"total":   len(convocatorias),
				"data":    convocatorias,
			},
		)
	}
}

/* =========================================================
   OBTENER IMAGEN
========================================================= */

func obtenerImagenConvocatoria(
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
					"message": "ID de convocatoria requerido.",
				},
			)

			return
		}

		var (
			imageData []byte
			imageType string
		)

		err :=
			db.QueryRow(
				context.Background(),
				`
				SELECT
					imagen_data,
					imagen_tipo
				FROM convocatorias_trabajo
				WHERE id = $1
				`,
				id,
			).Scan(
				&imageData,
				&imageType,
			)

		if err == pgx.ErrNoRows {
			c.JSON(
				http.StatusNotFound,
				gin.H{
					"success": false,
					"message": "Convocatoria no encontrada.",
				},
			)

			return
		}

		if err != nil {
			c.JSON(
				http.StatusInternalServerError,
				gin.H{
					"success": false,
					"message": "No se pudo obtener la imagen.",
					"error":   err.Error(),
				},
			)

			return
		}

		/* =========================================
		   CACHE
		========================================= */

		c.Header(
			"Cache-Control",
			"public, max-age=3600",
		)

		c.Data(
			http.StatusOK,
			imageType,
			imageData,
		)
	}
}

/* =========================================================
   CAMBIAR ESTADO

   DASHBOARD:

   PATCH
   /api/convocatorias/:id/estado

   {
      "activo": false
   }
========================================================= */

func actualizarEstadoConvocatoria(
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
					"message": "ID de convocatoria requerido.",
				},
			)

			return
		}

		var request CambiarEstadoConvocatoriaRequest

		if err :=
			c.ShouldBindJSON(
				&request,
			); err != nil {
			c.JSON(
				http.StatusBadRequest,
				gin.H{
					"success": false,
					"message": "Debes enviar el estado activo.",
					"error":   err.Error(),
				},
			)

			return
		}

		if request.Activo == nil {
			c.JSON(
				http.StatusBadRequest,
				gin.H{
					"success": false,
					"message": "El campo activo es obligatorio.",
				},
			)

			return
		}

		var (
			activo        bool
			actualizadoEn time.Time
		)

		err :=
			db.QueryRow(
				context.Background(),
				`
				UPDATE convocatorias_trabajo
				SET
					activo = $1,
					actualizado_en = NOW()
				WHERE id = $2
				RETURNING
					activo,
					actualizado_en
				`,
				*request.Activo,
				id,
			).Scan(
				&activo,
				&actualizadoEn,
			)

		if err == pgx.ErrNoRows {
			c.JSON(
				http.StatusNotFound,
				gin.H{
					"success": false,
					"message": "Convocatoria no encontrada.",
				},
			)

			return
		}

		if err != nil {
			fmt.Println(
				"Error actualizando estado convocatoria:",
				err,
			)

			c.JSON(
				http.StatusInternalServerError,
				gin.H{
					"success": false,
					"message": "No se pudo actualizar el estado.",
					"error":   err.Error(),
				},
			)

			return
		}

		message :=
			"Convocatoria desactivada correctamente."

		if activo {
			message =
				"Convocatoria activada correctamente."
		}

		c.JSON(
			http.StatusOK,
			gin.H{
				"success": true,
				"message": message,

				"data": gin.H{
					"id": id,

					"activo": activo,

					"actualizado_en": actualizadoEn,
				},
			},
		)
	}
}

/* =========================================================
   EDITAR CONVOCATORIA

   PUT
   /api/convocatorias/:id

   multipart/form-data

   titulo
   descripcion
   imagen -> opcional
========================================================= */

func actualizarConvocatoria(
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
					"message": "ID de convocatoria requerido.",
				},
			)

			return
		}

		/* =========================================
		   VERIFICAR EXISTENCIA
		========================================= */

		var existe bool

		err :=
			db.QueryRow(
				context.Background(),
				`
				SELECT EXISTS(
					SELECT 1
					FROM convocatorias_trabajo
					WHERE id = $1
				)
				`,
				id,
			).Scan(
				&existe,
			)

		if err != nil {
			c.JSON(
				http.StatusInternalServerError,
				gin.H{
					"success": false,
					"message": "No se pudo verificar la convocatoria.",
					"error":   err.Error(),
				},
			)

			return
		}

		if !existe {
			c.JSON(
				http.StatusNotFound,
				gin.H{
					"success": false,
					"message": "Convocatoria no encontrada.",
				},
			)

			return
		}

		/* =========================================
		   DATOS
		========================================= */

		titulo :=
			strings.TrimSpace(
				c.PostForm("titulo"),
			)

		descripcion :=
			strings.TrimSpace(
				c.PostForm("descripcion"),
			)

		if titulo == "" {
			c.JSON(
				http.StatusBadRequest,
				gin.H{
					"success": false,
					"message": "El título es obligatorio.",
				},
			)

			return
		}

		if descripcion == "" {
			c.JSON(
				http.StatusBadRequest,
				gin.H{
					"success": false,
					"message": "La descripción es obligatoria.",
				},
			)

			return
		}

		/* =========================================
		   REVISAR SI VIENE NUEVA IMAGEN
		========================================= */

		file,
			header,
			imageErr :=
			c.Request.FormFile(
				"imagen",
			)

		/* =========================================
		   SIN NUEVA IMAGEN
		========================================= */

		if imageErr != nil {
			var item Convocatoria

			err :=
				db.QueryRow(
					context.Background(),
					`
					UPDATE convocatorias_trabajo
					SET
						titulo = $1,
						descripcion = $2,
						actualizado_en = NOW()
					WHERE id = $3
					RETURNING
						id,
						titulo,
						descripcion,

						imagen_nombre,
						imagen_tipo,
						imagen_tamano,

						activo,

						creado_en,
						actualizado_en
					`,
					titulo,
					descripcion,
					id,
				).Scan(
					&item.ID,
					&item.Titulo,
					&item.Descripcion,

					&item.ImagenNombre,
					&item.ImagenTipo,
					&item.ImagenTamano,

					&item.Activo,

					&item.CreadoEn,
					&item.ActualizadoEn,
				)

			if err != nil {
				c.JSON(
					http.StatusInternalServerError,
					gin.H{
						"success": false,
						"message": "No se pudo actualizar la convocatoria.",
						"error":   err.Error(),
					},
				)

				return
			}

			item.ImagenURL =
				"/api/convocatorias/" +
					item.ID +
					"/imagen"

			c.JSON(
				http.StatusOK,
				gin.H{
					"success": true,
					"message": "Convocatoria actualizada correctamente.",
					"data":    item,
				},
			)

			return
		}

		defer file.Close()

		/* =========================================
		   VALIDAR IMAGEN NUEVA
		========================================= */

		if header.Size >
			maxConvocatoriaImageSize {
			c.JSON(
				http.StatusRequestEntityTooLarge,
				gin.H{
					"success": false,
					"message": "La imagen no debe superar los 5 MB.",
				},
			)

			return
		}

		imageData,
			err :=
			io.ReadAll(
				io.LimitReader(
					file,
					maxConvocatoriaImageSize+1,
				),
			)

		if err != nil {
			c.JSON(
				http.StatusInternalServerError,
				gin.H{
					"success": false,
					"message": "No se pudo leer la nueva imagen.",
				},
			)

			return
		}

		if int64(len(imageData)) >
			maxConvocatoriaImageSize {
			c.JSON(
				http.StatusRequestEntityTooLarge,
				gin.H{
					"success": false,
					"message": "La imagen no debe superar los 5 MB.",
				},
			)

			return
		}

		contentType :=
			http.DetectContentType(
				imageData,
			)

		if !imagenPermitida(
			contentType,
		) {
			c.JSON(
				http.StatusBadRequest,
				gin.H{
					"success": false,
					"message": "La imagen debe ser JPG, PNG o WEBP.",
				},
			)

			return
		}

		/* =========================================
		   UPDATE CON IMAGEN
		========================================= */

		var item Convocatoria

		err =
			db.QueryRow(
				context.Background(),
				`
				UPDATE convocatorias_trabajo
				SET
					titulo = $1,
					descripcion = $2,

					imagen_nombre = $3,
					imagen_tipo = $4,
					imagen_tamano = $5,
					imagen_data = $6,

					actualizado_en = NOW()
				WHERE id = $7
				RETURNING
					id,
					titulo,
					descripcion,

					imagen_nombre,
					imagen_tipo,
					imagen_tamano,

					activo,

					creado_en,
					actualizado_en
				`,
				titulo,
				descripcion,

				header.Filename,
				contentType,
				len(imageData),
				imageData,

				id,
			).Scan(
				&item.ID,
				&item.Titulo,
				&item.Descripcion,

				&item.ImagenNombre,
				&item.ImagenTipo,
				&item.ImagenTamano,

				&item.Activo,

				&item.CreadoEn,
				&item.ActualizadoEn,
			)

		if err != nil {
			c.JSON(
				http.StatusInternalServerError,
				gin.H{
					"success": false,
					"message": "No se pudo actualizar la convocatoria.",
					"error":   err.Error(),
				},
			)

			return
		}

		item.ImagenURL =
			"/api/convocatorias/" +
				item.ID +
				"/imagen"

		c.JSON(
			http.StatusOK,
			gin.H{
				"success": true,

				"message": "Convocatoria actualizada correctamente.",

				"data": item,
			},
		)
	}
}

/* =========================================================
   HELPERS
========================================================= */

func imagenPermitida(
	contentType string,
) bool {
	switch contentType {
	case "image/jpeg":
		return true

	case "image/png":
		return true

	case "image/webp":
		return true

	default:
		return false
	}
}

/* =========================================================
   HELPER OPCIONAL

   Útil si posteriormente quieres recibir
   valores boolean desde multipart.
========================================================= */

func parseBool(
	value string,
	defaultValue bool,
) bool {
	value =
		strings.TrimSpace(
			value,
		)

	if value == "" {
		return defaultValue
	}

	result,
		err :=
		strconv.ParseBool(
			value,
		)

	if err != nil {
		return defaultValue
	}

	return result
}