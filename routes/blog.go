package routes

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

/* =========================================================
   MODELOS
========================================================= */

type BlogPost struct {
	ID            string          `json:"id"`
	Title         string          `json:"title"`
	Slug          string          `json:"slug"`
	Category      string          `json:"category"`
	Excerpt       string          `json:"excerpt"`
	CoverImageURL *string         `json:"cover_image_url"`
	Content       json.RawMessage `json:"content"`
	Status        string          `json:"status"`
	AuthorName    *string         `json:"author_name"`
	CreatedAt     time.Time       `json:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at"`
	PublishedAt   *time.Time      `json:"published_at"`
}

type CreateBlogRequest struct {
	Title         string          `json:"title" binding:"required"`
	Slug          string          `json:"slug" binding:"required"`
	Category      string          `json:"category" binding:"required"`
	Excerpt       string          `json:"excerpt"`
	CoverImageURL *string         `json:"cover_image_url"`
	Content       json.RawMessage `json:"content" binding:"required"`
	Status        string          `json:"status"`
	AuthorName    string          `json:"author_name"`
}

type UpdateBlogRequest struct {
	Title         string          `json:"title"`
	Slug          string          `json:"slug"`
	Category      string          `json:"category"`
	Excerpt       string          `json:"excerpt"`
	CoverImageURL *string         `json:"cover_image_url"`
	Content       json.RawMessage `json:"content"`
	Status        string          `json:"status"`
	AuthorName    string          `json:"author_name"`
}

/* =========================================================
   RUTAS DEL BLOG
========================================================= */

func RutasBlog(
	api *gin.RouterGroup,
	db *pgxpool.Pool,
) {

	blog := api.Group("/blog")

	// =====================================================
	// IMÁGENES
	// =====================================================

	// Subir una imagen
	// POST /api/blog/imagen
	blog.POST(
		"/imagen",
		SubirImagenBlog(db),
	)

	// Mostrar una imagen
	// GET /api/blog/imagen/:id
	blog.GET(
		"/imagen/:id",
		VerImagenBlog(db),
	)

	// =====================================================
	// ADMINISTRACIÓN
	// =====================================================

	// Obtener artículo por ID para editar
	// GET /api/blog/admin/:id
	blog.GET(
		"/admin/:id",
		ObtenerBlogPorID(db),
	)

	// =====================================================
	// CREAR
	// =====================================================

	// Crear artículo
	// POST /api/blog
	blog.POST(
		"",
		CrearBlog(db),
	)

	// =====================================================
	// LISTAR
	// =====================================================

	// Listar todos los artículos
	// GET /api/blog
	//
	// También permite:
	// GET /api/blog?status=publicado
	// GET /api/blog?status=borrador
	blog.GET(
		"",
		ListarBlog(db),
	)

	// =====================================================
	// ACTUALIZAR
	// =====================================================

	// Actualizar artículo
	// PATCH /api/blog/:id
	blog.PATCH(
		"/:id",
		ActualizarBlog(db),
	)

	// =====================================================
	// ELIMINAR
	// =====================================================

	// Eliminar artículo
	// DELETE /api/blog/:id
	blog.DELETE(
		"/:id",
		EliminarBlog(db),
	)

	// =====================================================
	// WEB PÚBLICA
	// =====================================================

	// Obtener artículo publicado por slug
	// GET /api/blog/:slug
	//
	// ESTA DEBE SER LA ÚLTIMA RUTA
	// porque :slug es dinámica.
	blog.GET(
		"/:slug",
		ObtenerBlogPorSlug(db),
	)
}

/* =========================================================
   HELPERS
========================================================= */

func normalizeBlogStatus(status string) string {
	status = strings.ToLower(
		strings.TrimSpace(status),
	)

	if status == "publicado" {
		return "publicado"
	}

	return "borrador"
}

func normalizeSlug(slug string) string {
	return strings.Trim(
		strings.ToLower(
			slug,
		),
		" ",
	)
}

/* =========================================================
   CREAR BLOG
========================================================= */

