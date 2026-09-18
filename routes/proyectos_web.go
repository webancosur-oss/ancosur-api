package routes

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const maxProyectoWebImageSize int64 = 5 * 1024 * 1024

type ProyectoWeb struct {
	ID string `json:"id"`

	Codigo string `json:"codigo"`
	Titulo string `json:"titulo"`
	Slug   string `json:"slug"`

	Tipo      string `json:"tipo"`
	Ciudad    string `json:"ciudad"`
	Direccion string `json:"direccion"`
	Etapa     string `json:"etapa"`

	ImagenNombre string `json:"imagen_nombre"`
	ImagenTipo   string `json:"imagen_tipo"`
	ImagenTamano int64  `json:"imagen_tamano"`
	ImagenURL    string `json:"imagen_url"`

	Dormitorios *string `json:"dormitorios"`

	MetrajeDesde *float64 `json:"metraje_desde"`
	MetrajeHasta *float64 `json:"metraje_hasta"`

	Estado string `json:"estado"`

	PrecioDesde *float64 `json:"precio_desde"`

	Activo bool `json:"activo"`
	Orden  int  `json:"orden"`

	CreadoEn      time.Time `json:"creado_en"`
	ActualizadoEn time.Time `json:"actualizado_en"`
}

/* =========================================================
   RUTAS
========================================================= */

func RutasProyectosWeb(api *gin.RouterGroup, db *pgxpool.Pool) {

	proyectos := api.Group("/web/proyectos")

	proyectos.GET("", listarProyectosWeb(db))

	proyectos.POST("", crearProyectoWeb(db))

	proyectos.GET("/:id", obtenerProyectoWeb(db))

	proyectos.GET("/:id/imagen", obtenerImagenProyectoWeb(db))

	proyectos.PUT("/:id", actualizarProyectoWeb(db))

	proyectos.DELETE("/:id", eliminarProyectoWeb(db))
}

/* =========================================================
   LISTAR PROYECTOS
========================================================= */

