package controller

import (
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"ancosur-api/services"
)

type ProyectosWebController struct {
	Service *services.ProyectoWebService
}

func NewProyectosWebController(
	service *services.ProyectoWebService,
) *ProyectosWebController {
	return &ProyectosWebController{
		Service: service,
	}
}

/*
===========================================================
CREAR PROYECTO
===========================================================
*/

func (ctrl *ProyectosWebController) Crear(c *gin.Context) {

	var proyecto services.ProyectoWeb

	contentType := c.GetHeader("Content-Type")

	/*
		=======================================================
		JSON
		=======================================================
	*/

	if strings.HasPrefix(
		contentType,
		"application/json",
	) {

		if err := c.ShouldBindJSON(&proyecto); err != nil {

			c.JSON(
				http.StatusBadRequest,
				gin.H{
					"success": false,
					"message": "datos JSON inválidos",
					"error":   err.Error(),
				},
			)

			return
		}

		/*
			===================================================
			IMAGEN COMO URL
			===================================================
		*/

		proyecto.Imagen =
			strings.TrimSpace(
				proyecto.Imagen,
			)

		if proyecto.Imagen == "" {

			c.JSON(
				http.StatusBadRequest,
				gin.H{
					"success": false,
					"message": "la imagen es obligatoria",
				},
			)

			return
		}

		if !esURLValida(
			proyecto.Imagen,
		) {

			c.JSON(
				http.StatusBadRequest,
				gin.H{
					"success": false,
					"message": "la imagen debe ser una URL válida con http o https",
				},
			)

			return
		}
	}

	/*
		=======================================================
		MULTIPART / FORM-DATA
		=======================================================
	*/

	if strings.HasPrefix(
		contentType,
		"multipart/form-data",
	) {

		proyecto.Codigo =
			strings.TrimSpace(
				c.PostForm("codigo"),
			)

		proyecto.Titulo =
			strings.TrimSpace(
				c.PostForm("titulo"),
			)

		proyecto.Slug =
			strings.TrimSpace(
				c.PostForm("slug"),
			)

		proyecto.Tipo =
			strings.TrimSpace(
				c.PostForm("tipo"),
			)

		proyecto.Ciudad =
			strings.TrimSpace(
				c.PostForm("ciudad"),
			)

		proyecto.Direccion =
			strings.TrimSpace(
				c.PostForm("direccion"),
			)

		proyecto.Etapa =
			strings.TrimSpace(
				c.PostForm("etapa"),
			)

		proyecto.Estado =
			strings.TrimSpace(
				c.PostForm("estado"),
			)

		proyecto.Dormitorios =
			stringPtr(
				c.PostForm("dormitorios"),
			)

		proyecto.MetrajeDesde =
			floatPtr(
				c.PostForm("metraje_desde"),
			)

		proyecto.MetrajeHasta =
			floatPtr(
				c.PostForm("metraje_hasta"),
			)

		proyecto.PrecioDesde =
			floatPtr(
				c.PostForm("precio_desde"),
			)

		proyecto.Activo =
			boolValue(
				c.PostForm("activo"),
				true,
			)

		proyecto.Orden =
			intValue(
				c.PostForm("orden"),
				0,
			)

		/*
			===================================================
			IMAGEN
			===================================================
		*/

		archivo, archivoErr :=
			c.FormFile("imagen")

		imagenURL :=
			strings.TrimSpace(
				c.PostForm("imagen_url"),
			)

		/*
			===================================================
			NO PERMITIR ARCHIVO + URL
			===================================================
		*/

		if archivoErr == nil &&
			archivo != nil &&
			imagenURL != "" {

			c.JSON(
				http.StatusBadRequest,
				gin.H{
					"success": false,
					"message": "envía una imagen como archivo o como URL, no ambas",
				},
			)

			return
		}

		/*
			===================================================
			IMAGEN COMO ARCHIVO
			===================================================
		*/

		if archivoErr == nil &&
			archivo != nil {

			imagePath, err :=
				guardarImagen(
					archivo,
				)

			if err != nil {

				c.JSON(
					http.StatusBadRequest,
					gin.H{
						"success": false,
						"message": err.Error(),
					},
				)

				return
			}

			proyecto.Imagen =
				imagePath

		} else if imagenURL != "" {

			/*
				================================================
				IMAGEN COMO URL
				================================================
			*/

			if !esURLValida(
				imagenURL,
			) {

				c.JSON(
					http.StatusBadRequest,
					gin.H{
						"success": false,
						"message": "la imagen_url debe ser una URL válida con http o https",
					},
				)

				return
			}

			proyecto.Imagen =
				imagenURL

		} else {

			c.JSON(
				http.StatusBadRequest,
				gin.H{
					"success": false,
					"message": "la imagen es obligatoria. Envía un archivo en 'imagen' o una URL en 'imagen_url'",
				},
			)

			return
		}
	}

	/*
		=======================================================
		CONTENT TYPE
		=======================================================
	*/

	if !strings.HasPrefix(
		contentType,
		"application/json",
	) &&
		!strings.HasPrefix(
			contentType,
			"multipart/form-data",
		) {

		c.JSON(
			http.StatusBadRequest,
			gin.H{
				"success": false,
				"message": "Content-Type no soportado. Usa application/json o multipart/form-data",
			},
		)

		return
	}

	/*
		=======================================================
		VALIDACIÓN FINAL
		=======================================================
	*/

	if strings.TrimSpace(
		proyecto.Imagen,
	) == "" {

		c.JSON(
			http.StatusBadRequest,
			gin.H{
				"success": false,
				"message": "la imagen es obligatoria",
			},
		)

		return
	}

	/*
		=======================================================
		GUARDAR EN POSTGRESQL
		=======================================================
	*/

	err := ctrl.Service.Crear(
		c.Request.Context(),
		&proyecto,
	)

	if err != nil {

		/*
			Si se subió una imagen local
			y PostgreSQL falló, eliminarla.
		*/

		eliminarImagen(
			proyecto.Imagen,
		)

		fmt.Println(
			"ERROR CREAR PROYECTO WEB:",
			err,
		)

		c.JSON(
			http.StatusBadRequest,
			gin.H{
				"success": false,
				"message": err.Error(),
			},
		)

		return
	}

	/*
		=======================================================
		RESPUESTA
		=======================================================
	*/

	c.JSON(
		http.StatusCreated,
		gin.H{
			"success": true,
			"message": "Proyecto creado correctamente",
			"data":    proyecto,
		},
	)
}

