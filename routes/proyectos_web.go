package routes

import (
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	maxProyectoWebImageSize int64 = 5 * 1024 * 1024
	maxProyectoWebLogoSize  int64 = 2 * 1024 * 1024
)

type ProyectoWeb struct {
	ID     string `json:"id"`
	Codigo string `json:"codigo"`
	Titulo string `json:"titulo"`
	Slug   string `json:"slug"`

	Tipo      string `json:"tipo"`
	Ciudad    string `json:"ciudad"`
	Direccion string `json:"direccion"`
	Etapa     string `json:"etapa"`

	Ruta     string  `json:"ruta"`
	Whatsapp *string `json:"whatsapp"`

	ImagenNombre string `json:"imagen_nombre"`
	ImagenTipo   string `json:"imagen_tipo"`
	ImagenTamano int64  `json:"imagen_tamano"`
	ImagenURL    string `json:"imagen_url"`

	LogoNombre string `json:"logo_nombre"`
	LogoTipo   string `json:"logo_tipo"`
	LogoTamano int64  `json:"logo_tamano"`
	LogoURL    string `json:"logo_url"`

	Dormitorios *string `json:"dormitorios"`

	MetrajeDesde *float64 `json:"metraje_desde"`
	MetrajeHasta *float64 `json:"metraje_hasta"`

	Estado     string   `json:"estado"`
	PrecioDesde *float64 `json:"precio_desde"`

	Activo bool `json:"activo"`
	Orden  int  `json:"orden"`

	CreadoEn      time.Time `json:"created_at"`
	ActualizadoEn time.Time `json:"updated_at"`
}

var tiposProyectoWeb = []string{
	"Departamento",
	"Lote",
	"Resort",
	"Casa",
}

var etapasProyectoWeb = []string{
	"PRE VENTA",
	"LANZAMIENTO",
	"EN CONSTRUCCIÓN",
	"ENTREGA INMEDIATA",
	"ENTREGADO",
	"TODOS VENDIDOS",
}

var estadosProyectoWeb = []string{
	"disponible",
	"vendido",
}

// RutasProyectosWeb registra:
// GET    /api/web/proyectos
// GET    /api/web/proyectos/opciones
// POST   /api/web/proyectos
// GET    /api/web/proyectos/:id
// GET    /api/web/proyectos/:id/imagen
// GET    /api/web/proyectos/:id/logo
// PUT    /api/web/proyectos/:id
// DELETE /api/web/proyectos/:id
func RutasProyectosWeb(api *gin.RouterGroup, db *pgxpool.Pool) {
	proyectos := api.Group("/web/proyectos")

	proyectos.GET("", listarProyectosWeb(db))
	proyectos.GET("/opciones", opcionesProyectosWeb())
	proyectos.POST("", crearProyectoWeb(db))

	proyectos.GET("/:id", obtenerProyectoWeb(db))
	proyectos.GET("/:id/imagen", obtenerImagenProyectoWeb(db))
	proyectos.GET("/:id/logo", obtenerLogoProyectoWeb(db))

	proyectos.PUT("/:id", actualizarProyectoWeb(db))
	proyectos.DELETE("/:id", eliminarProyectoWeb(db))
}

func opcionesProyectosWeb() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data": gin.H{
				"tipos":   tiposProyectoWeb,
				"etapas":  etapasProyectoWeb,
				"estados": estadosProyectoWeb,
			},
		})
	}
}