func listarProyectosWeb(db *pgxpool.Pool) gin.HandlerFunc {

	return func(c *gin.Context) {

		page := parseIntProyecto(
			c.DefaultQuery("page", "1"),
			1,
		)

		limit := parseIntProyecto(
			c.DefaultQuery("limit", "10"),
			10,
		)

		if page < 1 {
			page = 1
		}

		if limit < 1 {
			limit = 10
		}

		if limit > 100 {
			limit = 100
		}

		offset := (page - 1) * limit

		buscar := strings.TrimSpace(
			c.Query("buscar"),
		)

		estado := strings.TrimSpace(
			c.Query("estado"),
		)

		tipo := strings.TrimSpace(
			c.Query("tipo"),
		)

		ciudad := strings.TrimSpace(
			c.Query("ciudad"),
		)

		etapa := strings.TrimSpace(
			c.Query("etapa"),
		)

		activo := strings.TrimSpace(
			c.Query("activo"),
		)

		args := make([]any, 0)

		where := []string{
			"1 = 1",
		}

		/* =====================================================
		   BUSCAR
		===================================================== */

		if buscar != "" {

			args = append(
				args,
				"%"+buscar+"%",
			)

			param := strconv.Itoa(
				len(args),
			)

			where = append(
				where,
				`(
					codigo ILIKE $`+param+`
					OR titulo ILIKE $`+param+`
					OR ciudad ILIKE $`+param+`
					OR direccion ILIKE $`+param+`
					OR etapa ILIKE $`+param+`
					OR slug ILIKE $`+param+`
				)`,
			)
		}

		/* =====================================================
		   ESTADO
		===================================================== */

		if estado != "" {

			args = append(
				args,
				estado,
			)

			where = append(
				where,
				fmt.Sprintf(
					"estado = $%d",
					len(args),
				),
			)
		}

		/* =====================================================
		   TIPO
		===================================================== */

		if tipo != "" {

			args = append(
				args,
				tipo,
			)

			where = append(
				where,
				fmt.Sprintf(
					"tipo = $%d",
					len(args),
				),
			)
		}

		/* =====================================================
		   CIUDAD
		===================================================== */

		if ciudad != "" {

			args = append(
				args,
				ciudad,
			)

			where = append(
				where,
				fmt.Sprintf(
					"ciudad = $%d",
					len(args),
				),
			)
		}

		/* =====================================================
		   ETAPA
		===================================================== */

		if etapa != "" {

			args = append(
				args,
				etapa,
			)

			where = append(
				where,
				fmt.Sprintf(
					"etapa = $%d",
					len(args),
				),
			)
		}

		/* =====================================================
		   ACTIVO
		===================================================== */

		if activo != "" {

			valor := strings.ToLower(
				activo,
			)

			if valor == "true" ||
				valor == "false" {

				args = append(
					args,
					valor == "true",
				)

				where = append(
					where,
					fmt.Sprintf(
						"activo = $%d",
						len(args),
					),
				)
			}
		}

		whereSQL := strings.Join(
			where,
			" AND ",
		)

		/* =====================================================
		   TOTAL
		===================================================== */

		var total int

		err := db.QueryRow(
			c,
			`
			SELECT COUNT(*)
			FROM proyectos_web
			WHERE `+whereSQL,
			args...,
		).Scan(
			&total,
		)

		if err != nil {

			c.JSON(
				http.StatusInternalServerError,
				gin.H{
					"success": false,
					"message": "Error al contar proyectos",
					"error":   err.Error(),
				},
			)

			return
		}

		/* =====================================================
		   PAGINACIÓN
		===================================================== */

		queryArgs := append(
			[]any{},
			args...,
		)

		queryArgs = append(
			queryArgs,
			limit,
		)

		limitParam := len(
			queryArgs,
		)

		queryArgs = append(
			queryArgs,
			offset,
		)

		offsetParam := len(
			queryArgs,
		)

		/* =====================================================
		   QUERY
		===================================================== */

		query := `
			SELECT
				id,
				codigo,
				titulo,
				slug,
				tipo,
				ciudad,
				direccion,
				etapa,

				COALESCE(imagen_nombre, ''),
				COALESCE(imagen_tipo, ''),
				COALESCE(imagen_tamano, 0),

				dormitorios,
				metraje_desde,
				metraje_hasta,

				estado,
				precio_desde,

				activo,
				orden,

				creado_en,
				actualizado_en

			FROM proyectos_web

			WHERE ` + whereSQL + `

			ORDER BY
				orden ASC,
				creado_en DESC

			LIMIT $` + strconv.Itoa(
			limitParam,
		) + `

			OFFSET $` + strconv.Itoa(
			offsetParam,
		)

		rows, err := db.Query(
			c,
			query,
			queryArgs...,
		)

		if err != nil {

			c.JSON(
				http.StatusInternalServerError,
				gin.H{
					"success": false,
					"message": "Error al obtener proyectos",
					"error":   err.Error(),
				},
			)

			return
		}

		defer rows.Close()

		proyectos := make(
			[]ProyectoWeb,
			0,
		)

		for rows.Next() {

			var proyecto ProyectoWeb

			err := rows.Scan(

				&proyecto.ID,

				&proyecto.Codigo,

				&proyecto.Titulo,

				&proyecto.Slug,

				&proyecto.Tipo,

				&proyecto.Ciudad,

				&proyecto.Direccion,

				&proyecto.Etapa,

				&proyecto.ImagenNombre,

				&proyecto.ImagenTipo,

				&proyecto.ImagenTamano,

				&proyecto.Dormitorios,

				&proyecto.MetrajeDesde,

				&proyecto.MetrajeHasta,

				&proyecto.Estado,

				&proyecto.PrecioDesde,

				&proyecto.Activo,

				&proyecto.Orden,

				&proyecto.CreadoEn,

				&proyecto.ActualizadoEn,
			)

			if err != nil {

				c.JSON(
					http.StatusInternalServerError,
					gin.H{
						"success": false,
						"message": "Error al leer proyecto",
						"error":   err.Error(),
					},
				)

				return
			}

			proyecto.ImagenURL =
				"/api/web/proyectos/" +
					proyecto.ID +
					"/imagen"

			proyectos = append(
				proyectos,
				proyecto,
			)
		}

		if err := rows.Err(); err != nil {

			c.JSON(
				http.StatusInternalServerError,
				gin.H{
					"success": false,
					"message": "Error al recorrer proyectos",
					"error":   err.Error(),
				},
			)

			return
		}

		totalPages := 0

		if total > 0 {

			totalPages =
				(total + limit - 1) /
					limit
		}

		c.JSON(
			http.StatusOK,
			gin.H{
				"success": true,
				"data":    proyectos,

				"pagination": gin.H{
					"page":        page,
					"limit":       limit,
					"total":       total,
					"total_pages": totalPages,
				},
			},
		)
	}
}