func CrearBlog(
	db *pgxpool.Pool,
) gin.HandlerFunc {

	return func(c *gin.Context) {

		var req CreateBlogRequest

		if err :=
			c.ShouldBindJSON(&req); err != nil {

			c.JSON(
				http.StatusBadRequest,
				gin.H{
					"success": false,
					"message": "Datos del artículo inválidos.",
					"error":   err.Error(),
				},
			)

			return
		}

		req.Title =
			strings.TrimSpace(
				req.Title,
			)

		req.Slug =
			normalizeSlug(
				req.Slug,
			)

		req.Category =
			strings.TrimSpace(
				req.Category,
			)

		req.Excerpt =
			strings.TrimSpace(
				req.Excerpt,
			)

		req.AuthorName =
			strings.TrimSpace(
				req.AuthorName,
			)

		if req.Title == "" {

			c.JSON(
				http.StatusBadRequest,
				gin.H{
					"success": false,
					"message": "El título es obligatorio.",
				},
			)

			return
		}

		if req.Slug == "" {

			c.JSON(
				http.StatusBadRequest,
				gin.H{
					"success": false,
					"message": "El slug es obligatorio.",
				},
			)

			return
		}

		if req.Category == "" {

			c.JSON(
				http.StatusBadRequest,
				gin.H{
					"success": false,
					"message": "La categoría es obligatoria.",
				},
			)

			return
		}

		if len(req.Content) == 0 {

			req.Content =
				json.RawMessage(`[]`)
		}

		/*
			Verificamos que content sea
			JSON válido.
		*/

		var contentCheck interface{}

		if err :=
			json.Unmarshal(
				req.Content,
				&contentCheck,
			); err != nil {

			c.JSON(
				http.StatusBadRequest,
				gin.H{
					"success": false,
					"message": "El contenido del artículo no tiene un JSON válido.",
					"error":   err.Error(),
				},
			)

			return
		}

		status :=
			normalizeBlogStatus(
				req.Status,
			)

		if req.AuthorName == "" {
			req.AuthorName =
				"ANCOSUR Inmobiliaria"
		}

		ctx, cancel :=
			context.WithTimeout(
				c.Request.Context(),
				10*time.Second,
			)

		defer cancel()

		/*
			IMPORTANTE:
			El slug es único.
			No permitimos duplicados.
		*/

		var exists bool

		err :=
			db.QueryRow(
				ctx,
				`
				SELECT EXISTS(
					SELECT 1
					FROM blog_posts
					WHERE slug = $1
				)
				`,
				req.Slug,
			).Scan(&exists)

		if err != nil {

			c.JSON(
				http.StatusInternalServerError,
				gin.H{
					"success": false,
					"message": "No se pudo verificar el slug.",
					"error":   err.Error(),
				},
			)

			return
		}

		if exists {

			c.JSON(
				http.StatusConflict,
				gin.H{
					"success": false,
					"message": "El slug ya existe. Utiliza otro slug.",
				},
			)

			return
		}

		id :=
			uuid.New()

		var publishedAt *time.Time

		if status == "publicado" {

			now :=
				time.Now()

			publishedAt =
				&now
		}

		var post BlogPost

		err =
			db.QueryRow(
				ctx,
				`
				INSERT INTO blog_posts (
					id,
					title,
					slug,
					category,
					excerpt,
					cover_image_url,
					content,
					status,
					author_name,
					created_at,
					updated_at,
					published_at
				)
				VALUES (
					$1,
					$2,
					$3,
					$4,
					$5,
					$6,
					$7::jsonb,
					$8,
					$9,
					NOW(),
					NOW(),
					$10
				)
				RETURNING
					id,
					title,
					slug,
					category,
					excerpt,
					cover_image_url,
					content,
					status,
					author_name,
					created_at,
					updated_at,
					published_at
				`,
				id,
				req.Title,
				req.Slug,
				req.Category,
				req.Excerpt,
				req.CoverImageURL,
				req.Content,
				status,
				req.AuthorName,
				publishedAt,
			).
				Scan(
					&post.ID,
					&post.Title,
					&post.Slug,
					&post.Category,
					&post.Excerpt,
					&post.CoverImageURL,
					&post.Content,
					&post.Status,
					&post.AuthorName,
					&post.CreatedAt,
					&post.UpdatedAt,
					&post.PublishedAt,
				)

		if err != nil {

			c.JSON(
				http.StatusInternalServerError,
				gin.H{
					"success": false,
					"message": "No se pudo crear el artículo.",
					"error":   err.Error(),
				},
			)

			return
		}

		c.JSON(
			http.StatusCreated,
			gin.H{
				"success": true,
				"message": "Artículo creado correctamente.",
				"data":    post,
			},
		)
	}
}

/* =========================================================
   LISTAR BLOG
========================================================= */

