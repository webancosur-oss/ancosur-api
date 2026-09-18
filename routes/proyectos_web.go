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

const maxProyectoWebImageSize int64 = 5 * 1024 * 1024 // 5 MB

/* =========================================================
   MODELO
========================================================= */

type ProyectoWeb struct {
	ID string `json:"id"`

	Codigo    string `json:"codigo"`
	Titulo    string `json:"titulo"`
	Slug      string `json:"slug"`
	Tipo      string `json:"tipo"`
	Ciudad    string `json:"ciudad"`
	Direccion string `json:"direccion"`
	Etapa     string `json:"etapa"`

	Dormitorios  *string  `json:"dormitorios"`
	MetrajeDesde *float64 `json:"metraje_desde"`
	MetrajeHasta *float64 `json:"metraje_hasta"`

	Estado      string   `json:"estado"`
	PrecioDesde *float64 `json:"precio_desde"`

	Activo bool `json:"activo"`
	Orden  int  `json:"orden"`

	ImagenNombre string `json:"imagen_nombre"`
	ImagenTipo   string `json:"imagen_tipo"`
	ImagenTamano int64  `json:"imagen_tamano"`

	ImagenURL string `json:"imagen_url"`

	CreadoEn      time.Time `json:"creado_en"`
	ActualizadoEn time.Time `json:"actualizado_en"`
}

/* =========================================================
   REQUEST
========================================================= */

type CrearProyectoWebRequest struct {
	Codigo       string   `json:"codigo"`
	Titulo       string   `json:"titulo"`
	Slug         string   `json:"slug"`
	Tipo         string   `json:"tipo"`
	Ciudad       string   `json:"ciudad"`
	Direccion    string   `json:"direccion"`
	Etapa        string   `json:"etapa"`
	Dormitorios  *string  `json:"dormitorios"`
	MetrajeDesde *float64 `json:"metraje_desde"`
	MetrajeHasta *float64 `json:"metraje_hasta"`
	Estado       string   `json:"estado"`
	PrecioDesde  *float64 `json:"precio_desde"`
	Activo       bool     `json:"activo"`
	Orden        int      `json:"orden"`
}

type ActualizarProyectoWebRequest struct {
	Codigo       string   `json:"codigo"`
	Titulo       string   `json:"titulo"`
	Slug         string   `json:"slug"`
	Tipo         string   `json:"tipo"`
	Ciudad       string   `json:"ciudad"`
	Direccion    string   `json:"direccion"`
	Etapa        string   `json:"etapa"`
	Dormitorios  *string  `json:"dormitorios"`
	MetrajeDesde *float64 `json:"metraje_desde"`
	MetrajeHasta *float64 `json:"metraje_hasta"`
	Estado       string   `json:"estado"`
	PrecioDesde  *float64 `json:"precio_desde"`
	Activo       bool     `json:"activo"`
	Orden        int      `json:"orden"`
}

/* =========================================================
   RUTAS
========================================================= */

func RutasProyectosWeb(
	api *gin.RouterGroup,
	db *pgxpool.Pool,
) {

	/*
		LISTAR
		DASHBOARD
	*/
	api.GET(
		"/web/proyectos",
		listarProyectosWeb(db),
	)

	/*
		CREAR
	*/
	api.POST(
		"/web/proyectos",
		crearProyectoWeb(db),
	)

	/*
		OBTENER POR ID
	*/
	api.GET(
		"/web/proyectos/:id",
		obtenerProyectoWeb(db),
	)

	/*
		IMAGEN
	*/
	api.GET(
		"/web/proyectos/:id/imagen",
		obtenerImagenProyectoWeb(db),
	)

	/*
		EDITAR
	*/
	api.PUT(
		"/web/proyectos/:id",
		actualizarProyectoWeb(db),
	)

	/*
		ELIMINAR
	*/
	api.DELETE(
		"/web/proyectos/:id",
		eliminarProyectoWeb(db),
	)
}

/* =========================================================
   CREAR PROYECTO
========================================================= */