/*
===========================================================
LISTAR PROYECTOS
===========================================================
*/

func (ctrl *ProyectosWebController) Listar(
	c *gin.Context,
) {

	page :=
		intValue(
			c.Query("page"),
			1,
		)

	if page < 1 {
		page = 1
	}

	limit :=
		intValue(
			c.Query("limit"),
			10,
		)

	if limit < 1 {
		limit = 10
	}

	if limit > 100 {
		limit = 100
	}

	activo, err :=
		boolPtr(
			c.Query("activo"),
		)

	if err != nil {

		c.JSON(
			http.StatusBadRequest,
			gin.H{
				"success": false,
				"message": "activo debe ser true o false",
			},
		)

		return
	}

	filtros :=
		services.FiltrosProyectoWeb{

			Buscar: strings.TrimSpace(
				c.Query("buscar"),
			),

			Estado: strings.TrimSpace(
				c.Query("estado"),
			),

			Tipo: strings.TrimSpace(
				c.Query("tipo"),
			),

			Ciudad: strings.TrimSpace(
				c.Query("ciudad"),
			),

			Etapa: strings.TrimSpace(
				c.Query("etapa"),
			),

			Activo: activo,

			Limit: limit,

			Offset: (page - 1) * limit,
		}

	proyectos, total, err :=
		ctrl.Service.Listar(
			c.Request.Context(),
			filtros,
		)

	if err != nil {

		fmt.Println(
			"ERROR LISTAR PROYECTOS WEB:",
			err,
		)

		c.JSON(
			http.StatusInternalServerError,
			gin.H{
				"success": false,
				"message": "no se pudieron obtener los proyectos",
				"error":   err.Error(),
			},
		)

		return
	}

	totalPaginas := 0

	if limit > 0 {
		totalPaginas =
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
				"total_pages": totalPaginas,
			},
		},
	)
}

/*
===========================================================
OBTENER POR ID
===========================================================
*/