func ListarBlog(
	db *pgxpool.Pool,
) gin.HandlerFunc {

	return func(c *gin.Context) {

		status :=
			strings.TrimSpace(
				c.Query("status"),
			)

		ctx, cancel :=
			context.WithTimeout(
				c.Request.Context(),
				10*time.Second,
			)

		defer cancel()

		query := `
			SELECT
				id,
				title,
				slug,
				category,
				excerpt,
				cover_image_url,
				content,
				status,
				author_name,
				created_at,
				updated_at,
				published_at
			FROM blog_posts
		`

		args :=
			[]interface{}{}

		if status != "" {

			query += `
				WHERE status = $1
			`

			args =
				append(
					args,
					status,
				)
		}

		query += `
			ORDER BY created_at DESC
		`

		rows, err :=
			db.Query(
				ctx,
				query,
				args...,
			)

		if err != nil {

			c.JSON(
				http.StatusInternalServerError,
				gin.H{
					"success": false,
					"message": "No se pudieron cargar los artículos.",
					"error":   err.Error(),
				},
			)

			return
		}

		defer rows.Close()

		posts :=
			make(
				[]BlogPost,
				0,
			)

		for rows.Next() {

			var post BlogPost

			err :=
				rows.Scan(
					&post.ID,
					&post.Title,
					&post.Slug,
					&post.Category,
					&post.Excerpt,
					&post.CoverImageURL,
					&post.Content,
					&post.Status,
					&post.AuthorName,
					&post.CreatedAt,
					&post.UpdatedAt,
					&post.PublishedAt,
				)

			if err != nil {

				c.JSON(
					http.StatusInternalServerError,
					gin.H{
						"success": false,
						"message": "Error leyendo los artículos.",
						"error":   err.Error(),
					},
				)

				return
			}

			posts =
				append(
					posts,
					post,
				)
		}

		if err :=
			rows.Err(); err != nil {

			c.JSON(
				http.StatusInternalServerError,
				gin.H{
					"success": false,
					"message": "Error procesando los artículos.",
					"error":   err.Error(),
				},
			)

			return
		}

		c.JSON(
			http.StatusOK,
			gin.H{
				"success": true,
				"total":   len(posts),
				"data":    posts,
			},
		)
	}
}

/* =========================================================
   OBTENER POR ID
========================================================= */

func ObtenerBlogPorID(
	db *pgxpool.Pool,
) gin.HandlerFunc {

	return func(c *gin.Context) {

		id :=
			c.Param("id")

		ctx, cancel :=
			context.WithTimeout(
				c.Request.Context(),
				10*time.Second,
			)

		defer cancel()

		var post BlogPost

		err :=
			db.QueryRow(
				ctx,
				`
				SELECT
					id,
					title,
					slug,
					category,
					excerpt,
					cover_image_url,
					content,
					status,
					author_name,
					created_at,
					updated_at,
					published_at
				FROM blog_posts
				WHERE id = $1
				`,
				id,
			).
				Scan(
					&post.ID,
					&post.Title,
					&post.Slug,
					&post.Category,
					&post.Excerpt,
					&post.CoverImageURL,
					&post.Content,
					&post.Status,
					&post.AuthorName,
					&post.CreatedAt,
					&post.UpdatedAt,
					&post.PublishedAt,
				)

		if err != nil {

			if err == pgx.ErrNoRows {

				c.JSON(
					http.StatusNotFound,
					gin.H{
						"success": false,
						"message": "Artículo no encontrado.",
					},
				)

				return
			}

			c.JSON(
				http.StatusInternalServerError,
				gin.H{
					"success": false,
					"message": "No se pudo obtener el artículo.",
					"error":   err.Error(),
				},
			)

			return
		}

		c.JSON(
			http.StatusOK,
			gin.H{
				"success": true,
				"data":    post,
			},
		)
	}
}

/* =========================================================
   OBTENER POR SLUG
========================================================= */