func listarProyectosWeb(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		page := parseIntProyecto(c.DefaultQuery("page", "1"), 1)
		limit := parseIntProyecto(c.DefaultQuery("limit", "10"), 10)

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

		buscar := strings.TrimSpace(c.Query("buscar"))
		estado := strings.TrimSpace(c.Query("estado"))
		tipo := strings.TrimSpace(c.Query("tipo"))
		ciudad := strings.TrimSpace(c.Query("ciudad"))
		etapa := strings.TrimSpace(c.Query("etapa"))
		activo := strings.TrimSpace(c.Query("activo"))

		args := make([]any, 0)
		where := []string{"1 = 1"}

		addArg := func(value any) string {
			args = append(args, value)
			return "$" + strconv.Itoa(len(args))
		}

		if buscar != "" {
			p := addArg("%" + buscar + "%")
			where = append(where, `(
				codigo ILIKE `+p+`
				OR titulo ILIKE `+p+`
				OR slug ILIKE `+p+`
				OR tipo ILIKE `+p+`
				OR ciudad ILIKE `+p+`
				OR direccion ILIKE `+p+`
				OR etapa ILIKE `+p+`
				OR COALESCE(ruta, '') ILIKE `+p+`
			)`)
		}

		if estado != "" {
			where = append(where, "estado = "+addArg(estado))
		}
		if tipo != "" {
			where = append(where, "tipo = "+addArg(tipo))
		}
		if ciudad != "" {
			where = append(where, "ciudad = "+addArg(ciudad))
		}
		if etapa != "" {
			where = append(where, "etapa = "+addArg(etapa))
		}
		if activo != "" {
			switch strings.ToLower(activo) {
			case "true":
				where = append(where, "activo = "+addArg(true))
			case "false":
				where = append(where, "activo = "+addArg(false))
			}
		}

		whereSQL := strings.Join(where, " AND ")

		var total int
		err := db.QueryRow(
			c.Request.Context(),
			`SELECT COUNT(*) FROM proyectos_web WHERE `+whereSQL,
			args...,
		).Scan(&total)
		if err != nil {
			responderError(c, http.StatusInternalServerError, "Error al contar proyectos", err)
			return
		}

		limitParam := len(args) + 1
		offsetParam := len(args) + 2
		queryArgs := append([]any{}, args...)
		queryArgs = append(queryArgs, limit, offset)

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
				COALESCE(ruta, ''),
				whatsapp,
				COALESCE(imagen_nombre, ''),
				COALESCE(imagen_tipo, ''),
				COALESCE(imagen_tamano, 0),
				COALESCE(logo_nombre, ''),
				COALESCE(logo_tipo, ''),
				COALESCE(logo_tamano, 0),
				dormitorios,
				metraje_desde,
				metraje_hasta,
				estado,
				precio_desde,
				activo,
				orden,
				created_at,
				updated_at
			FROM proyectos_web
			WHERE ` + whereSQL + `
			ORDER BY orden ASC, created_at DESC
			LIMIT $` + strconv.Itoa(limitParam) + `
			OFFSET $` + strconv.Itoa(offsetParam)

		rows, err := db.Query(c.Request.Context(), query, queryArgs...)
		if err != nil {
			responderError(c, http.StatusInternalServerError, "Error al obtener proyectos", err)
			return
		}
		defer rows.Close()

		proyectos := make([]ProyectoWeb, 0)

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
				&proyecto.Ruta,
				&proyecto.Whatsapp,
				&proyecto.ImagenNombre,
				&proyecto.ImagenTipo,
				&proyecto.ImagenTamano,
				&proyecto.LogoNombre,
				&proyecto.LogoTipo,
				&proyecto.LogoTamano,
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
				responderError(c, http.StatusInternalServerError, "Error al leer proyecto", err)
				return
			}

			normalizarProyectoRespuesta(&proyecto)
			proyectos = append(proyectos, proyecto)
		}

		if err := rows.Err(); err != nil {
			responderError(c, http.StatusInternalServerError, "Error al recorrer proyectos", err)
			return
		}

		totalPages := 0
		if total > 0 {
			totalPages = (total + limit - 1) / limit
		}

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data":    proyectos,
			"pagination": gin.H{
				"page":        page,
				"limit":       limit,
				"total":       total,
				"total_pages": totalPages,
			},
		})
	}
}

func crearProyectoWeb(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		codigo := strings.TrimSpace(c.PostForm("codigo"))
		titulo := strings.TrimSpace(c.PostForm("titulo"))
		slug := normalizarSlugProyecto(c.PostForm("slug"))
		tipo := strings.TrimSpace(c.PostForm("tipo"))
		ciudad := strings.TrimSpace(c.PostForm("ciudad"))
		direccion := strings.TrimSpace(c.PostForm("direccion"))
		etapa := strings.TrimSpace(c.PostForm("etapa"))

		if codigo == "" || titulo == "" || slug == "" || tipo == "" || ciudad == "" || direccion == "" || etapa == "" {
			responderMensaje(c, http.StatusBadRequest, "Completa todos los campos obligatorios")
			return
		}

		if !tipoProyectoPermitido(tipo) {
			responderMensaje(c, http.StatusBadRequest, "Tipo de proyecto inválido")
			return
		}

		if !etapaProyectoPermitida(etapa) {
			responderMensaje(c, http.StatusBadRequest, "Etapa inválida")
			return
		}

		ruta, whatsapp, err := resolverRutaWhatsapp(
			etapa,
			c.PostForm("ruta"),
			c.PostForm("whatsapp"),
			slug,
		)
		if err != nil {
			responderMensaje(c, http.StatusBadRequest, err.Error())
			return
		}

		estado := strings.ToLower(strings.TrimSpace(c.DefaultPostForm("estado", "disponible")))
		if !estadoProyectoPermitido(estado) {
			responderMensaje(c, http.StatusBadRequest, "Estado inválido")
			return
		}
		if etapa == "ENTREGADO" {
			estado = "vendido"
		}

		dormitorios := nullableStringProyecto(c.PostForm("dormitorios"))
		metrajeDesde := nullableFloatProyecto(c.PostForm("metraje_desde"))
		metrajeHasta := nullableFloatProyecto(c.PostForm("metraje_hasta"))
		precioDesde := nullableFloatProyecto(c.PostForm("precio_desde"))

		if estado == "vendido" {
			precioDesde = nil
		}

		if metrajeDesde != nil && *metrajeDesde <= 0 {
			responderMensaje(c, http.StatusBadRequest, "El metraje desde debe ser mayor a 0")
			return
		}
		if metrajeHasta != nil && *metrajeHasta <= 0 {
			responderMensaje(c, http.StatusBadRequest, "El metraje hasta debe ser mayor a 0")
			return
		}
		if metrajeDesde != nil && metrajeHasta != nil && *metrajeHasta < *metrajeDesde {
			responderMensaje(c, http.StatusBadRequest, "El metraje hasta no puede ser menor que el metraje desde")
			return
		}
		if precioDesde != nil && *precioDesde < 0 {
			responderMensaje(c, http.StatusBadRequest, "El precio no puede ser negativo")
			return
		}

		activo := parseBoolProyecto(c.DefaultPostForm("activo", "true"), true)
		orden := parseIntProyecto(c.DefaultPostForm("orden", "0"), 0)
		id := uuid.New()

		var imageData []byte
		var imageName, imageType string

		if file, fileErr := c.FormFile("imagen"); fileErr == nil {
			imageData, imageType, err = leerArchivoImagenProyecto(file, maxProyectoWebImageSize)
			if err != nil {
				responderMensaje(c, http.StatusBadRequest, err.Error())
				return
			}
			imageName = file.Filename
		}

		var logoData []byte
		var logoName, logoType string

		if file, fileErr := c.FormFile("logo"); fileErr == nil {
			logoData, logoType, err = leerArchivoLogoProyecto(file, maxProyectoWebLogoSize)
			if err != nil {
				responderMensaje(c, http.StatusBadRequest, err.Error())
				return
			}
			logoName = file.Filename
		}

		_, err = db.Exec(
			c.Request.Context(),
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
				ruta,
				whatsapp,
				imagen_nombre,
				imagen_tipo,
				imagen_tamano,
				imagen_data,
				logo_nombre,
				logo_tipo,
				logo_tamano,
				logo_data,
				dormitorios,
				metraje_desde,
				metraje_hasta,
				estado,
				precio_desde,
				activo,
				orden
			)
			VALUES (
				$1,$2,$3,$4,$5,$6,$7,$8,$9,$10,
				$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,
				$21,$22,$23,$24,$25
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
			nullableStringValue(ruta),
			whatsapp,
			nullableStringValue(imageName),
			nullableStringValue(imageType),
			nullableInt64Value(int64(len(imageData))),
			nullableBytesValue(imageData),
			nullableStringValue(logoName),
			nullableStringValue(logoType),
			nullableInt64Value(int64(len(logoData))),
			nullableBytesValue(logoData),
			dormitorios,
			metrajeDesde,
			metrajeHasta,
			estado,
			precioDesde,
			activo,
			orden,
		)

		if err != nil {
			responderError(c, http.StatusInternalServerError, "No se pudo crear el proyecto", err)
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"success": true,
			"message": "Proyecto creado correctamente",
			"data": gin.H{
				"id":         id.String(),
				"codigo":     codigo,
				"slug":       slug,
				"ruta":       ruta,
				"whatsapp":   whatsapp,
				"imagen_url": "/api/web/proyectos/" + id.String() + "/imagen",
				"logo_url":   "/api/web/proyectos/" + id.String() + "/logo",
			},
		})
	}
}

func obtenerProyectoWeb(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := validarUUIDProyecto(c)
		if !ok {
			return
		}

		proyecto, err := buscarProyectoPorID(c, db, id)
		if err != nil {
			if err == pgx.ErrNoRows {
				responderMensaje(c, http.StatusNotFound, "Proyecto no encontrado")
				return
			}
			responderError(c, http.StatusInternalServerError, "Error al obtener proyecto", err)
			return
		}

		normalizarProyectoRespuesta(&proyecto)

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data":    proyecto,
		})
	}
}

func obtenerImagenProyectoWeb(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := validarUUIDProyecto(c)
		if !ok {
			return
		}

		var data []byte
		var contentType string

		err := db.QueryRow(
			c.Request.Context(),
			`
			SELECT imagen_data, COALESCE(imagen_tipo, '')
			FROM proyectos_web
			WHERE id = $1
			`,
			id,
		).Scan(&data, &contentType)

		if err != nil {
			if err == pgx.ErrNoRows {
				responderMensaje(c, http.StatusNotFound, "Proyecto no encontrado")
				return
			}
			responderError(c, http.StatusInternalServerError, "Error al obtener imagen", err)
			return
		}

		if len(data) == 0 {
			responderMensaje(c, http.StatusNotFound, "El proyecto no tiene imagen")
			return
		}

		if contentType == "" {
			contentType = http.DetectContentType(data)
		}

		c.Header("Cache-Control", "public, max-age=86400")
		c.Header("Content-Disposition", "inline")
		c.Data(http.StatusOK, contentType, data)
	}
}

func obtenerLogoProyectoWeb(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := validarUUIDProyecto(c)
		if !ok {
			return
		}

		var data []byte
		var contentType string

		err := db.QueryRow(
			c.Request.Context(),
			`
			SELECT logo_data, COALESCE(logo_tipo, '')
			FROM proyectos_web
			WHERE id = $1
			`,
			id,
		).Scan(&data, &contentType)

		if err != nil {
			if err == pgx.ErrNoRows {
				responderMensaje(c, http.StatusNotFound, "Proyecto no encontrado")
				return
			}
			responderError(c, http.StatusInternalServerError, "Error al obtener logo", err)
			return
		}

		if len(data) == 0 {
			responderMensaje(c, http.StatusNotFound, "El proyecto no tiene logo")
			return
		}

		if contentType == "" {
			contentType = http.DetectContentType(data)
		}

		c.Header("Cache-Control", "public, max-age=86400")
		c.Header("Content-Disposition", "inline")
		c.Data(http.StatusOK, contentType, data)
	}
}

func actualizarProyectoWeb(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := validarUUIDProyecto(c)
		if !ok {
			return
		}

		// Para PUT multipart/form-data, Gin puede leer PostForm y FormFile.
		codigo := strings.TrimSpace(c.PostForm("codigo"))
		titulo := strings.TrimSpace(c.PostForm("titulo"))
		slug := normalizarSlugProyecto(c.PostForm("slug"))
		tipo := strings.TrimSpace(c.PostForm("tipo"))
		ciudad := strings.TrimSpace(c.PostForm("ciudad"))
		direccion := strings.TrimSpace(c.PostForm("direccion"))
		etapa := strings.TrimSpace(c.PostForm("etapa"))

		if codigo == "" || titulo == "" || slug == "" || tipo == "" || ciudad == "" || direccion == "" || etapa == "" {
			responderMensaje(c, http.StatusBadRequest, "Completa todos los campos obligatorios")
			return
		}

		if !tipoProyectoPermitido(tipo) {
			responderMensaje(c, http.StatusBadRequest, "Tipo de proyecto inválido")
			return
		}
		if !etapaProyectoPermitida(etapa) {
			responderMensaje(c, http.StatusBadRequest, "Etapa inválida")
			return
		}

		ruta, whatsapp, err := resolverRutaWhatsapp(
			etapa,
			c.PostForm("ruta"),
			c.PostForm("whatsapp"),
			slug,
		)
		if err != nil {
			responderMensaje(c, http.StatusBadRequest, err.Error())
			return
		}

		estado := strings.ToLower(strings.TrimSpace(c.DefaultPostForm("estado", "disponible")))
		if !estadoProyectoPermitido(estado) {
			responderMensaje(c, http.StatusBadRequest, "Estado inválido")
			return
		}
		if etapa == "ENTREGADO" {
			estado = "vendido"
		}

		dormitorios := nullableStringProyecto(c.PostForm("dormitorios"))
		metrajeDesde := nullableFloatProyecto(c.PostForm("metraje_desde"))
		metrajeHasta := nullableFloatProyecto(c.PostForm("metraje_hasta"))
		precioDesde := nullableFloatProyecto(c.PostForm("precio_desde"))

		if estado == "vendido" {
			precioDesde = nil
		}

		if metrajeDesde != nil && *metrajeDesde <= 0 {
			responderMensaje(c, http.StatusBadRequest, "El metraje desde debe ser mayor a 0")
			return
		}
		if metrajeHasta != nil && *metrajeHasta <= 0 {
			responderMensaje(c, http.StatusBadRequest, "El metraje hasta debe ser mayor a 0")
			return
		}
		if metrajeDesde != nil && metrajeHasta != nil && *metrajeHasta < *metrajeDesde {
			responderMensaje(c, http.StatusBadRequest, "El metraje hasta no puede ser menor que el metraje desde")
			return
		}
		if precioDesde != nil && *precioDesde < 0 {
			responderMensaje(c, http.StatusBadRequest, "El precio no puede ser negativo")
			return
		}

		activo := parseBoolProyecto(c.DefaultPostForm("activo", "true"), true)
		orden := parseIntProyecto(c.DefaultPostForm("orden", "0"), 0)

		// IMPORTANTE:
		// Los campos imagen/logo solamente se reemplazan si se envía
		// un archivo nuevo. Si no se envían, se conserva el anterior.
		imagen, imagenErr := c.FormFile("imagen")
		logo, logoErr := c.FormFile("logo")

		var imageData []byte
		var imageType string
		var imageName string
		hasImage := imagenErr == nil

		if hasImage {
			imageData, imageType, err = leerArchivoImagenProyecto(imagen, maxProyectoWebImageSize)
			if err != nil {
				responderMensaje(c, http.StatusBadRequest, err.Error())
				return
			}
			imageName = imagen.Filename
		}

		var logoData []byte
		var logoType string
		var logoName string
		hasLogo := logoErr == nil

		if hasLogo {
			logoData, logoType, err = leerArchivoLogoProyecto(logo, maxProyectoWebLogoSize)
			if err != nil {
				responderMensaje(c, http.StatusBadRequest, err.Error())
				return
			}
			logoName = logo.Filename
		}

		sets := []string{
			"codigo = $1",
			"titulo = $2",
			"slug = $3",
			"tipo = $4",
			"ciudad = $5",
			"direccion = $6",
			"etapa = $7",
			"ruta = $8",
			"whatsapp = $9",
			"dormitorios = $10",
			"metraje_desde = $11",
			"metraje_hasta = $12",
			"estado = $13",
			"precio_desde = $14",
			"activo = $15",
			"orden = $16",
			"updated_at = NOW()",
		}

		args := []any{
			codigo,
			titulo,
			slug,
			tipo,
			ciudad,
			direccion,
			etapa,
			nullableStringValue(ruta),
			whatsapp,
			dormitorios,
			metrajeDesde,
			metrajeHasta,
			estado,
			precioDesde,
			activo,
			orden,
		}

		next := 17

		if hasImage {
			sets = append(sets,
				"imagen_nombre = $"+strconv.Itoa(next),
				"imagen_tipo = $"+strconv.Itoa(next+1),
				"imagen_tamano = $"+strconv.Itoa(next+2),
				"imagen_data = $"+strconv.Itoa(next+3),
			)
			args = append(args,
				imageName,
				imageType,
				int64(len(imageData)),
				imageData,
			)
			next += 4
		}

		if hasLogo {
			sets = append(sets,
				"logo_nombre = $"+strconv.Itoa(next),
				"logo_tipo = $"+strconv.Itoa(next+1),
				"logo_tamano = $"+strconv.Itoa(next+2),
				"logo_data = $"+strconv.Itoa(next+3),
			)
			args = append(args,
				logoName,
				logoType,
				int64(len(logoData)),
				logoData,
			)
			next += 4
		}

		args = append(args, id)

		query := `
			UPDATE proyectos_web
			SET ` + strings.Join(sets, ", ") + `
			WHERE id = $` + strconv.Itoa(next)

		result, err := db.Exec(c.Request.Context(), query, args...)
		if err != nil {
			responderError(c, http.StatusInternalServerError, "No se pudo actualizar el proyecto", err)
			return
		}

		if result.RowsAffected() == 0 {
			responderMensaje(c, http.StatusNotFound, "Proyecto no encontrado")
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "Proyecto actualizado correctamente",
			"data": gin.H{
				"id":         id,
				"codigo":     codigo,
				"slug":       slug,
				"ruta":       ruta,
				"whatsapp":   whatsapp,
				"imagen_url": "/api/web/proyectos/" + id + "/imagen",
				"logo_url":   "/api/web/proyectos/" + id + "/logo",
			},
		})
	}
}

func eliminarProyectoWeb(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := validarUUIDProyecto(c)
		if !ok {
			return
		}

		result, err := db.Exec(
			c.Request.Context(),
			`DELETE FROM proyectos_web WHERE id = $1`,
			id,
		)
		if err != nil {
			responderError(c, http.StatusInternalServerError, "No se pudo eliminar el proyecto", err)
			return
		}

		if result.RowsAffected() == 0 {
			responderMensaje(c, http.StatusNotFound, "Proyecto no encontrado")
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "Proyecto eliminado correctamente",
		})
	}
}

func buscarProyectoPorID(c *gin.Context, db *pgxpool.Pool, id string) (ProyectoWeb, error) {
	var proyecto ProyectoWeb

	err := db.QueryRow(
		c.Request.Context(),
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
			COALESCE(ruta, ''),
			whatsapp,
			COALESCE(imagen_nombre, ''),
			COALESCE(imagen_tipo, ''),
			COALESCE(imagen_tamano, 0),
			COALESCE(logo_nombre, ''),
			COALESCE(logo_tipo, ''),
			COALESCE(logo_tamano, 0),
			dormitorios,
			metraje_desde,
			metraje_hasta,
			estado,
			precio_desde,
			activo,
			orden,
			created_at,
			updated_at
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
		&proyecto.Ruta,
		&proyecto.Whatsapp,
		&proyecto.ImagenNombre,
		&proyecto.ImagenTipo,
		&proyecto.ImagenTamano,
		&proyecto.LogoNombre,
		&proyecto.LogoTipo,
		&proyecto.LogoTamano,
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

	return proyecto, err
}

func resolverRutaWhatsapp(etapa, rutaInput, whatsappInput, slug string) (string, *string, error) {
	ruta := strings.TrimSpace(rutaInput)
	whatsapp := nullableWhatsappProyecto(whatsappInput)

	if etapa == "ENTREGADO" {
		if whatsapp == nil {
			return "", nil, fmt.Errorf("para un proyecto ENTREGADO debes registrar un número de WhatsApp")
		}
		return "", whatsapp, nil
	}

	if ruta == "" {
		ruta = "/" + normalizarSlugProyecto(slug)
	} else {
		ruta = normalizarRutaProyecto(ruta)
	}

	return ruta, whatsapp, nil
}

func normalizarProyectoRespuesta(proyecto *ProyectoWeb) {
	if proyecto.Etapa == "ENTREGADO" {
		proyecto.Ruta = ""
	} else if proyecto.Ruta == "" {
		proyecto.Ruta = "/" + normalizarSlugProyecto(proyecto.Slug)
	} else {
		proyecto.Ruta = normalizarRutaProyecto(proyecto.Ruta)
	}

	proyecto.ImagenURL = "/api/web/proyectos/" + proyecto.ID + "/imagen"
	proyecto.LogoURL = "/api/web/proyectos/" + proyecto.ID + "/logo"
}

func normalizarSlugProyecto(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	value = strings.ReplaceAll(value, " ", "-")
	value = strings.ReplaceAll(value, "_", "-")

	var b strings.Builder
	for _, r := range value {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' {
			b.WriteRune(r)
		}
	}

	value = b.String()

	for strings.Contains(value, "--") {
		value = strings.ReplaceAll(value, "--", "-")
	}

	return strings.Trim(value, "-")
}

func normalizarRutaProyecto(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}

	if strings.HasPrefix(value, "http://") || strings.HasPrefix(value, "https://") {
		return value
	}

	return "/" + strings.Trim(value, "/")
}

func nullableWhatsappProyecto(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}

	var b strings.Builder
	for i, r := range value {
		if r == '+' && i == 0 {
			b.WriteRune(r)
			continue
		}
		if unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}

	result := b.String()
	if result == "" || result == "+" {
		return nil
	}

	return &result
}

func nullableStringProyecto(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return &value
}

func nullableStringValue(value string) any {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return value
}

func nullableBytesValue(value []byte) any {
	if len(value) == 0 {
		return nil
	}
	return value
}

func nullableInt64Value(value int64) any {
	if value <= 0 {
		return nil
	}
	return value
}

func nullableFloatProyecto(value string) *float64 {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}

	value = strings.ReplaceAll(value, ",", ".")

	n, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return nil
	}

	return &n
}

func parseIntProyecto(value string, fallback int) int {
	n, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return fallback
	}
	return n
}

func parseBoolProyecto(value string, fallback bool) bool {
	b, err := strconv.ParseBool(strings.TrimSpace(value))
	if err != nil {
		return fallback
	}
	return b
}

func tipoProyectoPermitido(value string) bool {
	for _, item := range tiposProyectoWeb {
		if value == item {
			return true
		}
	}
	return false
}

func etapaProyectoPermitida(value string) bool {
	for _, item := range etapasProyectoWeb {
		if value == item {
			return true
		}
	}
	return false
}

func estadoProyectoPermitido(value string) bool {
	for _, item := range estadosProyectoWeb {
		if value == item {
			return true
		}
	}
	return false
}

func validarUUIDProyecto(c *gin.Context) (string, bool) {
	id := strings.TrimSpace(c.Param("id"))

	parsed, err := uuid.Parse(id)
	if err != nil {
		responderMensaje(c, http.StatusBadRequest, "ID de proyecto inválido")
		return "", false
	}

	return parsed.String(), true
}

func leerArchivoImagenProyecto(header *multipart.FileHeader, maxSize int64) ([]byte, string, error) {
	if header == nil {
		return nil, "", fmt.Errorf("archivo de imagen no encontrado")
	}

	if header.Size <= 0 {
		return nil, "", fmt.Errorf("la imagen está vacía")
	}

	if header.Size > maxSize {
		return nil, "", fmt.Errorf("la imagen no puede superar los 5 MB")
	}

	file, err := header.Open()
	if err != nil {
		return nil, "", fmt.Errorf("no se pudo abrir la imagen")
	}
	defer file.Close()

	data, err := io.ReadAll(io.LimitReader(file, maxSize+1))
	if err != nil {
		return nil, "", fmt.Errorf("no se pudo leer la imagen")
	}

	if int64(len(data)) > maxSize {
		return nil, "", fmt.Errorf("la imagen no puede superar los 5 MB")
	}

	contentType := http.DetectContentType(data)
	if !imagenProyectoPermitida(contentType) {
		return nil, "", fmt.Errorf("solo se permiten imágenes JPG, PNG o WEBP")
	}

	return data, contentType, nil
}

func leerArchivoLogoProyecto(header *multipart.FileHeader, maxSize int64) ([]byte, string, error) {
	if header == nil {
		return nil, "", fmt.Errorf("archivo de logo no encontrado")
	}

	if header.Size <= 0 {
		return nil, "", fmt.Errorf("el logo está vacío")
	}

	if header.Size > maxSize {
		return nil, "", fmt.Errorf("el logo no puede superar los 2 MB")
	}

	file, err := header.Open()
	if err != nil {
		return nil, "", fmt.Errorf("no se pudo abrir el logo")
	}
	defer file.Close()

	data, err := io.ReadAll(io.LimitReader(file, maxSize+1))
	if err != nil {
		return nil, "", fmt.Errorf("no se pudo leer el logo")
	}

	if int64(len(data)) > maxSize {
		return nil, "", fmt.Errorf("el logo no puede superar los 2 MB")
	}

	contentType := detectarTipoLogo(data, header.Filename)
	if !logoProyectoPermitido(contentType) {
		return nil, "", fmt.Errorf("solo se permiten logos SVG, PNG, JPG o WEBP")
	}

	return data, contentType, nil
}

func imagenProyectoPermitida(contentType string) bool {
	switch strings.ToLower(contentType) {
	case "image/jpeg", "image/png", "image/webp":
		return true
	default:
		return false
	}
}

func logoProyectoPermitido(contentType string) bool {
	switch strings.ToLower(contentType) {
	case "image/svg+xml", "image/png", "image/jpeg", "image/webp":
		return true
	default:
		return false
	}
}

func detectarTipoLogo(data []byte, filename string) string {
	if strings.HasSuffix(strings.ToLower(filename), ".svg") {
		return "image/svg+xml"
	}
	return http.DetectContentType(data)
}

func responderMensaje(c *gin.Context, status int, message string) {
	c.JSON(status, gin.H{
		"success": false,
		"message": message,
	})
}

func responderError(c *gin.Context, status int, message string, err error) {
	c.JSON(status, gin.H{
		"success": false,
		"message": message,
		"error":   err.Error(),
	})
}