func crearProyectoWeb(
	db *pgxpool.Pool,
) gin.HandlerFunc {

	return func(c *gin.Context) {

		/* =========================================
		   MULTIPART
		========================================= */

		if err := c.Request.ParseMultipartForm(
			maxProyectoWebImageSize,
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

		estado := strings.TrimSpace(
			c.PostForm("estado"),
		)

		if estado == "" {
			estado = "disponible"
		}

		activo := parseBool(
			c.PostForm("activo"),
			true,
		)

		orden := parseInt(
			c.PostForm("orden"),
			0,
		)

		dormitorios := nullableString(
			c.PostForm("dormitorios"),
		)

		metrajeDesde := parseNullableFloat(
			c.PostForm("metraje_desde"),
		)

		metrajeHasta := parseNullableFloat(
			c.PostForm("metraje_hasta"),
		)

		precioDesde := parseNullableFloat(
			c.PostForm("precio_desde"),
		)

		/* =========================================
		   VALIDACIONES
		========================================= */

		if codigo == "" {
			c.JSON(
				http.StatusBadRequest,
				gin.H{
					"success": false,
					"message": "El código es obligatorio.",
				},
			)
			return
		}

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

		if slug == "" {
			c.JSON(
				http.StatusBadRequest,
				gin.H{
					"success": false,
					"message": "El slug es obligatorio.",
				},
			)
			return
		}

		if tipo == "" {
			c.JSON(
				http.StatusBadRequest,
				gin.H{
					"success": false,
					"message": "El tipo es obligatorio.",
				},
			)
			return
		}

		if ciudad == "" {
			c.JSON(
				http.StatusBadRequest,
				gin.H{
					"success": false,
					"message": "La ciudad es obligatoria.",
				},
			)
			return
		}

		if direccion == "" {
			c.JSON(
				http.StatusBadRequest,
				gin.H{
					"success": false,
					"message": "La dirección es obligatoria.",
				},
			)
			return
		}

		if etapa == "" {
			c.JSON(
				http.StatusBadRequest,
				gin.H{
					"success": false,
					"message": "La etapa es obligatoria.",
				},
			)
			return
		}

		/* =========================================
		   IMAGEN
		========================================= */

		file, header, err := c.Request.FormFile(
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
		   TAMAÑO
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

		if header.Size > maxProyectoWebImageSize {
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
					"message": "No se pudo leer la imagen.",
				},
			)

			return
		}

		if int64(len(imageData)) > maxProyectoWebImageSize {
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
		   MIME REAL
		========================================= */

		contentType := http.DetectContentType(
			imageData,
		)

		if !imagenProyectoPermitida(contentType) {
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

		var proyecto ProyectoWeb

		err = db.QueryRow(
			context.Background(),
			`
			INSERT INTO proyectos_web (
				codigo,
				titulo,
				slug,
				tipo,
				ciudad,
				direccion,
				etapa,

				dormitorios,
				metraje_desde,
				metraje_hasta,

				estado,
				precio_desde,

				activo,
				orden,

				imagen_nombre,
				imagen_tipo,
				imagen_tamano,
				imagen_data,

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

				NOW(),
				NOW()
			)
			RETURNING
				id,
				codigo,
				titulo,
				slug,
				tipo,
				ciudad,
				direccion,
				etapa,

				dormitorios,
				metraje_desde,
				metraje_hasta,

				estado,
				precio_desde,

				activo,
				orden,

				imagen_nombre,
				imagen_tipo,
				imagen_tamano,

				creado_en,
				actualizado_en
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

			header.Filename,
			contentType,
			len(imageData),
			imageData,
		).Scan(
			&proyecto.ID,
			&proyecto.Codigo,
			&proyecto.Titulo,
			&proyecto.Slug,
			&proyecto.Tipo,
			&proyecto.Ciudad,
			&proyecto.Direccion,
			&proyecto.Etapa,

			&proyecto.Dormitorios,
			&proyecto.MetrajeDesde,
			&proyecto.MetrajeHasta,

			&proyecto.Estado,
			&proyecto.PrecioDesde,

			&proyecto.Activo,
			&proyecto.Orden,

			&proyecto.ImagenNombre,
			&proyecto.ImagenTipo,
			&proyecto.ImagenTamano,

			&proyecto.CreadoEn,
			&proyecto.ActualizadoEn,
		)

		if err != nil {

			fmt.Println(
				"Error creando proyecto web:",
				err,
			)

			c.JSON(
				http.StatusInternalServerError,
				gin.H{
					"success": false,
					"message": "No se pudo registrar el proyecto.",
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
			http.StatusCreated,
			gin.H{
				"success": true,
				"message": "Proyecto registrado correctamente.",
				"data":    proyecto,
			},
		)
	}
}

/* =========================================================
   LISTAR
========================================================= */

func listarProyectosWeb(
	db *pgxpool.Pool,
) gin.HandlerFunc {

	return func(c *gin.Context) {

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

				dormitorios,
				metraje_desde,
				metraje_hasta,

				estado,
				precio_desde,

				activo,
				orden,

				imagen_nombre,
				imagen_tipo,
				imagen_tamano,

				creado_en,
				actualizado_en

			FROM proyectos_web

			ORDER BY
				orden ASC,
				creado_en DESC
		`

		rows, err := db.Query(
			context.Background(),
			query,
		)

		if err != nil {

			c.JSON(
				http.StatusInternalServerError,
				gin.H{
					"success": false,
					"message": "No se pudieron cargar los proyectos.",
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

			var item ProyectoWeb

			err := rows.Scan(
				&item.ID,
				&item.Codigo,
				&item.Titulo,
				&item.Slug,
				&item.Tipo,
				&item.Ciudad,
				&item.Direccion,
				&item.Etapa,

				&item.Dormitorios,
				&item.MetrajeDesde,
				&item.MetrajeHasta,

				&item.Estado,
				&item.PrecioDesde,

				&item.Activo,
				&item.Orden,

				&item.ImagenNombre,
				&item.ImagenTipo,
				&item.ImagenTamano,

				&item.CreadoEn,
				&item.ActualizadoEn,
			)

			if err != nil {

				c.JSON(
					http.StatusInternalServerError,
					gin.H{
						"success": false,
						"message": "No se pudo leer un proyecto.",
						"error":   err.Error(),
					},
				)

				return
			}

			item.ImagenURL =
				"/api/web/proyectos/" +
					item.ID +
					"/imagen"

			proyectos = append(
				proyectos,
				item,
			)
		}

		if err := rows.Err(); err != nil {

			c.JSON(
				http.StatusInternalServerError,
				gin.H{
					"success": false,
					"message": "Error recorriendo los proyectos.",
					"error":   err.Error(),
				},
			)

			return
		}

		c.JSON(
			http.StatusOK,
			gin.H{
				"success": true,
				"total":   len(proyectos),
				"data":    proyectos,
			},
		)
	}
}

/* =========================================================
   OBTENER PROYECTO
========================================================= */

func obtenerProyectoWeb(
	db *pgxpool.Pool,
) gin.HandlerFunc {

	return func(c *gin.Context) {

		id := strings.TrimSpace(
			c.Param("id"),
		)

		if id == "" {
			c.JSON(
				http.StatusBadRequest,
				gin.H{
					"success": false,
					"message": "ID de proyecto requerido.",
				},
			)

			return
		}

		var item ProyectoWeb

		err := db.QueryRow(
			context.Background(),
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

				dormitorios,
				metraje_desde,
				metraje_hasta,

				estado,
				precio_desde,

				activo,
				orden,

				imagen_nombre,
				imagen_tipo,
				imagen_tamano,

				creado_en,
				actualizado_en

			FROM proyectos_web

			WHERE id = $1
			`,
			id,
		).Scan(
			&item.ID,
			&item.Codigo,
			&item.Titulo,
			&item.Slug,
			&item.Tipo,
			&item.Ciudad,
			&item.Direccion,
			&item.Etapa,

			&item.Dormitorios,
			&item.MetrajeDesde,
			&item.MetrajeHasta,

			&item.Estado,
			&item.PrecioDesde,

			&item.Activo,
			&item.Orden,

			&item.ImagenNombre,
			&item.ImagenTipo,
			&item.ImagenTamano,

			&item.CreadoEn,
			&item.ActualizadoEn,
		)

		if err == pgx.ErrNoRows {

			c.JSON(
				http.StatusNotFound,
				gin.H{
					"success": false,
					"message": "Proyecto no encontrado.",
				},
			)

			return
		}

		if err != nil {

			c.JSON(
				http.StatusInternalServerError,
				gin.H{
					"success": false,
					"message": "No se pudo obtener el proyecto.",
					"error":   err.Error(),
				},
			)

			return
		}

		item.ImagenURL =
			"/api/web/proyectos/" +
				item.ID +
				"/imagen"

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
   OBTENER IMAGEN
========================================================= */

func obtenerImagenProyectoWeb(
	db *pgxpool.Pool,
) gin.HandlerFunc {

	return func(c *gin.Context) {

		id := strings.TrimSpace(
			c.Param("id"),
		)

		if id == "" {

			c.JSON(
				http.StatusBadRequest,
				gin.H{
					"success": false,
					"message": "ID de proyecto requerido.",
				},
			)

			return
		}

		var (
			imageData []byte
			imageType string
		)

		err := db.QueryRow(
			context.Background(),
			`
			SELECT
				imagen_data,
				imagen_tipo

			FROM proyectos_web

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
					"message": "Proyecto no encontrado.",
				},
			)

			return
		}

		if err != nil {

			fmt.Println(
				"Error obteniendo imagen proyecto:",
				err,
			)

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

		if len(imageData) == 0 {

			c.JSON(
				http.StatusNotFound,
				gin.H{
					"success": false,
					"message": "El proyecto no tiene una imagen.",
				},
			)

			return
		}

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
   ACTUALIZAR
========================================================= */

func actualizarProyectoWeb(
	db *pgxpool.Pool,
) gin.HandlerFunc {

	return func(c *gin.Context) {

		id := strings.TrimSpace(
			c.Param("id"),
		)

		if id == "" {

			c.JSON(
				http.StatusBadRequest,
				gin.H{
					"success": false,
					"message": "ID de proyecto requerido.",
				},
			)

			return
		}

		/* =========================================
		   VERIFICAR EXISTENCIA
		========================================= */

		var existe bool

		err := db.QueryRow(
			context.Background(),
			`
			SELECT EXISTS(
				SELECT 1
				FROM proyectos_web
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
					"message": "No se pudo verificar el proyecto.",
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
					"message": "Proyecto no encontrado.",
				},
			)

			return
		}

		/* =========================================
		   CAMPOS
		========================================= */

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

		estado := strings.TrimSpace(
			c.PostForm("estado"),
		)

		if estado == "" {
			estado = "disponible"
		}

		activo := parseBool(
			c.PostForm("activo"),
			true,
		)

		orden := parseInt(
			c.PostForm("orden"),
			0,
		)

		dormitorios := nullableString(
			c.PostForm("dormitorios"),
		)

		metrajeDesde := parseNullableFloat(
			c.PostForm("metraje_desde"),
		)

		metrajeHasta := parseNullableFloat(
			c.PostForm("metraje_hasta"),
		)

		precioDesde := parseNullableFloat(
			c.PostForm("precio_desde"),
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
					"message": "Completa todos los campos obligatorios.",
				},
			)

			return
		}

		/* =========================================
		   NUEVA IMAGEN
		========================================= */

		file, header, imageErr := c.Request.FormFile(
			"imagen",
		)

		/* =========================================
		   SIN NUEVA IMAGEN
		========================================= */

		if imageErr != nil {

			var item ProyectoWeb

			err := db.QueryRow(
				context.Background(),
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

				RETURNING
					id,
					codigo,
					titulo,
					slug,
					tipo,
					ciudad,
					direccion,
					etapa,

					dormitorios,
					metraje_desde,
					metraje_hasta,

					estado,
					precio_desde,

					activo,
					orden,

					imagen_nombre,
					imagen_tipo,
					imagen_tamano,

					creado_en,
					actualizado_en
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
			).Scan(
				&item.ID,
				&item.Codigo,
				&item.Titulo,
				&item.Slug,
				&item.Tipo,
				&item.Ciudad,
				&item.Direccion,
				&item.Etapa,

				&item.Dormitorios,
				&item.MetrajeDesde,
				&item.MetrajeHasta,

				&item.Estado,
				&item.PrecioDesde,

				&item.Activo,
				&item.Orden,

				&item.ImagenNombre,
				&item.ImagenTipo,
				&item.ImagenTamano,

				&item.CreadoEn,
				&item.ActualizadoEn,
			)

			if err != nil {

				c.JSON(
					http.StatusInternalServerError,
					gin.H{
						"success": false,
						"message": "No se pudo actualizar el proyecto.",
						"error":   err.Error(),
					},
				)

				return
			}

			item.ImagenURL =
				"/api/web/proyectos/" +
					item.ID +
					"/imagen"

			c.JSON(
				http.StatusOK,
				gin.H{
					"success": true,
					"message": "Proyecto actualizado correctamente.",
					"data":    item,
				},
			)

			return
		}

		defer file.Close()

		/* =========================================
		   VALIDAR NUEVA IMAGEN
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

		if header.Size > maxProyectoWebImageSize {

			c.JSON(
				http.StatusRequestEntityTooLarge,
				gin.H{
					"success": false,
					"message": "La imagen no debe superar los 5 MB.",
				},
			)

			return
		}

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
					"message": "No se pudo leer la nueva imagen.",
				},
			)

			return
		}

		if int64(len(imageData)) > maxProyectoWebImageSize {

			c.JSON(
				http.StatusRequestEntityTooLarge,
				gin.H{
					"success": false,
					"message": "La imagen no debe superar los 5 MB.",
				},
			)

			return
		}

		contentType := http.DetectContentType(
			imageData,
		)

		if !imagenProyectoPermitida(contentType) {

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

		var item ProyectoWeb

		err = db.QueryRow(
			context.Background(),
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

				imagen_nombre = $15,
				imagen_tipo = $16,
				imagen_tamano = $17,
				imagen_data = $18,

				actualizado_en = NOW()

			WHERE id = $19

			RETURNING
				id,
				codigo,
				titulo,
				slug,
				tipo,
				ciudad,
				direccion,
				etapa,

				dormitorios,
				metraje_desde,
				metraje_hasta,

				estado,
				precio_desde,

				activo,
				orden,

				imagen_nombre,
				imagen_tipo,
				imagen_tamano,

				creado_en,
				actualizado_en
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

			header.Filename,
			contentType,
			len(imageData),
			imageData,

			id,
		).Scan(
			&item.ID,
			&item.Codigo,
			&item.Titulo,
			&item.Slug,
			&item.Tipo,
			&item.Ciudad,
			&item.Direccion,
			&item.Etapa,

			&item.Dormitorios,
			&item.MetrajeDesde,
			&item.MetrajeHasta,

			&item.Estado,
			&item.PrecioDesde,

			&item.Activo,
			&item.Orden,

			&item.ImagenNombre,
			&item.ImagenTipo,
			&item.ImagenTamano,

			&item.CreadoEn,
			&item.ActualizadoEn,
		)

		if err != nil {

			c.JSON(
				http.StatusInternalServerError,
				gin.H{
					"success": false,
					"message": "No se pudo actualizar el proyecto.",
					"error":   err.Error(),
				},
			)

			return
		}

		item.ImagenURL =
			"/api/web/proyectos/" +
				item.ID +
				"/imagen"

		c.JSON(
			http.StatusOK,
			gin.H{
				"success": true,
				"message": "Proyecto actualizado correctamente.",
				"data":    item,
			},
		)
	}
}

/* =========================================================
   ELIMINAR
========================================================= */

func eliminarProyectoWeb(
	db *pgxpool.Pool,
) gin.HandlerFunc {

	return func(c *gin.Context) {

		id := strings.TrimSpace(
			c.Param("id"),
		)

		if id == "" {

			c.JSON(
				http.StatusBadRequest,
				gin.H{
					"success": false,
					"message": "ID de proyecto requerido.",
				},
			)

			return
		}

		result, err := db.Exec(
			context.Background(),
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
					"message": "No se pudo eliminar el proyecto.",
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
					"message": "Proyecto no encontrado.",
				},
			)

			return
		}

		c.JSON(
			http.StatusOK,
			gin.H{
				"success": true,
				"message": "Proyecto eliminado correctamente.",
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

func parseInt(
	value string,
	defaultValue int,
) int {

	value = strings.TrimSpace(
		value,
	)

	if value == "" {
		return defaultValue
	}

	result, err := strconv.Atoi(
		value,
	)

	if err != nil {
		return defaultValue
	}

	return result
}

func parseNullableFloat(
	value string,
) *float64 {

	value = strings.TrimSpace(
		value,
	)

	if value == "" {
		return nil
	}

	result, err := strconv.ParseFloat(
		value,
		64,
	)

	if err != nil {
		return nil
	}

	return &result
}

func nullableString(
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