func (ctrl *ProyectosWebController) Obtener(
	c *gin.Context,
) {

	id :=
		strings.TrimSpace(
			c.Param("id"),
		)

	if id == "" {

		c.JSON(
			http.StatusBadRequest,
			gin.H{
				"success": false,
				"message": "el id es obligatorio",
			},
		)

		return
	}

	proyecto, err :=
		ctrl.Service.ObtenerPorID(
			c.Request.Context(),
			id,
		)

	if err != nil {

		if errors.Is(
			err,
			pgx.ErrNoRows,
		) {

			c.JSON(
				http.StatusNotFound,
				gin.H{
					"success": false,
					"message": "proyecto no encontrado",
				},
			)

			return
		}

		fmt.Println(
			"ERROR OBTENER PROYECTO WEB:",
			err,
		)

		c.JSON(
			http.StatusInternalServerError,
			gin.H{
				"success": false,
				"message": "error al obtener el proyecto",
				"error":   err.Error(),
			},
		)

		return
	}

	c.JSON(
		http.StatusOK,
		gin.H{
			"success": true,
			"data":    proyecto,
		},
	)
}

/*
===========================================================
OBTENER POR SLUG
===========================================================
*/

func (ctrl *ProyectosWebController) ObtenerPorSlug(
	c *gin.Context,
) {

	slug :=
		strings.TrimSpace(
			c.Param("slug"),
		)

	if slug == "" {

		c.JSON(
			http.StatusBadRequest,
			gin.H{
				"success": false,
				"message": "el slug es obligatorio",
			},
		)

		return
	}

	proyecto, err :=
		ctrl.Service.ObtenerPorSlug(
			c.Request.Context(),
			slug,
		)

	if err != nil {

		if errors.Is(
			err,
			pgx.ErrNoRows,
		) {

			c.JSON(
				http.StatusNotFound,
				gin.H{
					"success": false,
					"message": "proyecto no encontrado",
				},
			)

			return
		}

		fmt.Println(
			"ERROR OBTENER PROYECTO POR SLUG:",
			err,
		)

		c.JSON(
			http.StatusInternalServerError,
			gin.H{
				"success": false,
				"message": "error al obtener el proyecto",
				"error":   err.Error(),
			},
		)

		return
	}

	if proyecto == nil {

		c.JSON(
			http.StatusNotFound,
			gin.H{
				"success": false,
				"message": "proyecto no encontrado",
			},
		)

		return
	}

	c.JSON(
		http.StatusOK,
		gin.H{
			"success": true,
			"data":    proyecto,
		},
	)
}

/*
===========================================================
ACTUALIZAR PROYECTO
===========================================================
*/