/* =========================================================
   CREAR PROYECTO
========================================================= */

func crearProyectoWeb(db *pgxpool.Pool) gin.HandlerFunc {

	return func(c *gin.Context) {

		codigo := strings.TrimSpace(
			c.PostForm("codigo"),
		)

		titulo := strings.TrimSpace(
			c.PostForm("titulo"),
		)

		slug := strings.TrimSpace(
			c.PostForm("slug"),
		)

		tipo := strings.TrimSpace(
			c.PostForm("tipo"),
		)

		ciudad := strings.TrimSpace(
			c.PostForm("ciudad"),
		)

		direccion := strings.TrimSpace(
			c.PostForm("direccion"),
		)

		etapa := strings.TrimSpace(
			c.PostForm("etapa"),
		)

		if codigo == "" ||
			titulo == "" ||
			slug == "" ||
			tipo == "" ||
			ciudad == "" ||
			direccion == "" ||
			etapa == "" {

			c.JSON(
				http.StatusBadRequest,
				gin.H{
					"success": false,
					"message": "Completa todos los campos obligatorios",
				},
			)

			return
		}

		/* =====================================================
		   IMAGEN
		===================================================== */

		imagen, err := c.FormFile(
			"imagen",
		)

		if err != nil {

			c.JSON(
				http.StatusBadRequest,
				gin.H{
					"success": false,
					"message": "La imagen es obligatoria",
				},
			)

			return
		}

		if imagen.Size <= 0 {

			c.JSON(
				http.StatusBadRequest,
				gin.H{
					"success": false,
					"message": "La imagen está vacía",
				},
			)

			return
		}

		if imagen.Size >
			maxProyectoWebImageSize {

			c.JSON(
				http.StatusBadRequest,
				gin.H{
					"success": false,
					"message": "La imagen no puede superar los 5 MB",
				},
			)

			return
		}

		file, err := imagen.Open()

		if err != nil {

			c.JSON(
				http.StatusInternalServerError,
				gin.H{
					"success": false,
					"message": "No se pudo abrir la imagen",
				},
			)

			return
		}

		defer file.Close()

		imageData, err := io.ReadAll(
			io.LimitReader(
				file,
				maxProyectoWebImageSize+1,
			),
		)

		if err != nil {

			c.JSON(
				http.StatusInternalServerError,
				gin.H{
					"success": false,
					"message": "No se pudo leer la imagen",
				},
			)

			return
		}

		if int64(len(imageData)) >
			maxProyectoWebImageSize {

			c.JSON(
				http.StatusBadRequest,
				gin.H{
					"success": false,
					"message": "La imagen no puede superar los 5 MB",
				},
			)

			return
		}

		contentType :=
			http.DetectContentType(
				imageData,
			)

		if !imagenProyectoPermitida(
			contentType,
		) {

			c.JSON(
				http.StatusBadRequest,
				gin.H{
					"success": false,
					"message": "Solo se permiten imágenes JPG, PNG o WEBP",
				},
			)

			return
		}

		/* =====================================================
		   CAMPOS OPCIONALES
		===================================================== */

		dormitorios :=
			nullableStringProyecto(
				c.PostForm("dormitorios"),
			)

		metrajeDesde :=
			nullableFloatProyecto(
				c.PostForm("metraje_desde"),
			)

		metrajeHasta :=
			nullableFloatProyecto(
				c.PostForm("metraje_hasta"),
			)

		precioDesde :=
			nullableFloatProyecto(
				c.PostForm("precio_desde"),
			)

		estado := strings.TrimSpace(
			c.DefaultPostForm(
				"estado",
				"disponible",
			),
		)

		activo := parseBoolProyecto(
			c.DefaultPostForm(
				"activo",
				"true",
			),
			true,
		)

		orden := parseIntProyecto(
			c.DefaultPostForm(
				"orden",
				"0",
			),
			0,
		)

		id := uuid.New()

		/* =====================================================
		   INSERTAR
		===================================================== */

		_, err = db.Exec(
			c,
			`
			INSERT INTO proyectos_web (

				id,

				codigo,
				titulo,
				slug,

				tipo,
				ciudad,
				direccion,
				etapa,

				imagen_nombre,
				imagen_tipo,
				imagen_tamano,
				imagen_data,

				dormitorios,

				metraje_desde,
				metraje_hasta,

				estado,

				precio_desde,

				activo,
				orden

			)
			VALUES (

				$1,

				$2,
				$3,
				$4,

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

				$17,

				$18,
				$19
			)
			`,
			id,

			codigo,
			titulo,
			slug,

			tipo,
			ciudad,
			direccion,
			etapa,

			imagen.Filename,
			contentType,
			int64(len(imageData)),
			imageData,

			dormitorios,

			metrajeDesde,
			metrajeHasta,

			estado,

			precioDesde,

			activo,
			orden,
		)

		if err != nil {

			c.JSON(
				http.StatusInternalServerError,
				gin.H{
					"success": false,
					"message": "No se pudo crear el proyecto",
					"error":   err.Error(),
				},
			)

			return
		}

		c.JSON(
			http.StatusCreated,
			gin.H{
				"success": true,
				"message": "Proyecto creado correctamente",

				"data": gin.H{
					"id": id.String(),

					"imagen_url":
						"/api/web/proyectos/" +
							id.String() +
							"/imagen",
				},
			},
		)
	}
}