func ObtenerBlogPorSlug(
	db *pgxpool.Pool,
) gin.HandlerFunc {

	return func(c *gin.Context) {

		slug :=
			normalizeSlug(
				c.Param("slug"),
			)

		ctx, cancel :=
			context.WithTimeout(
				c.Request.Context(),
				10*time.Second,
			)

		defer cancel()

		var post BlogPost

		err :=
			db.QueryRow(
				ctx,
				`
				SELECT
					id,
					title,
					slug,
					category,
					excerpt,
					cover_image_url,
					content,
					status,
					author_name,
					created_at,
					updated_at,
					published_at
				FROM blog_posts
				WHERE slug = $1
				  AND status = 'publicado'
				`,
				slug,
			).
				Scan(
					&post.ID,
					&post.Title,
					&post.Slug,
					&post.Category,
					&post.Excerpt,
					&post.CoverImageURL,
					&post.Content,
					&post.Status,
					&post.AuthorName,
					&post.CreatedAt,
					&post.UpdatedAt,
					&post.PublishedAt,
				)

		if err != nil {

			if err == pgx.ErrNoRows {

				c.JSON(
					http.StatusNotFound,
					gin.H{
						"success": false,
						"message": "Artículo no encontrado.",
					},
				)

				return
			}

			c.JSON(
				http.StatusInternalServerError,
				gin.H{
					"success": false,
					"message": "No se pudo obtener el artículo.",
					"error":   err.Error(),
				},
			)

			return
		}

		c.JSON(
			http.StatusOK,
			gin.H{
				"success": true,
				"data":    post,
			},
		)
	}
}

/* =========================================================
   ACTUALIZAR
========================================================= */