func (ctrl *ProyectosWebController) Actualizar(
	c *gin.Context,
) {

	id :=
		strings.TrimSpace(
			c.Param("id"),
		)

	if id == "" {

		c.JSON(
			http.StatusBadRequest,
			gin.H{
				"success": false,
				"message": "el id es obligatorio",
			},
		)

		return
	}

	/*
		=======================================================
		OBTENER PROYECTO ACTUAL
		=======================================================
	*/

	proyecto, err :=
		ctrl.Service.ObtenerPorID(
			c.Request.Context(),
			id,
		)

	if err != nil {

		if errors.Is(
			err,
			pgx.ErrNoRows,
		) {

			c.JSON(
				http.StatusNotFound,
				gin.H{
					"success": false,
					"message": "proyecto no encontrado",
				},
			)

			return
		}

		c.JSON(
			http.StatusInternalServerError,
			gin.H{
				"success": false,
				"message": "error al obtener el proyecto",
				"error":   err.Error(),
			},
		)

		return
	}

	imagenAnterior :=
		proyecto.Imagen

	contentType :=
		c.GetHeader("Content-Type")

	/*
		=======================================================
		JSON
		=======================================================
	*/

	if strings.HasPrefix(
		contentType,
		"application/json",
	) {

		var datos services.ProyectoWeb

		if err :=
			c.ShouldBindJSON(
				&datos,
			); err != nil {

			c.JSON(
				http.StatusBadRequest,
				gin.H{
					"success": false,
					"message": "datos JSON inválidos",
					"error":   err.Error(),
				},
			)

			return
		}

		/*
			===================================================
			CAMPOS
			===================================================
		*/

		proyecto.Codigo =
			strings.TrimSpace(
				datos.Codigo,
			)

		proyecto.Titulo =
			strings.TrimSpace(
				datos.Titulo,
			)

		proyecto.Slug =
			strings.TrimSpace(
				datos.Slug,
			)

		proyecto.Tipo =
			strings.TrimSpace(
				datos.Tipo,
			)

		proyecto.Ciudad =
			strings.TrimSpace(
				datos.Ciudad,
			)

		proyecto.Direccion =
			strings.TrimSpace(
				datos.Direccion,
			)

		proyecto.Etapa =
			strings.TrimSpace(
				datos.Etapa,
			)

		proyecto.Dormitorios =
			datos.Dormitorios

		proyecto.MetrajeDesde =
			datos.MetrajeDesde

		proyecto.MetrajeHasta =
			datos.MetrajeHasta

		proyecto.Estado =
			datos.Estado

		proyecto.PrecioDesde =
			datos.PrecioDesde

		proyecto.Activo =
			datos.Activo

		proyecto.Orden =
			datos.Orden

		/*
			===================================================
			IMAGEN URL
			===================================================
		*/

		imagenNueva :=
			strings.TrimSpace(
				datos.Imagen,
			)

		/*
			Si viene vacía,
			conservar imagen anterior.
		*/

		if imagenNueva != "" {

			if !esURLValida(
				imagenNueva,
			) {

				c.JSON(
					http.StatusBadRequest,
					gin.H{
						"success": false,
						"message": "la imagen debe ser una URL válida con http o https",
					},
				)

				return
			}

			proyecto.Imagen =
				imagenNueva
		}
	}

	/*
		=======================================================
		MULTIPART / FORM-DATA
		=======================================================
	*/

	if strings.HasPrefix(
		contentType,
		"multipart/form-data",
	) {

		proyecto.Codigo =
			strings.TrimSpace(
				c.PostForm("codigo"),
			)

		proyecto.Titulo =
			strings.TrimSpace(
				c.PostForm("titulo"),
			)

		proyecto.Slug =
			strings.TrimSpace(
				c.PostForm("slug"),
			)

		proyecto.Tipo =
			strings.TrimSpace(
				c.PostForm("tipo"),
			)

		proyecto.Ciudad =
			strings.TrimSpace(
				c.PostForm("ciudad"),
			)

		proyecto.Direccion =
			strings.TrimSpace(
				c.PostForm("direccion"),
			)

		proyecto.Etapa =
			strings.TrimSpace(
				c.PostForm("etapa"),
			)

		proyecto.Estado =
			strings.TrimSpace(
				c.PostForm("estado"),
			)

		proyecto.Dormitorios =
			stringPtr(
				c.PostForm("dormitorios"),
			)

		proyecto.MetrajeDesde =
			floatPtr(
				c.PostForm("metraje_desde"),
			)

		proyecto.MetrajeHasta =
			floatPtr(
				c.PostForm("metraje_hasta"),
			)

		proyecto.PrecioDesde =
			floatPtr(
				c.PostForm("precio_desde"),
			)

		proyecto.Activo =
			boolValue(
				c.PostForm("activo"),
				proyecto.Activo,
			)

		proyecto.Orden =
			intValue(
				c.PostForm("orden"),
				proyecto.Orden,
			)

		/*
			===================================================
			IMAGEN
			===================================================
		*/

		archivo, archivoErr :=
			c.FormFile("imagen")

		imagenURL :=
			strings.TrimSpace(
				c.PostForm("imagen_url"),
			)

		/*
			===================================================
			NO PERMITIR AMBOS
			===================================================
		*/

		if archivoErr == nil &&
			archivo != nil &&
			imagenURL != "" {

			c.JSON(
				http.StatusBadRequest,
				gin.H{
					"success": false,
					"message": "envía una imagen como archivo o como URL, no ambas",
				},
			)

			return
		}

		/*
			===================================================
			NUEVO ARCHIVO
			===================================================
		*/

		if archivoErr == nil &&
			archivo != nil {

			nuevaImagen, err :=
				guardarImagen(
					archivo,
				)

			if err != nil {

				c.JSON(
					http.StatusBadRequest,
					gin.H{
						"success": false,
						"message": err.Error(),
					},
				)

				return
			}

			proyecto.Imagen =
				nuevaImagen

		} else if imagenURL != "" {

			/*
				================================================
				NUEVA URL
				================================================
			*/

			if !esURLValida(
				imagenURL,
			) {

				c.JSON(
					http.StatusBadRequest,
					gin.H{
						"success": false,
						"message": "la imagen_url debe ser una URL válida con http o https",
					},
				)

				return
			}

			proyecto.Imagen =
				imagenURL
		}

		/*
			Si no se envía imagen,
			se conserva la anterior.
		*/
	}

	/*
		=======================================================
		VALIDAR CONTENT TYPE
		=======================================================
	*/

	if !strings.HasPrefix(
		contentType,
		"application/json",
	) &&
		!strings.HasPrefix(
			contentType,
			"multipart/form-data",
		) {

		c.JSON(
			http.StatusBadRequest,
			gin.H{
				"success": false,
				"message": "Content-Type no soportado. Usa application/json o multipart/form-data",
			},
		)

		return
	}

	/*
		=======================================================
		ACTUALIZAR POSTGRESQL
		=======================================================
	*/

	err =
		ctrl.Service.Actualizar(
			c.Request.Context(),
			proyecto,
		)

	if err != nil {

		/*
			Si la nueva imagen era local,
			eliminarla porque DB falló.
		*/

		if proyecto.Imagen != imagenAnterior {

			eliminarImagen(
				proyecto.Imagen,
			)
		}

		fmt.Println(
			"ERROR ACTUALIZAR PROYECTO WEB:",
			err,
		)

		c.JSON(
			http.StatusBadRequest,
			gin.H{
				"success": false,
				"message": err.Error(),
			},
		)

		return
	}

	/*
		=======================================================
		ELIMINAR IMAGEN LOCAL ANTERIOR
		=======================================================
	*/

	if proyecto.Imagen != imagenAnterior {

		eliminarImagen(
			imagenAnterior,
		)
	}

	c.JSON(
		http.StatusOK,
		gin.H{
			"success": true,
			"message": "Proyecto actualizado correctamente",
			"data":    proyecto,
		},
	)
}