/* =========================================================
   OBTENER PROYECTO
========================================================= */

func obtenerProyectoWeb(db *pgxpool.Pool) gin.HandlerFunc {

	return func(c *gin.Context) {

		id := c.Param("id")

		var proyecto ProyectoWeb

		err := db.QueryRow(
			c,
			`
			SELECT

				id,

				codigo,
				titulo,
				slug,

				tipo,
				ciudad,
				direccion,
				etapa,

				COALESCE(imagen_nombre, ''),
				COALESCE(imagen_tipo, ''),
				COALESCE(imagen_tamano, 0),

				dormitorios,

				metraje_desde,
				metraje_hasta,

				estado,

				precio_desde,

				activo,
				orden,

				creado_en,
				actualizado_en

			FROM proyectos_web

			WHERE id = $1
			`,
			id,
		).Scan(

			&proyecto.ID,

			&proyecto.Codigo,
			&proyecto.Titulo,
			&proyecto.Slug,

			&proyecto.Tipo,
			&proyecto.Ciudad,
			&proyecto.Direccion,
			&proyecto.Etapa,

			&proyecto.ImagenNombre,
			&proyecto.ImagenTipo,
			&proyecto.ImagenTamano,

			&proyecto.Dormitorios,

			&proyecto.MetrajeDesde,
			&proyecto.MetrajeHasta,

			&proyecto.Estado,

			&proyecto.PrecioDesde,

			&proyecto.Activo,
			&proyecto.Orden,

			&proyecto.CreadoEn,
			&proyecto.ActualizadoEn,
		)

		if err != nil {

			if err == pgx.ErrNoRows {

				c.JSON(
					http.StatusNotFound,
					gin.H{
						"success": false,
						"message": "Proyecto no encontrado",
					},
				)

				return
			}

			c.JSON(
				http.StatusInternalServerError,
				gin.H{
					"success": false,
					"message": "Error al obtener proyecto",
					"error":   err.Error(),
				},
			)

			return
		}

		proyecto.ImagenURL =
			"/api/web/proyectos/" +
				proyecto.ID +
				"/imagen"

		c.JSON(
			http.StatusOK,
			gin.H{
				"success": true,
				"data":    proyecto,
			},
		)
	}
}

/* =========================================================
   OBTENER IMAGEN
========================================================= */

func obtenerImagenProyectoWeb(db *pgxpool.Pool) gin.HandlerFunc {

	return func(c *gin.Context) {

		id := c.Param("id")

		var imageData []byte
		var imageType string

		err := db.QueryRow(
			c,
			`
			SELECT

				imagen_data,
				COALESCE(imagen_tipo, '')

			FROM proyectos_web

			WHERE id = $1
			`,
			id,
		).Scan(
			&imageData,
			&imageType,
		)

		if err != nil {

			if err == pgx.ErrNoRows {

				c.JSON(
					http.StatusNotFound,
					gin.H{
						"success": false,
						"message": "Proyecto no encontrado",
					},
				)

				return
			}

			c.JSON(
				http.StatusInternalServerError,
				gin.H{
					"success": false,
					"message": "Error al obtener imagen",
					"error":   err.Error(),
				},
			)

			return
		}

		if len(imageData) == 0 {

			c.JSON(
				http.StatusNotFound,
				gin.H{
					"success": false,
					"message": "El proyecto no tiene imagen",
				},
			)

			return
		}

		if imageType == "" {

			imageType =
				http.DetectContentType(
					imageData,
				)
		}

		c.Header(
			"Cache-Control",
			"public, max-age=86400",
		)

		c.Header(
			"Content-Disposition",
			"inline",
		)

		c.Data(
			http.StatusOK,
			imageType,
			imageData,
		)
	}
}