func ActualizarBlog(
	db *pgxpool.Pool,
) gin.HandlerFunc {

	return func(c *gin.Context) {

		id :=
			c.Param("id")

		var req UpdateBlogRequest

		if err :=
			c.ShouldBindJSON(&req); err != nil {

			c.JSON(
				http.StatusBadRequest,
				gin.H{
					"success": false,
					"message": "Datos inválidos.",
					"error":   err.Error(),
				},
			)

			return
		}

		req.Title =
			strings.TrimSpace(
				req.Title,
			)

		req.Slug =
			normalizeSlug(
				req.Slug,
			)

		req.Category =
			strings.TrimSpace(
				req.Category,
			)

		req.Excerpt =
			strings.TrimSpace(
				req.Excerpt,
			)

		req.AuthorName =
			strings.TrimSpace(
				req.AuthorName,
			)

		status :=
			normalizeBlogStatus(
				req.Status,
			)

		if len(req.Content) == 0 {
			req.Content =
				json.RawMessage(`[]`)
		}

		var contentCheck interface{}

		if err :=
			json.Unmarshal(
				req.Content,
				&contentCheck,
			); err != nil {

			c.JSON(
				http.StatusBadRequest,
				gin.H{
					"success": false,
					"message": "El contenido no tiene JSON válido.",
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

		var publishedAt *time.Time

		/*
			Si se publica ahora y todavía
			no tenía fecha, colocamos NOW().
		*/

		if status == "publicado" {

			var currentPublished *time.Time

			err :=
				db.QueryRow(
					ctx,
					`
					SELECT published_at
					FROM blog_posts
					WHERE id = $1
					`,
					id,
				).
					Scan(
						&currentPublished,
					)

			if err == nil &&
				currentPublished != nil {

				publishedAt =
					currentPublished

			} else {

				now :=
					time.Now()

				publishedAt =
					&now
			}
		}

		var post BlogPost

		err :=
			db.QueryRow(
				ctx,
				`
				UPDATE blog_posts
				SET
					title = $1,
					slug = $2,
					category = $3,
					excerpt = $4,
					cover_image_url = $5,
					content = $6::jsonb,
					status = $7,
					author_name = $8,
					updated_at = NOW(),
					published_at = $9
				WHERE id = $10
				RETURNING
					id,
					title,
					slug,
					category,
					excerpt,
					cover_image_url,
					content,
					status,
					author_name,
					created_at,
					updated_at,
					published_at
				`,
				req.Title,
				req.Slug,
				req.Category,
				req.Excerpt,
				req.CoverImageURL,
				req.Content,
				status,
				req.AuthorName,
				publishedAt,
				id,
			).
				Scan(
					&post.ID,
					&post.Title,
					&post.Slug,
					&post.Category,
					&post.Excerpt,
					&post.CoverImageURL,
					&post.Content,
					&post.Status,
					&post.AuthorName,
					&post.CreatedAt,
					&post.UpdatedAt,
					&post.PublishedAt,
				)

		if err != nil {

			c.JSON(
				http.StatusInternalServerError,
				gin.H{
					"success": false,
					"message": "No se pudo actualizar el artículo.",
					"error":   err.Error(),
				},
			)

			return
		}

		c.JSON(
			http.StatusOK,
			gin.H{
				"success": true,
				"message": "Artículo actualizado correctamente.",
				"data":    post,
			},
		)
	}
}

/* =========================================================
   ELIMINAR
========================================================= */

func EliminarBlog(
	db *pgxpool.Pool,
) gin.HandlerFunc {

	return func(c *gin.Context) {

		id :=
			c.Param("id")

		ctx, cancel :=
			context.WithTimeout(
				c.Request.Context(),
				10*time.Second,
			)

		defer cancel()

		result, err :=
			db.Exec(
				ctx,
				`
				DELETE FROM blog_posts
				WHERE id = $1
				`,
				id,
			)

		if err != nil {

			c.JSON(
				http.StatusInternalServerError,
				gin.H{
					"success": false,
					"message": "No se pudo eliminar el artículo.",
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
					"message": "Artículo no encontrado.",
				},
			)

			return
		}

		c.JSON(
			http.StatusOK,
			gin.H{
				"success": true,
				"message": "Artículo eliminado correctamente.",
			},
		)
	}
}

/* =========================================================
   SUBIR IMAGEN
========================================================= */

func SubirImagenBlog(
	db *pgxpool.Pool,
) gin.HandlerFunc {

	return func(c *gin.Context) {

		fileHeader, err :=
			c.FormFile("image")

		if err != nil {

			c.JSON(
				http.StatusBadRequest,
				gin.H{
					"success": false,
					"message": "No se recibió ninguna imagen.",
				},
			)

			return
		}

		if fileHeader.Size >
			8*1024*1024 {

			c.JSON(
				http.StatusBadRequest,
				gin.H{
					"success": false,
					"message": "La imagen no puede superar los 8 MB.",
				},
			)

			return
		}

		contentType :=
			fileHeader.Header.Get(
				"Content-Type",
			)

		allowedTypes :=
			map[string]bool{
				"image/jpeg": true,
				"image/png":  true,
				"image/webp": true,
			}

		if !allowedTypes[contentType] {

			c.JSON(
				http.StatusBadRequest,
				gin.H{
					"success": false,
					"message": "Formato de imagen no permitido. Usa JPG, PNG o WEBP.",
				},
			)

			return
		}

		file, err :=
			fileHeader.Open()

		if err != nil {

			c.JSON(
				http.StatusInternalServerError,
				gin.H{
					"success": false,
					"message": "No se pudo abrir la imagen.",
				},
			)

			return
		}

		defer file.Close()

		data :=
			make(
				[]byte,
				fileHeader.Size,
			)

		_, err =
			file.Read(data)

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

		id :=
			uuid.New()

		filename :=
			fileHeader.Filename

		ctx, cancel :=
			context.WithTimeout(
				c.Request.Context(),
				20*time.Second,
			)

		defer cancel()

		_, err =
			db.Exec(
				ctx,
				`
				INSERT INTO blog_images (
					id,
					file_name,
					content_type,
					file_size,
					file_data,
					created_at
				)
				VALUES (
					$1,
					$2,
					$3,
					$4,
					$5,
					NOW()
				)
				`,
				id,
				filename,
				contentType,
				fileHeader.Size,
				data,
			)

		if err != nil {

			c.JSON(
				http.StatusInternalServerError,
				gin.H{
					"success": false,
					"message": "No se pudo guardar la imagen.",
					"error":   err.Error(),
				},
			)

			return
		}

		imageURL :=
			fmt.Sprintf(
				"/api/blog/imagen/%s",
				id.String(),
			)

		c.JSON(
			http.StatusCreated,
			gin.H{
				"success": true,
				"message": "Imagen subida correctamente.",
				"data": gin.H{
					"id":  id.String(),
					"url": imageURL,
				},
			},
		)
	}
}

/* =========================================================
   SERVIR IMAGEN
========================================================= */

func VerImagenBlog(
	db *pgxpool.Pool,
) gin.HandlerFunc {

	return func(c *gin.Context) {

		id :=
			c.Param("id")

		ctx, cancel :=
			context.WithTimeout(
				c.Request.Context(),
				10*time.Second,
			)

		defer cancel()

		var (
			contentType string
			data        []byte
		)

		err :=
			db.QueryRow(
				ctx,
				`
				SELECT
					content_type,
					file_data
				FROM blog_images
				WHERE id = $1
				`,
				id,
			).
				Scan(
					&contentType,
					&data,
				)

		if err != nil {

			if err == pgx.ErrNoRows {

				c.JSON(
					http.StatusNotFound,
					gin.H{
						"success": false,
						"message": "Imagen no encontrada.",
					},
				)

				return
			}

			c.JSON(
				http.StatusInternalServerError,
				gin.H{
					"success": false,
					"message": "No se pudo obtener la imagen.",
				},
			)

			return
		}

		c.Data(
			http.StatusOK,
			contentType,
			data,
		)
	}
}