/*
===========================================================
ELIMINAR PROYECTO
===========================================================
*/

func (ctrl *ProyectosWebController) Eliminar(
	c *gin.Context,
) {

	id :=
		strings.TrimSpace(
			c.Param("id"),
		)

	if id == "" {

		c.JSON(
			http.StatusBadRequest,
			gin.H{
				"success": false,
				"message": "el id es obligatorio",
			},
		)

		return
	}

	/*
		=======================================================
		OBTENER PROYECTO
		=======================================================
	*/

	proyecto, err :=
		ctrl.Service.ObtenerPorID(
			c.Request.Context(),
			id,
		)

	if err != nil {

		if errors.Is(
			err,
			pgx.ErrNoRows,
		) {

			c.JSON(
				http.StatusNotFound,
				gin.H{
					"success": false,
					"message": "proyecto no encontrado",
				},
			)

			return
		}

		c.JSON(
			http.StatusInternalServerError,
			gin.H{
				"success": false,
				"message": "error al obtener el proyecto",
				"error":   err.Error(),
			},
		)

		return
	}

	/*
		=======================================================
		ELIMINAR DB
		=======================================================
	*/

	err =
		ctrl.Service.Eliminar(
			c.Request.Context(),
			id,
		)

	if err != nil {

		c.JSON(
			http.StatusInternalServerError,
			gin.H{
				"success": false,
				"message": "no se pudo eliminar el proyecto",
				"error":   err.Error(),
			},
		)

		return
	}

	/*
		=======================================================
		ELIMINAR IMAGEN LOCAL
		=======================================================
	*/

	eliminarImagen(
		proyecto.Imagen,
	)

	c.JSON(
		http.StatusOK,
		gin.H{
			"success": true,
			"message": "Proyecto eliminado correctamente",
		},
	)
}

/*
===========================================================
GUARDAR IMAGEN
===========================================================
*/