/* =========================================================
   ACTUALIZAR PROYECTO
========================================================= */

func actualizarProyectoWeb(db *pgxpool.Pool) gin.HandlerFunc {

	return func(c *gin.Context) {

		id := c.Param("id")

		if _, err := uuid.Parse(id); err != nil {

			c.JSON(
				http.StatusBadRequest,
				gin.H{
					"success": false,
					"message": "ID de proyecto inválido",
				},
			)

			return
		}

		codigo := strings.TrimSpace(
			c.PostForm("codigo"),
		)

		titulo := strings.TrimSpace(
			c.PostForm("titulo"),
		)

		slug := strings.TrimSpace(
			c.PostForm("slug"),
		)

		tipo := strings.TrimSpace(
			c.PostForm("tipo"),
		)

		ciudad := strings.TrimSpace(
			c.PostForm("ciudad"),
		)

		direccion := strings.TrimSpace(
			c.PostForm("direccion"),
		)

		etapa := strings.TrimSpace(
			c.PostForm("etapa"),
		)

		if codigo == "" ||
			titulo == "" ||
			slug == "" ||
			tipo == "" ||
			ciudad == "" ||
			direccion == "" ||
			etapa == "" {

			c.JSON(
				http.StatusBadRequest,
				gin.H{
					"success": false,
					"message": "Completa todos los campos obligatorios",
				},
			)

			return
		}

		dormitorios :=
			nullableStringProyecto(
				c.PostForm("dormitorios"),
			)

		metrajeDesde :=
			nullableFloatProyecto(
				c.PostForm("metraje_desde"),
			)

		metrajeHasta :=
			nullableFloatProyecto(
				c.PostForm("metraje_hasta"),
			)

		precioDesde :=
			nullableFloatProyecto(
				c.PostForm("precio_desde"),
			)

		estado := strings.TrimSpace(
			c.DefaultPostForm(
				"estado",
				"disponible",
			),
		)

		activo := parseBoolProyecto(
			c.DefaultPostForm(
				"activo",
				"true",
			),
			true,
		)

		orden := parseIntProyecto(
			c.DefaultPostForm(
				"orden",
				"0",
			),
			0,
		)

		/* =====================================================
		   COMPROBAR SI VIENE NUEVA IMAGEN
		===================================================== */

		imagen,
		imagenErr := c.FormFile(
			"imagen",
		)

		if imagenErr == nil {

			/* =================================================
			   NUEVA IMAGEN
			================================================= */

			if imagen.Size <= 0 {

				c.JSON(
					http.StatusBadRequest,
					gin.H{
						"success": false,
						"message": "La imagen está vacía",
					},
				)

				return
			}

			if imagen.Size >
				maxProyectoWebImageSize {

				c.JSON(
					http.StatusBadRequest,
					gin.H{
						"success": false,
						"message": "La imagen no puede superar los 5 MB",
					},
				)

				return
			}

			file, err := imagen.Open()

			if err != nil {

				c.JSON(
					http.StatusInternalServerError,
					gin.H{
						"success": false,
						"message": "No se pudo abrir la imagen",
					},
				)

				return
			}

			defer file.Close()

			imageData, err := io.ReadAll(
				io.LimitReader(
					file,
					maxProyectoWebImageSize+1,
				),
			)

			if err != nil {

				c.JSON(
					http.StatusInternalServerError,
					gin.H{
						"success": false,
						"message": "No se pudo leer la imagen",
					},
				)

				return
			}

			if int64(len(imageData)) >
				maxProyectoWebImageSize {

				c.JSON(
					http.StatusBadRequest,
					gin.H{
						"success": false,
						"message": "La imagen no puede superar los 5 MB",
					},
				)

				return
			}

			contentType :=
				http.DetectContentType(
					imageData,
				)

			if !imagenProyectoPermitida(
				contentType,
			) {

				c.JSON(
					http.StatusBadRequest,
					gin.H{
						"success": false,
						"message": "Solo se permiten imágenes JPG, PNG o WEBP",
					},
				)

				return
			}

			_, err = db.Exec(
				c,
				`
				UPDATE proyectos_web

				SET

					codigo = $1,
					titulo = $2,
					slug = $3,

					tipo = $4,
					ciudad = $5,
					direccion = $6,
					etapa = $7,

					imagen_nombre = $8,
					imagen_tipo = $9,
					imagen_tamano = $10,
					imagen_data = $11,

					dormitorios = $12,

					metraje_desde = $13,
					metraje_hasta = $14,

					estado = $15,

					precio_desde = $16,

					activo = $17,
					orden = $18,

					actualizado_en = NOW()

				WHERE id = $19
				`,
				codigo,
				titulo,
				slug,

				tipo,
				ciudad,
				direccion,
				etapa,

				imagen.Filename,
				contentType,
				int64(len(imageData)),
				imageData,

				dormitorios,

				metrajeDesde,
				metrajeHasta,

				estado,

				precioDesde,

				activo,
				orden,

				id,
			)

			if err != nil {

				c.JSON(
					http.StatusInternalServerError,
					gin.H{
						"success": false,
						"message": "No se pudo actualizar el proyecto",
						"error":   err.Error(),
					},
				)

				return
			}

		} else {

			/* =================================================
			   SIN CAMBIAR IMAGEN
			================================================= */

			_, err := db.Exec(
				c,
				`
				UPDATE proyectos_web

				SET

					codigo = $1,
					titulo = $2,
					slug = $3,

					tipo = $4,
					ciudad = $5,
					direccion = $6,
					etapa = $7,

					dormitorios = $8,

					metraje_desde = $9,
					metraje_hasta = $10,

					estado = $11,

					precio_desde = $12,

					activo = $13,
					orden = $14,

					actualizado_en = NOW()

				WHERE id = $15
				`,
				codigo,
				titulo,
				slug,

				tipo,
				ciudad,
				direccion,
				etapa,

				dormitorios,

				metrajeDesde,
				metrajeHasta,

				estado,

				precioDesde,

				activo,
				orden,

				id,
			)

			if err != nil {

				c.JSON(
					http.StatusInternalServerError,
					gin.H{
						"success": false,
						"message": "No se pudo actualizar el proyecto",
						"error":   err.Error(),
					},
				)

				return
			}
		}

		c.JSON(
			http.StatusOK,
			gin.H{
				"success": true,
				"message": "Proyecto actualizado correctamente",

				"data": gin.H{
					"id": id,

					"imagen_url":
						"/api/web/proyectos/" +
							id +
							"/imagen",
				},
			},
		)
	}
}

/* =========================================================
   ELIMINAR
========================================================= */

func eliminarProyectoWeb(db *pgxpool.Pool) gin.HandlerFunc {

	return func(c *gin.Context) {

		id := c.Param("id")

		result, err := db.Exec(
			c,
			`
			DELETE FROM proyectos_web
			WHERE id = $1
			`,
			id,
		)

		if err != nil {

			c.JSON(
				http.StatusInternalServerError,
				gin.H{
					"success": false,
					"message": "No se pudo eliminar el proyecto",
					"error":   err.Error(),
				},
			)

			return
		}

		if result.RowsAffected() == 0 {

			c.JSON(
				http.StatusNotFound,
				gin.H{
					"success": false,
					"message": "Proyecto no encontrado",
				},
			)

			return
		}

		c.JSON(
			http.StatusOK,
			gin.H{
				"success": true,
				"message": "Proyecto eliminado correctamente",
			},
		)
	}
}

/* =========================================================
   HELPERS
========================================================= */

func imagenProyectoPermitida(
	contentType string,
) bool {

	switch contentType {

	case
		"image/jpeg",
		"image/png",
		"image/webp":

		return true

	default:

		return false
	}
}

func parseIntProyecto(
	value string,
	fallback int,
) int {

	n, err := strconv.Atoi(
		value,
	)

	if err != nil {

		return fallback
	}

	return n
}

func parseBoolProyecto(
	value string,
	fallback bool,
) bool {

	b, err := strconv.ParseBool(
		value,
	)

	if err != nil {

		return fallback
	}

	return b
}

func nullableStringProyecto(
	value string,
) *string {

	value = strings.TrimSpace(
		value,
	)

	if value == "" {

		return nil
	}

	return &value
}

func nullableFloatProyecto(
	value string,
) *float64 {

	value = strings.TrimSpace(
		value,
	)

	if value == "" {

		return nil
	}

	n, err := strconv.ParseFloat(
		value,
		64,
	)

	if err != nil {

		return nil
	}

	return &n
}