func guardarImagen(
	fileHeader *multipart.FileHeader,
) (string, error) {

	if fileHeader == nil {
		return "",
			errors.New(
				"no se recibió ninguna imagen",
			)
	}

	/*
		=======================================================
		TAMAÑO
		=======================================================
	*/

	const maxSize = 5 * 1024 * 1024

	if fileHeader.Size > maxSize {

		return "",
			errors.New(
				"la imagen no puede superar los 5 MB",
			)
	}

	/*
		=======================================================
		EXTENSIÓN
		=======================================================
	*/

	extension :=
		strings.ToLower(
			filepath.Ext(
				fileHeader.Filename,
			),
		)

	switch extension {

	case ".jpg":
	case ".jpeg":
	case ".png":
	case ".webp":

	default:

		return "",
			errors.New(
				"solo se permiten imágenes JPG, JPEG, PNG o WEBP",
			)
	}

	/*
		=======================================================
		DIRECTORIO
		=======================================================
	*/

	directorio :=
		"./uploads/proyectos-web"

	if err :=
		os.MkdirAll(
			directorio,
			0755,
		); err != nil {

		return "",
			fmt.Errorf(
				"no se pudo crear el directorio de imágenes: %w",
				err,
			)
	}

	/*
		=======================================================
		NOMBRE ÚNICO
		=======================================================
	*/

	nombre :=
		uuid.New().String() +
			extension

	ruta :=
		filepath.Join(
			directorio,
			nombre,
		)

	/*
		=======================================================
		ABRIR
		=======================================================
	*/

	src, err :=
		fileHeader.Open()

	if err != nil {

		return "",
			fmt.Errorf(
				"no se pudo abrir la imagen: %w",
				err,
			)
	}

	defer src.Close()

	/*
		=======================================================
		CREAR
		=======================================================
	*/

	dst, err :=
		os.Create(ruta)

	if err != nil {

		return "",
			fmt.Errorf(
				"no se pudo crear la imagen: %w",
				err,
			)
	}

	/*
		=======================================================
		COPIAR
		=======================================================
	*/

	if _, err :=
		io.Copy(
			dst,
			src,
		); err != nil {

		dst.Close()

		os.Remove(ruta)

		return "",
			fmt.Errorf(
				"no se pudo guardar la imagen: %w",
				err,
			)
	}

	if err :=
		dst.Close(); err != nil {

		os.Remove(ruta)

		return "",
			fmt.Errorf(
				"no se pudo cerrar la imagen: %w",
				err,
			)
	}

	/*
		=======================================================
		IMPORTANTE
		=======================================================

		La BD solamente guarda esta ruta:

		/uploads/proyectos-web/archivo.webp

		NO guardar localhost.
		NO guardar Railway.
		=======================================================
	*/

	return "/uploads/proyectos-web/" +
		nombre, nil
}

/*
===========================================================
ELIMINAR IMAGEN
===========================================================
*/

func eliminarImagen(
	imagePath string,
) {

	if imagePath == "" {
		return
	}

	/*
		=======================================================
		SOLO ARCHIVOS PROPIOS
		=======================================================
	*/

	if !strings.HasPrefix(
		imagePath,
		"/uploads/proyectos-web/",
	) {
		return
	}

	/*
		=======================================================
		EVITAR PATH TRAVERSAL
		=======================================================
	*/

	cleanPath :=
		filepath.Clean(
			imagePath,
		)

	if !strings.HasPrefix(
		cleanPath,
		"/uploads/proyectos-web/",
	) {
		return
	}

	ruta :=
		"." + cleanPath

	_ = os.Remove(
		ruta,
	)
}

/*
===========================================================
VALIDAR URL
===========================================================
*/

func esURLValida(
	value string,
) bool {

	value =
		strings.TrimSpace(
			value,
		)

	if value == "" {
		return false
	}

	u, err :=
		url.ParseRequestURI(
			value,
		)

	if err != nil {
		return false
	}

	if u.Scheme != "http" &&
		u.Scheme != "https" {

		return false
	}

	if u.Host == "" {
		return false
	}

	return true
}

/*
===========================================================
STRING POINTER
===========================================================
*/

func stringPtr(
	value string,
) *string {

	value =
		strings.TrimSpace(
			value,
		)

	if value == "" {
		return nil
	}

	return &value
}

/*
===========================================================
FLOAT POINTER
===========================================================
*/

func floatPtr(
	value string,
) *float64 {

	value =
		strings.TrimSpace(
			value,
		)

	if value == "" {
		return nil
	}

	number, err :=
		strconv.ParseFloat(
			value,
			64,
		)

	if err != nil {
		return nil
	}

	return &number
}

/*
===========================================================
INT
===========================================================
*/

func intValue(
	value string,
	fallback int,
) int {

	value =
		strings.TrimSpace(
			value,
		)

	if value == "" {
		return fallback
	}

	number, err :=
		strconv.Atoi(
			value,
		)

	if err != nil {
		return fallback
	}

	return number
}

/*
===========================================================
BOOL
===========================================================
*/

func boolValue(
	value string,
	fallback bool,
) bool {

	value =
		strings.TrimSpace(
			value,
		)

	if value == "" {
		return fallback
	}

	result, err :=
		strconv.ParseBool(
			value,
		)

	if err != nil {
		return fallback
	}

	return result
}

/*
===========================================================
BOOL POINTER
===========================================================
*/

func boolPtr(
	value string,
) (*bool, error) {

	value =
		strings.TrimSpace(
			value,
		)

	if value == "" {
		return nil, nil
	}

	result, err :=
		strconv.ParseBool(
			value,
		)

	if err != nil {
		return nil, err
	}

	return &result, nil
}
